# Cogpit — developer targets. CI (.github/workflows/build.yml) runs the same ones: lint,
# parity, test-unit-coverage and build on macOS; vuln, kind-up and test-kind on Linux
# (docs/developer/releasing.md#ci).
#
# Go: go.mod declares go 1.27.2; GOTOOLCHAIN=auto fetches it when the local Go is older.
# Kubeconfig safety: every target that talks to a cluster uses KIND_KUBECONFIG, a separate file;
# nothing here reads or writes ~/.kube/config. Tests run with KUBECONFIG=/dev/null so that code
# falling back to the default loading rules finds no cluster.

export GOTOOLCHAIN := auto

GO              ?= go
BIN             := $(CURDIR)/bin
CACHE           := $(CURDIR)/.cache
KIND_CLUSTER    ?= cogpit
KIND_KUBECONFIG ?= $(CACHE)/kind/kubeconfig
KIND_CONTEXT    := kind-$(KIND_CLUSTER)
KUBECTL         := KUBECONFIG=$(KIND_KUBECONFIG) kubectl --context $(KIND_CONTEXT)
# renovate: datasource=go depName=fyne.io/tools/cmd/fyne
FYNE_TOOLS      ?= v1.7.3
FYNE            := $(BIN)/fyne
# renovate: datasource=go depName=github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_LINT   := $(BIN)/golangci-lint
VERSION         ?= 0.0.1
APP_ID          := io.guided-traffic.cogpit
APP_OS          ?= $(shell $(GO) env GOOS)
TEST_ENV        := KUBECONFIG=/dev/null

.PHONY: all build run run-demo test test-unit test-unit-race test-kind bench lint parity fmt tidy \
        kind-up kind-down kind-seed app icon clean help

all: lint test-unit-race build ## Lint, race tests, build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

build: ## Build bin/cogpit (cgo, Fyne) and bin/cogpit-cli (pure Go)
	$(GO) build -o $(BIN)/cogpit ./cmd/cogpit
	CGO_ENABLED=0 $(GO) build -o $(BIN)/cogpit-cli ./cmd/cogpit-cli

run: build ## Run Cogpit against the kind cluster (make kind-up first)
	@test -s $(KIND_KUBECONFIG) || { echo "no kubeconfig at $(KIND_KUBECONFIG): run make kind-up first"; exit 1; }
	$(BIN)/cogpit --kubeconfig $(KIND_KUBECONFIG) --context $(KIND_CONTEXT)

run-demo: build ## Run Cogpit on the synthetic demo cluster (10,000 pods, 1,000 events/s)
	$(BIN)/cogpit --demo

test: test-unit-race ## All tests that need no cluster (race detector on); test-kind needs make kind-up

test-unit: ## Unit tests without the race detector
	$(TEST_ENV) $(GO) test ./...

test-unit-race: test-fork ## Unit tests with the race detector, the terminal widget's included
	$(TEST_ENV) $(GO) test -race ./...

# The patched terminal widget (third_party/fyne-terminal, D-14) is a module of its own, which
# go test ./... from the root leaves out: its tests and fuzz seeds run here. The tests that start a
# local shell race inside the widget's RunLocalShell, which Cogpit does not use (it connects the
# widget with RunWithConnection), so they run without the race detector.
FORK_DIR         := third_party/fyne-terminal
FORK_SHELL_TESTS := ^(TestExitCode01|TestRunLocalShell_MissingStartDir|TestTerminal_Close|TestTerminal_Close_BeforeOpen)$$

.PHONY: test-fork
test-fork: ## Tests and fuzz seeds of the patched terminal widget, a module of its own
	cd $(FORK_DIR) && $(TEST_ENV) $(GO) test -race -skip '$(FORK_SHELL_TESTS)' ./...
	cd $(FORK_DIR) && $(TEST_ENV) $(GO) test -run '$(FORK_SHELL_TESTS)' ./...

test-kind: ## Cluster tests against the kind cluster (make kind-up first)
	@test -s $(KIND_KUBECONFIG) || { echo "no kubeconfig at $(KIND_KUBECONFIG): run make kind-up first"; exit 1; }
	$(TEST_ENV) COGPIT_KIND_KUBECONFIG=$(KIND_KUBECONFIG) COGPIT_KIND_CONTEXT=$(KIND_CONTEXT) \
		$(GO) test -race -tags kind ./...

bench: ## Benchmarks: demo churn and build, and the gate criteria S1 (snapshots under churn) and S2 (sort)
	$(TEST_ENV) $(GO) test -run '^$$' -bench . -benchmem ./internal/engine/... ./internal/demo/... ./internal/ui/...

