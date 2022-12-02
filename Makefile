# See https://tech.davis-hansson.com/p/make/
SHELL := bash
.DELETE_ON_ERROR:
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory
GO ?= go
BIN := .tmp/bin

.PHONY: help
help: ## Describe useful make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-30s %s\n", $$1, $$2}'

.PHONY: all
all: ## Build, test, and lint (default)
	$(MAKE) test
	$(MAKE) lint

.PHONY: test
test: build ## Run unit tests
	$(GO) test -vet=off -race -cover ./...

.PHONY: build
build: ## Build all packages
	$(GO) build ./...

.PHONY: lint
lint: $(BIN)/gofmt $(BIN)/staticcheck ## Lint Go
	test -z "$$($(BIN)/gofmt -s -l . | tee /dev/stderr)"
	$(GO) vet ./...
	$(BIN)/staticcheck ./...

.PHONY: lintfix
lintfix: $(BIN)/gofmt ## Automatically fix some lint errors
	$(BIN)/gofmt -s -w .

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	go get -u -t ./... && go mod tidy -v

.PHONY: clean
clean: ## Remove intermediate artifacts
	rm -rf .tmp

$(BIN)/gofmt:
	@mkdir -p $(@D)
	$(GO) build -o $(@) cmd/gofmt

$(BIN)/staticcheck:
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install honnef.co/go/tools/cmd/staticcheck@latest
