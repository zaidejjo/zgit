BINARY := zgit
GO ?= go
GOFLAGS ?= -ldflags="-s -w"

# Detect OS for output binary name
ifeq ($(OS),Windows_NT)
	BINARY := zgit.exe
endif

.PHONY: all build clean test lint vet fmt install dev

all: build

## build — Compile the zgit binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) ./cmd/zgit/

## clean — Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf ./tmp/

## test — Run all tests
test:
	$(GO) test ./... -count=1 -timeout=60s

## test-v — Run all tests with verbose output
test-v:
	$(GO) test ./... -v -count=1 -timeout=60s

## test-race — Run tests with race detector
test-race:
	$(GO) test ./... -race -count=1 -timeout=120s

## lint — Run golangci-lint (if installed)
lint:
	which golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

## vet — Run go vet
vet:
	$(GO) vet ./...

## fmt — Format all Go code
fmt:
	$(GO) fmt ./...

## install — Build and install to GOPATH/bin
install:
	$(GO) install $(GOFLAGS) ./cmd/zgit/

## dev — Quick dev loop: fmt -> vet -> test -> build
dev: fmt vet test build
	@echo "✓ dev build complete"

## coverage — Run tests with coverage
coverage:
	$(GO) test ./... -coverprofile=coverage.out -covermode=atomic -count=1
	$(GO) tool cover -func=coverage.out

## coverage-html — Open coverage report in browser
coverage-html: coverage
	$(GO) tool cover -html=coverage.out

## tidy — Tidy Go module dependencies
tidy:
	$(GO) mod tidy

## help — Show this help
help:
	@printf "\033[33mUsage:\033[0m\n  make \033[32m<target>\033[0m\n\n\033[33mTargets:\033[0m\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[32m%-20s\033[0m %s\n", $$1, $$2}'