# Coverage and vulnerability targets that CI runs (docs/developer/releasing.md#ci).
# test-unit-coverage is test-unit-race with a profile: the same packages, the same race detector.
# The badge is the profile's total ("go tool cover -func"), coloured as in the team's other
# repositories: >= 90 brightgreen, >= 80 green, >= 70 yellow, >= 60 orange, else red.
COVERAGE_DIR    := $(CACHE)/coverage
COVERAGE_BADGE  := .github/badges/coverage.json
# renovate: datasource=go depName=golang.org/x/vuln/cmd/govulncheck
GOVULNCHECK_VERSION ?= v1.9.0
GOVULNCHECK     := $(BIN)/govulncheck

.PHONY: test-unit-coverage coverage-badge vuln

test-unit-coverage: test-fork ## Unit tests with the race detector and a coverage profile (.cache/coverage/unit.out)
	@mkdir -p $(COVERAGE_DIR)
	$(TEST_ENV) $(GO) test -race -covermode=atomic -coverprofile=$(COVERAGE_DIR)/unit.out ./...
	@$(GO) tool cover -func=$(COVERAGE_DIR)/unit.out | tail -n 1

coverage-badge: ## Write .github/badges/coverage.json from the profile of test-unit-coverage
	@test -s $(COVERAGE_DIR)/unit.out || { echo "no profile at $(COVERAGE_DIR)/unit.out: run make test-unit-coverage first"; exit 1; }
	@mkdir -p $(dir $(COVERAGE_BADGE))
	@$(GO) tool cover -func=$(COVERAGE_DIR)/unit.out | awk '$$1 == "total:" { \
		pct = $$NF; sub(/%$$/, "", pct); p = pct + 0; \
		color = p >= 90 ? "brightgreen" : p >= 80 ? "green" : p >= 70 ? "yellow" : p >= 60 ? "orange" : "red"; \
		printf "{\n  \"schemaVersion\": 1,\n  \"label\": \"coverage\",\n  \"message\": \"%s%%\",\n  \"color\": \"%s\"\n}\n", pct, color }' \
		> $(COVERAGE_BADGE)
	@cat $(COVERAGE_BADGE)

$(GOVULNCHECK):
	GOBIN=$(BIN) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

vuln: $(GOVULNCHECK) ## govulncheck: known vulnerabilities in the code paths Cogpit calls
	$(GOVULNCHECK) ./...

# golangci-lint built with the project's Go: v2.13.x cannot read Go 1.27.2 export data.
$(GOLANGCI_LINT):
	GOBIN=$(BIN) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT) ## golangci-lint: the team's linters plus depguard layering
	$(GOLANGCI_LINT) run ./...

parity: ## Check the parity checklists and the numbers the progress pages report (D-15); PARITY_FLAGS=-rows skips the numbers
	$(GO) run ./hack/parity $(PARITY_FLAGS)

fmt: $(GOLANGCI_LINT) ## Format with gofmt and goimports
	$(GOLANGCI_LINT) fmt ./...

tidy: ## go mod tidy
	$(GO) mod tidy

# The kind targets need docker and kind; kubectl talks only to KIND_KUBECONFIG.
kind-up: ## Create the kind cluster (own kubeconfig file), or reuse it, and seed it from hack/kind
	@command -v kind >/dev/null || { echo "kind is not installed: https://kind.sigs.k8s.io"; exit 1; }
	@mkdir -p $(dir $(KIND_KUBECONFIG))
	@if kind get clusters 2>/dev/null | grep -qx '$(KIND_CLUSTER)'; then \
		echo "kind cluster $(KIND_CLUSTER) exists"; \
		kind export kubeconfig --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG); \
	else \
		kind create cluster --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG) \
			--config hack/kind/cluster.yaml --wait 120s; \
	fi
	$(MAKE) kind-seed
	@echo "kubeconfig: $(KIND_KUBECONFIG) (context $(KIND_CONTEXT))"

kind-seed: ## Apply the seed manifests (namespace cogpit-demo) and wait for web and api
	$(KUBECTL) apply -f hack/kind/seed/
	@$(KUBECTL) -n cogpit-demo rollout status deployment/web --timeout=180s && \
		$(KUBECTL) -n cogpit-demo rollout status deployment/api --timeout=180s || \
		echo "the seed is still starting (image pulls); check: $(KUBECTL) get all -n cogpit-demo"

kind-down: ## Delete the kind cluster and its kubeconfig file
	kind delete cluster --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG)
	rm -f $(KIND_KUBECONFIG)

