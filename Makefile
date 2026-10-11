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
FYNE_TOOLS      ?= v1.7.3
FYNE            := $(BIN)/fyne
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_LINT   := $(BIN)/golangci-lint
VERSION         ?= 0.0.1
TEST_ENV        := KUBECONFIG=/dev/null

.PHONY: all build run run-demo test test-unit test-unit-race test-kind bench lint fmt tidy \
        kind-up kind-down kind-seed app clean help

all: lint test-unit-race build ## Lint, race tests, build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

build: ## Build bin/cogpit (cgo, Fyne) and bin/cogpit-cli (pure Go)
	$(GO) build -o $(BIN)/cogpit ./cmd/cogpit
	CGO_ENABLED=0 $(GO) build -o $(BIN)/cogpit-cli ./cmd/cogpit-cli

run: build ## Run Cogpit against the kind cluster (make kind-up first)
	$(BIN)/cogpit --kubeconfig $(KIND_KUBECONFIG) --context kind-$(KIND_CLUSTER)

run-demo: build ## Run Cogpit on the synthetic demo cluster (10,000 pods, 1,000 events/s)
	$(BIN)/cogpit --demo

test: test-unit-race ## All tests that need no cluster (race detector on); test-kind needs make kind-up

test-unit: ## Unit tests without the race detector
	$(TEST_ENV) $(GO) test ./...

test-unit-race: ## Unit tests with the race detector
	$(TEST_ENV) $(GO) test -race ./...

test-kind: ## Cluster tests against the kind cluster (make kind-up first)
	COGPIT_KIND_KUBECONFIG=$(KIND_KUBECONFIG) COGPIT_KIND_CONTEXT=kind-$(KIND_CLUSTER) \
		$(GO) test -race -tags kind ./...

bench: ## Benchmarks for the gate criteria S1 (snapshots under churn) and S2 (sort)
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

kind-up: ## Create the kind cluster (own kubeconfig file) and seed it from hack/kind
	@mkdir -p $(dir $(KIND_KUBECONFIG))
	@if kind get clusters 2>/dev/null | grep -qx '$(KIND_CLUSTER)'; then \
		echo "kind cluster $(KIND_CLUSTER) exists"; \
		kind export kubeconfig --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG); \
	else \
		kind create cluster --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG) \
			--config hack/kind/cluster.yaml --wait 120s; \
	fi
	$(MAKE) kind-seed
	@echo "kubeconfig: $(KIND_KUBECONFIG)"

kind-seed: ## Apply the seed manifests to the kind cluster
	KUBECONFIG=$(KIND_KUBECONFIG) kubectl --context kind-$(KIND_CLUSTER) apply -f hack/kind/seed/

kind-down: ## Delete the kind cluster and its kubeconfig file
	kind delete cluster --name $(KIND_CLUSTER) --kubeconfig $(KIND_KUBECONFIG)
	rm -f $(KIND_KUBECONFIG)

$(FYNE):
	GOBIN=$(BIN) $(GO) install fyne.io/tools/cmd/fyne@$(FYNE_TOOLS)

app: $(FYNE) ## Build an unsigned Cogpit.app with fyne package (bin/Cogpit.app)
	cd cmd/cogpit && $(FYNE) package --os darwin --release --app-version $(VERSION) \
		--name Cogpit --app-id io.guided-traffic.cogpit --icon $(CURDIR)/packaging/icon.png
	rm -rf $(BIN)/Cogpit.app && mv cmd/cogpit/Cogpit.app $(BIN)/Cogpit.app
	@echo "built $(BIN)/Cogpit.app"

clean: ## Remove build output
	rm -rf $(BIN) dist
