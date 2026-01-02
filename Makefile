.PHONY: all build build-all test test-no-ocr test-coverage test-coverage-no-ocr lint fmt vet tidy install clean deps verify run dev version help

# Binary name
BINARY_NAME=legible
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install

# CGO configuration
# OCR now uses Ollama (HTTP API) instead of Tesseract, so CGO is not required
# Note: unipdf may still require CGO for some PDF operations
export CGO_ENABLED=1

# Build directory
BUILD_DIR=./dist

all: lint test build-local

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary (Go-based, single platform)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-local: ## Build binary for current platform using goreleaser (recommended)
	@echo "Building with goreleaser for current platform..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser build --snapshot --clean --single-target
	@echo "Binary available in ${BUILD_DIR}/$(BINARY_NAME)_*/$(BINARY_NAME)"

build-all: ## Build for all platforms using goreleaser
	@echo "Building for all platforms with goreleaser..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser build --snapshot --clean
	@echo "Binaries available in ${BUILD_DIR}/$(BINARY_NAME)_*/"

build-release: ## Create a release build (requires git tag)
	@echo "Creating release build..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser release --clean

test: ## Run all tests
	@echo "Running all tests..."
	@echo "Note: OCR tests use mock Ollama server and do not require Ollama installed."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-no-ocr: ## Run tests without OCR and Ollama packages
	@echo "Running tests (excluding OCR and Ollama packages)..."
	$(GOTEST) -v -race -coverprofile=coverage-no-ocr.out ./internal/config ./internal/converter ./internal/logger ./internal/rmclient ./internal/state

test-coverage: test ## Run all tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-no-ocr: test-no-ocr ## Run tests with coverage report (excluding OCR/Ollama)
	@echo "Generating coverage report (no OCR/Ollama)..."
	$(GOCMD) tool cover -html=coverage-no-ocr.out -o coverage-no-ocr.html
	@echo "Coverage report generated: coverage-no-ocr.html"

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

install: build ## Install the binary
	@echo "Installing $(BINARY_NAME)..."
	$(GOINSTALL) $(LDFLAGS) ./cmd/$(BINARY_NAME)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html coverage-no-ocr.out coverage-no-ocr.html

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify

run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run in development mode
	@echo "Running in development mode..."
	$(GOCMD) run ./cmd/$(BINARY_NAME)

version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
