BIN  ?= archivist
PKG  := ./cmd/archivist
ARGS ?=

.PHONY: all build test vet fmt tidy check install run clean help

all: build

build: ## Build the archivist binary
	go build -o $(BIN) $(PKG)

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go sources
	gofmt -w .

tidy: ## Sync go.mod / go.sum
	go mod tidy

check: fmt vet test ## Format, vet, and test

install: ## Install archivist to GOPATH/bin
	go install $(PKG)

run: ## Run archivist (e.g. make run ARGS='dump --help')
	go run $(PKG) $(ARGS)

clean: ## Remove the local binary
	rm -f $(BIN)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
