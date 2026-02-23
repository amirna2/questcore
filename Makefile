# Makefile for QuestCore
# Single source of truth for build, test, and lint checks.
# Used by both local development and GitHub Actions CI.

BINARY  := bin/questcore
GO      := go
GOFLAGS := -v
TIMEOUT := 120s

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint vet fmt-check fmt ci clean

## build: Compile the questcore binary with version info
build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/questcore

## test: Run all tests with race detection
test:
	$(GO) test $(GOFLAGS) -timeout $(TIMEOUT) -race ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## vet: Run go vet
vet:
	$(GO) vet ./...

## fmt-check: Check that all Go files are gofmt-formatted
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## fmt: Format all Go files in place
fmt:
	gofmt -w .

## ci: Run the full check pipeline (same as GitHub Actions)
ci: fmt-check vet lint build test

## clean: Remove build artifacts
clean:
	rm -rf bin/
