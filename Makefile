LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Location to install dependencies into.
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

LDFLAGS := -w -s
ifdef VERSION
LDFLAGS += -X github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner.version=$(VERSION)
endif

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(GOLANGCI_LINT) || curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(LOCALBIN)

.PHONY: fmt
fmt: golangci-lint ## Run the golangci-lint formatters against the code.
	$(GOLANGCI_LINT) fmt

.PHONY: lint
lint: golangci-lint ## Run golangci-lint against the code.
	CGO_ENABLED=0 $(GOLANGCI_LINT) run --timeout 5m

.PHONY: test
test: ## Run unit tests against the code in pkg.
	cd pkg && go test -v -timeout 0 ./... -ginkgo.v -ginkgo.randomize-all

.PHONY: functest
functest: ## Run functional tests against the code.
	cd tests && go test -v -timeout 0 ./... -ginkgo.v -ginkgo.randomize-all

.PHONY: build
build: ## Build the hetzner-dnsapi-proxy binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(LOCALBIN)/hetzner-dnsapi-proxy .

.PHONY: vendor
vendor: ## Run go mod tidy and go mod vendor and vendor dependencies.
	go mod tidy
	go mod vendor
