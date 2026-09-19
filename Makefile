BIN      ?= archivist
PKG      := ./cmd/archivist
ARGS     ?=
VERSION  ?=
COMMIT   ?=
LDFLAGS  := $(strip $(if $(VERSION),-X github.com/bwireman/archivist/internal/version.Version=$(VERSION)) $(if $(COMMIT),-X github.com/bwireman/archivist/internal/version.Commit=$(COMMIT)))
LDFLAGS  := $(if $(LDFLAGS),-ldflags "$(LDFLAGS)")

.PHONY: all build test vet fmt tidy check install run import index embed export refresh-archive clean help

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

index: build ## Index code map and git (no Ollama)
	./$(BIN) index

embed: build ## Drain embed queue (needs Ollama)
	./$(BIN) embed --worker --once

export: build ## Generate docs/archive/ (no-op unless records.write_docs)
	./$(BIN) export

import: build ## Upsert typed markdown into SQLite (no prune)
	./$(BIN) import

refresh-archive: import index embed export ## Import, index, embed, then export

clean: ## Remove the local binary
	rm -f $(BIN)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
