.PHONY: all build build-all test test-no-ocr test-coverage test-coverage-no-ocr lint fmt vet tidy install clean deps verify run dev version help

# Binary name
BINARY_NAME=remarkable-sync
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

# CGO flags for Tesseract/Leptonica (using pkg-config for dynamic path resolution)
# If pkg-config is not available or libraries not found, these will be empty
PKG_CONFIG?=pkg-config
CGO_CFLAGS_TESS=$(shell $(PKG_CONFIG) --cflags tesseract 2>/dev/null)
CGO_CFLAGS_LEPT=$(shell $(PKG_CONFIG) --cflags lept 2>/dev/null | sed 's|/leptonica$$||')
CGO_LDFLAGS_TESSERACT=$(shell $(PKG_CONFIG) --libs tesseract lept 2>/dev/null)

# Export CGO flags if libraries are found
# Note: leptonica's pkg-config includes /leptonica in the path, but headers use #include <leptonica/...>
# so we need both the original path and the parent directory
ifneq ($(CGO_CFLAGS_TESS),)
export CGO_CFLAGS=$(CGO_CFLAGS_TESS) $(CGO_CFLAGS_LEPT) $(shell $(PKG_CONFIG) --cflags lept 2>/dev/null)
export CGO_CXXFLAGS=$(CGO_CFLAGS_TESS) $(CGO_CFLAGS_LEPT) $(shell $(PKG_CONFIG) --cflags lept 2>/dev/null)
export CGO_LDFLAGS=$(CGO_LDFLAGS_TESSERACT)
export CGO_ENABLED=1
else
$(warning Warning: pkg-config could not find tesseract/lept. OCR features may not compile.)
export CGO_ENABLED=0
endif

# Build directory
BUILD_DIR=./bin

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
	@echo "Binary available in dist/$(BINARY_NAME)_*/$(BINARY_NAME)"

build-all: ## Build for all platforms using goreleaser
	@echo "Building for all platforms with goreleaser..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser build --snapshot --clean
	@echo "Binaries available in dist/$(BINARY_NAME)_*/"

build-release: ## Create a release build (requires git tag)
	@echo "Creating release build..."
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser release --clean

test: ## Run all tests (requires Tesseract OCR installed)
	@echo "Running all tests..."
	@echo "Note: This requires Tesseract OCR to be installed. Use 'make test-no-ocr' if Tesseract is not available."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-no-ocr: ## Run tests without OCR dependencies (no Tesseract required)
	@echo "Running tests (excluding OCR-dependent packages)..."
	$(GOTEST) -v -race -coverprofile=coverage-no-ocr.out ./internal/config ./internal/converter ./internal/logger ./internal/rmclient ./internal/state

test-coverage: test ## Run all tests with coverage report (requires Tesseract)
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-no-ocr: test-no-ocr ## Run tests with coverage report (no Tesseract required)
	@echo "Generating coverage report (no OCR)..."
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
