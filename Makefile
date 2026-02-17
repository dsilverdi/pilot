.PHONY: build build-pilot build-gateway run run-gateway test test-cover clean install setup uninstall lint fmt help

# Binary names
PILOT_BINARY=pilot
GATEWAY_BINARY=pilot-gateway
BUILD_DIR=bin

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

# Load .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'

## build: Build all binaries (pilot and pilot-gateway)
build: build-pilot build-gateway

## build-pilot: Build the pilot CLI binary
build-pilot:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PILOT_BINARY) ./cmd/pilot

## build-gateway: Build the pilot-gateway binary
build-gateway:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GATEWAY_BINARY) ./cmd/pilot-gateway

## run: Build and run the pilot CLI
run: build-pilot
	./$(BUILD_DIR)/$(PILOT_BINARY)

## run-gateway: Build and run the gateway server
run-gateway: build-gateway
	./$(BUILD_DIR)/$(GATEWAY_BINARY)

## run-gateway-bg: Run gateway in background (logs to gateway.log)
run-gateway-bg: build-gateway
	@echo "Starting gateway in background..."
	./$(BUILD_DIR)/$(GATEWAY_BINARY) > gateway.log 2>&1 &
	@echo "Gateway started. Logs: gateway.log"
	@echo "Stop with: make stop-gateway"

## stop-gateway: Stop the background gateway
stop-gateway:
	@pkill -f $(GATEWAY_BINARY) 2>/dev/null || echo "Gateway not running"

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
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html gateway.log

## install: Install binaries to /usr/local/bin (requires sudo)
install: build
	@echo "Installing binaries to $(INSTALL_DIR)..."
	sudo cp $(BUILD_DIR)/$(PILOT_BINARY) $(INSTALL_DIR)/$(PILOT_BINARY)
	sudo cp $(BUILD_DIR)/$(GATEWAY_BINARY) $(INSTALL_DIR)/$(GATEWAY_BINARY)
	sudo chmod +x $(INSTALL_DIR)/$(PILOT_BINARY)
	sudo chmod +x $(INSTALL_DIR)/$(GATEWAY_BINARY)
	@echo "Installed! Run 'pilot' or 'pilot-gateway' to start"

## install-user: Install binaries to ~/bin (no sudo required)
install-user: build
	@mkdir -p $(HOME)/bin
	cp $(BUILD_DIR)/$(PILOT_BINARY) $(HOME)/bin/$(PILOT_BINARY)
	cp $(BUILD_DIR)/$(GATEWAY_BINARY) $(HOME)/bin/$(GATEWAY_BINARY)
	chmod +x $(HOME)/bin/$(PILOT_BINARY)
	chmod +x $(HOME)/bin/$(GATEWAY_BINARY)
	@echo "Installed to ~/bin/"
	@if echo "$$PATH" | grep -q "$(HOME)/bin"; then \
		echo "Run 'pilot' or 'pilot-gateway' to start"; \
	else \
		echo "Add this to your shell config: export PATH=\"\$$HOME/bin:\$$PATH\""; \
	fi

## uninstall: Remove installed binaries
uninstall:
	@if [ -f $(INSTALL_DIR)/$(PILOT_BINARY) ]; then \
		sudo rm -f $(INSTALL_DIR)/$(PILOT_BINARY); \
		echo "Removed $(INSTALL_DIR)/$(PILOT_BINARY)"; \
	fi
	@if [ -f $(INSTALL_DIR)/$(GATEWAY_BINARY) ]; then \
		sudo rm -f $(INSTALL_DIR)/$(GATEWAY_BINARY); \
		echo "Removed $(INSTALL_DIR)/$(GATEWAY_BINARY)"; \
	fi
	@if [ -f $(HOME)/bin/$(PILOT_BINARY) ]; then \
		rm -f $(HOME)/bin/$(PILOT_BINARY); \
		echo "Removed $(HOME)/bin/$(PILOT_BINARY)"; \
	fi
	@if [ -f $(HOME)/bin/$(GATEWAY_BINARY) ]; then \
		rm -f $(HOME)/bin/$(GATEWAY_BINARY); \
		echo "Removed $(HOME)/bin/$(GATEWAY_BINARY)"; \
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

## api-key-generate: Generate a new API key (usage: make api-key-generate NAME=mykey)
api-key-generate: build-pilot
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make api-key-generate NAME=<key-name>"; \
		exit 1; \
	fi
	./$(BUILD_DIR)/$(PILOT_BINARY) api-key generate --name $(NAME)

## api-key-list: List all API keys
api-key-list: build-pilot
	./$(BUILD_DIR)/$(PILOT_BINARY) api-key list

## api-key-revoke: Revoke an API key (usage: make api-key-revoke NAME=mykey)
api-key-revoke: build-pilot
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make api-key-revoke NAME=<key-name>"; \
		exit 1; \
	fi
	./$(BUILD_DIR)/$(PILOT_BINARY) api-key revoke --name $(NAME)

## setup-systemd: Install pilot-gateway as a systemd service (requires sudo)
setup-systemd: build
	@echo "Installing pilot-gateway as systemd service..."
	@echo "This requires sudo privileges."
	sudo ./scripts/setup-systemd.sh install

## uninstall-systemd: Remove the systemd service
uninstall-systemd:
	sudo ./scripts/setup-systemd.sh uninstall

## status-systemd: Show systemd service status
status-systemd:
	./scripts/setup-systemd.sh status
