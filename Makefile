# Cogpit — local developer targets. CI is out of scope for the MVP.
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
FYNE_TOOLS      ?= v1.7.3
FYNE            := $(BIN)/fyne
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_LINT   := $(BIN)/golangci-lint
VERSION         ?= 0.0.1
APP_ID          := io.guided-traffic.cogpit
APP_OS          ?= $(shell $(GO) env GOOS)
TEST_ENV        := KUBECONFIG=/dev/null

.PHONY: all build run run-demo test test-unit test-unit-race test-kind bench lint fmt tidy \
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

test-unit-race: ## Unit tests with the race detector
	$(TEST_ENV) $(GO) test -race ./...

test-kind: ## Cluster tests against the kind cluster (make kind-up first)
	@test -s $(KIND_KUBECONFIG) || { echo "no kubeconfig at $(KIND_KUBECONFIG): run make kind-up first"; exit 1; }
	$(TEST_ENV) COGPIT_KIND_KUBECONFIG=$(KIND_KUBECONFIG) COGPIT_KIND_CONTEXT=$(KIND_CONTEXT) \
		$(GO) test -race -tags kind ./...

bench: ## Benchmarks: demo churn and build, and the gate criteria S1 (snapshots under churn) and S2 (sort)
	$(TEST_ENV) $(GO) test -run '^$$' -bench . -benchmem ./internal/engine/... ./internal/demo/... ./internal/ui/...

# golangci-lint built with the project's Go: v2.13.x cannot read Go 1.27.2 export data.
$(GOLANGCI_LINT):
	GOBIN=$(BIN) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT) ## golangci-lint: the team's linters plus depguard layering
	$(GOLANGCI_LINT) run ./...

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
