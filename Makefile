.PHONY: build run test test-cover clean install lint fmt help

# Binary name
BINARY_NAME=pilot
BUILD_DIR=.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pilot

## run: Build and run the application
run: build
	./$(BINARY_NAME)

## test: Run all tests
test:
	$(GOTEST) ./...

## test-cover: Run tests with coverage report
test-cover:
	$(GOTEST) ./... -cover

## test-cover-html: Generate HTML coverage report
test-cover-html:
	$(GOTEST) ./... -coverprofile=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-race: Run tests with race detector
test-race:
	$(GOTEST) ./... -race

## test-verbose: Run tests with verbose output
test-verbose:
	$(GOTEST) ./... -v

## lint: Run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOFMT) ./...

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## install: Install binary to $GOPATH/bin
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/pilot

## dev: Run with live reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

## deps: Download dependencies
deps:
	$(GOMOD) download

## update: Update dependencies
update:
	$(GOGET) -u ./...
	$(GOMOD) tidy
