BIN      ?= archivist
PKG      := ./cmd/archivist
ARGS     ?=
VERSION  ?=
LDFLAGS  := $(if $(VERSION),-ldflags "-X github.com/bwireman/archivist/internal/version.Version=$(VERSION)")

.PHONY: all build test vet fmt tidy check install run index embed export refresh-archive clean help

all: build

build: ## Build the archivist binary
	go build $(LDFLAGS) -o $(BIN) $(PKG)

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
	go install $(LDFLAGS) $(PKG)

run: ## Run archivist (e.g. make run ARGS='search --help')
	go run $(LDFLAGS) $(PKG) $(ARGS)

index: build ## Index records and code map (no Ollama)
	./$(BIN) index

embed: build ## Drain embed queue (needs Ollama)
	./$(BIN) embed --worker --once

export: build ## Generate docs/archive/
	./$(BIN) export

refresh-archive: index embed export ## Index, embed, then export

clean: ## Remove the local binary
	rm -f $(BIN)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