# The drain cluster: one control plane and two workers, for the tests that cordon and drain a
# node, which must never run against the shared single-node cluster above.
KIND_DRAIN_CLUSTER    ?= cogpit-drain
KIND_DRAIN_KUBECONFIG ?= $(CACHE)/kind-drain/kubeconfig
KIND_DRAIN_CONTEXT    := kind-$(KIND_DRAIN_CLUSTER)

.PHONY: kind-drain-up test-kind-drain kind-drain-down

kind-drain-up: ## Create the multi-node drain cluster (own kubeconfig file), or reuse it
	@command -v kind >/dev/null || { echo "kind is not installed: https://kind.sigs.k8s.io"; exit 1; }
	@mkdir -p $(dir $(KIND_DRAIN_KUBECONFIG))
	@if kind get clusters 2>/dev/null | grep -qx '$(KIND_DRAIN_CLUSTER)'; then \
		echo "kind cluster $(KIND_DRAIN_CLUSTER) exists"; \
		kind export kubeconfig --name $(KIND_DRAIN_CLUSTER) --kubeconfig $(KIND_DRAIN_KUBECONFIG); \
	else \
		kind create cluster --name $(KIND_DRAIN_CLUSTER) --kubeconfig $(KIND_DRAIN_KUBECONFIG) \
			--config hack/kind/drain-cluster.yaml --wait 120s; \
	fi
	@echo "kubeconfig: $(KIND_DRAIN_KUBECONFIG) (context $(KIND_DRAIN_CONTEXT))"

test-kind-drain: ## Cordon, uncordon and drain tests against the drain cluster (make kind-drain-up first)
	@test -s $(KIND_DRAIN_KUBECONFIG) || { echo "no kubeconfig at $(KIND_DRAIN_KUBECONFIG): run make kind-drain-up first"; exit 1; }
	$(TEST_ENV) COGPIT_KIND_DRAIN_KUBECONFIG=$(KIND_DRAIN_KUBECONFIG) COGPIT_KIND_DRAIN_CONTEXT=$(KIND_DRAIN_CONTEXT) \
		$(GO) test -race -tags kind -count=1 -v -timeout 15m -run 'TestKindDrain' ./internal/testkit/integration/

kind-drain-down: ## Delete the drain cluster and its kubeconfig file
	kind delete cluster --name $(KIND_DRAIN_CLUSTER) --kubeconfig $(KIND_DRAIN_KUBECONFIG)
	rm -f $(KIND_DRAIN_KUBECONFIG)

$(FYNE):
	GOBIN=$(BIN) $(GO) install fyne.io/tools/cmd/fyne@$(FYNE_TOOLS)

# fyne package runs in bin/, so the package lands there: bin/Cogpit.app on macOS, bin/Cogpit.tar.xz
# on Linux, bin/Cogpit.exe on Windows. It wraps the binary `make build` produced (--executable):
# letting fyne build it would inject a file importing fyne.io/fyne/v2/app, whose driver modules are
# only in go.sum once cmd/cogpit uses Fyne. A packaged binary therefore carries no injected Fyne
# metadata, so cmd/cogpit calls app.NewWithID("io.guided-traffic.cogpit") rather than app.New.
# Unsigned: macOS asks once on first open.
app: build $(FYNE) ## Package Cogpit for this OS with fyne package (macOS: bin/Cogpit.app)
	rm -rf $(BIN)/Cogpit.app
	cd $(BIN) && $(FYNE) package --os $(APP_OS) --src $(CURDIR)/cmd/cogpit --executable $(BIN)/cogpit --release \
		--name Cogpit --app-id $(APP_ID) --app-version $(VERSION) --icon $(CURDIR)/packaging/icon.png
	@# macOS: fyne package leaves the category empty (only fyne release sets one); the ad-hoc
	@# signature seals the bundle so codesign --verify passes. Then check what Finder needs.
	@if [ "$(APP_OS)" = darwin ]; then \
		plutil -replace LSApplicationCategoryType -string public.app-category.developer-tools $(BIN)/Cogpit.app/Contents/Info.plist && \
		codesign --force --sign - $(BIN)/Cogpit.app && \
		codesign --verify $(BIN)/Cogpit.app && \
		test -x $(BIN)/Cogpit.app/Contents/MacOS/cogpit && \
		test "$$(plutil -extract CFBundleIdentifier raw $(BIN)/Cogpit.app/Contents/Info.plist)" = "$(APP_ID)" && \
		echo "built $(BIN)/Cogpit.app"; \
	fi

icon: ## Redraw packaging/icon.png from hack/icon
	$(GO) run ./hack/icon -o packaging/icon.png

clean: ## Remove build output
	rm -rf $(BIN) dist
