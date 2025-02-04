## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: fmt
fmt: gofumpt ## Run gofumt against code.
	go mod tidy -compat=1.23
	$(GOFUMPT) -w -extra .

.PHONY: vendor
vendor:
	go mod vendor

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

.PHONY: lint
lint:
	test -s $(GOLANGCI_LINT) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN)
	CGO_ENABLED=0 $(GOLANGCI_LINT) run --timeout 5m

GOFUMPT ?= $(LOCALBIN)/gofumpt

.PHONY: gofumpt
gofumpt: $(GOFUMPT) ## Download gofumpt locally if necessary.
$(GOFUMPT): $(LOCALBIN)
	test -s $(LOCALBIN)/gofumpt || GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@latest

.PHONY: build
build: ## Run go build against code.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -tags timetzdata -tags=nomsgpack -o $(LOCALBIN)/hetzner-dnsapi-proxy .

.PHONY: functest
functest:
	cd tests && go test -v -timeout 0 ./... -ginkgo.v -ginkgo.randomize-all
