BIN      ?= archivist
PKG      := ./cmd/archivist
ARGS     ?=
DUMP       ?= docs/dump/decisions.md
GLOBALDUMP ?= docs/dump/global-decisions.md
VERSION  ?=
LDFLAGS  := $(if $(VERSION),-ldflags "-X github.com/bwireman/archivist/internal/version.Version=$(VERSION)")

.PHONY: all build test vet fmt tidy check install run index dump-docs refresh-docs clean help

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

run: ## Run archivist (e.g. make run ARGS='dump --help')
	go run $(LDFLAGS) $(PKG) $(ARGS)

index: build ## Incrementally index the repository (needs Ollama)
	./$(BIN) index --plain

dump-docs: build ## Dump indexed ADRs to docs/dump/
	./$(BIN) dump --type adr --adr-scope repo -o $(DUMP)
	./$(BIN) dump --type adr --adr-scope global -o $(GLOBALDUMP)

refresh-docs: index dump-docs ## Index, then dump ADRs into docs/dump/

clean: ## Remove the local binary
	rm -f $(BIN)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
