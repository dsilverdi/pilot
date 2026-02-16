.PHONY: build run test test-cover clean install setup uninstall lint fmt help

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

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Install directory
INSTALL_DIR ?= /usr/local/bin

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

## install: Install binary to /usr/local/bin (requires sudo)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed! Run 'pilot' to start"

## install-user: Install binary to ~/bin (no sudo required)
install-user: build
	@mkdir -p $(HOME)/bin
	cp $(BINARY_NAME) $(HOME)/bin/$(BINARY_NAME)
	chmod +x $(HOME)/bin/$(BINARY_NAME)
	@echo "Installed to ~/bin/$(BINARY_NAME)"
	@if echo "$$PATH" | grep -q "$(HOME)/bin"; then \
		echo "Run 'pilot' to start"; \
	else \
		echo "Add this to your shell config: export PATH=\"\$$HOME/bin:\$$PATH\""; \
	fi

## uninstall: Remove installed binary
uninstall:
	@if [ -f $(INSTALL_DIR)/$(BINARY_NAME) ]; then \
		sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
		echo "Removed $(INSTALL_DIR)/$(BINARY_NAME)"; \
	fi
	@if [ -f $(HOME)/bin/$(BINARY_NAME) ]; then \
		rm -f $(HOME)/bin/$(BINARY_NAME); \
		echo "Removed $(HOME)/bin/$(BINARY_NAME)"; \
	fi

## setup: Run the setup script (interactive installation)
setup:
	./scripts/setup.sh

## setup-user: Run setup script for user-local installation
setup-user:
	./scripts/setup.sh --user

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

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
