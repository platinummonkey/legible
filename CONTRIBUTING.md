# Contributing to reMarkable Sync

Thank you for your interest in contributing to reMarkable Sync! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Reporting Bugs](#reporting-bugs)
- [Feature Requests](#feature-requests)

## Code of Conduct

This project follows a standard code of conduct. Please be respectful and constructive in all interactions.

- Be welcoming to newcomers
- Be respectful of differing viewpoints
- Accept constructive criticism gracefully
- Focus on what is best for the community

## Getting Started

### Prerequisites

- Go 1.24 or higher
- goreleaser (for building)
- pkg-config (for library detection)
- Tesseract OCR and Leptonica (for OCR features)
- Git
- Make (optional, but recommended)
- golangci-lint (for linting)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/remarkable-sync.git
   cd remarkable-sync
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/platinummonkey/remarkable-sync.git
   ```

## Development Setup

### Install Dependencies

```bash
# Download Go module dependencies
go mod download

# Install goreleaser (macOS)
brew install goreleaser

# Install pkg-config and libraries (macOS)
brew install pkg-config tesseract leptonica

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Build the Project

We use goreleaser for consistent, production-quality builds:

```bash
# Build for current platform (recommended)
make build-local

# The binary will be in:
# dist/remarkable-sync_<os>_<arch>/remarkable-sync

# Build for all platforms
make build-all
```

**Why goreleaser?**
- Same configuration for local dev and releases
- Consistent builds across platforms
- Automatic version injection
- Pre-configured CGO handling via pkg-config

### Verify Installation

```bash
# Verify pkg-config can find libraries
pkg-config --modversion tesseract lept

# Run tests (without OCR if Tesseract not available)
make test-no-ocr

# Run linter
make lint
```

## Development Workflow

### Create a Feature Branch

```bash
# Update your fork
git fetch upstream
git checkout main
git merge upstream/main

# Create a feature branch
git checkout -b feature/my-new-feature
```

### Make Changes

1. **Write code** following the project's code style
2. **Add tests** for new functionality
3. **Update documentation** if needed
4. **Run tests** to ensure nothing breaks
5. **Run linter** to catch style issues

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Commit Your Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "Add feature: description of what you did"
```

**Good commit message examples:**
- `Add OCR support for Spanish language`
- `Fix authentication token expiration handling`
- `Improve daemon shutdown gracefully`

**Bad commit message examples:**
- `fix bug`
- `update`
- `changes`

### Keep Your Branch Updated

```bash
git fetch upstream
git rebase upstream/main
```

## Code Style

### Go Style Guide

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (automatically applied by `make fmt`)
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names

### Code Organization

- **One package per directory**: Keep related code together
- **Interfaces first**: Define interfaces before implementations
- **Error handling**: Always handle errors explicitly
- **Logging**: Use the internal logger package, not fmt.Println

### Documentation

- **Exported functions**: Must have documentation comments
- **Complex logic**: Add inline comments explaining why, not what
- **Package comments**: Each package should have a doc.go or comment on the package declaration

Example:

```go
// ProcessDocument converts a .rmdoc file to PDF with optional OCR.
// It returns the output path and any error encountered during processing.
//
// The conversion process includes:
//   - Extracting the .rmdoc archive
//   - Parsing metadata and content
//   - Rendering pages to PDF
//   - Optionally adding OCR text layer
func ProcessDocument(input, output string, opts *Options) (string, error) {
    // Implementation
}
```

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to read file %s: %w", path, err)
}

// Bad: Lose error context
if err != nil {
    return err
}
```

## Testing

### Unit Tests

- Write tests for all new functions
- Use table-driven tests for multiple scenarios
- Aim for 80%+ code coverage

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

- Add integration tests to `integration/` directory
- Tests should be idempotent and isolated
- Use `t.TempDir()` for temporary files
- Mark slow tests with `if testing.Short() { t.Skip() }`

### Running Tests

```bash
# Run all tests
make test

# Run tests without Tesseract dependencies
make test-no-ocr

# Run with coverage
make test-coverage

# Run specific test
go test ./internal/converter -run TestConvert -v

# Run integration tests
go test ./integration/... -v
```

## Submitting Changes

### Before Submitting

- [ ] All tests pass: `make test`
- [ ] Linter passes: `make lint`
- [ ] Code is formatted: `make fmt`
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with upstream main

### Create a Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin feature/my-new-feature
   ```

2. Go to GitHub and create a pull request

3. Fill out the pull request template:
   - Describe what the PR does
   - Reference any related issues
   - Note any breaking changes
   - Include screenshots if UI changes

### Pull Request Guidelines

- **Keep PRs focused**: One feature or fix per PR
- **Write clear descriptions**: Explain what and why, not just what
- **Link to issues**: Use "Fixes #123" or "Relates to #456"
- **Be responsive**: Address review comments promptly
- **Update if needed**: Keep your PR updated with main

### Review Process

- Maintainers will review your PR
- Address any feedback or requested changes
- Once approved, maintainers will merge your PR
- Your contribution will be in the next release!

## Reporting Bugs

### Before Reporting

- Check if the bug has already been reported
- Verify it's actually a bug (not a feature or configuration issue)
- Test with the latest version

### Bug Report Template

When opening an issue, include:

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run command '...'
2. With config '....'
3. See error

**Expected behavior**
What you expected to happen.

**Actual behavior**
What actually happened.

**Environment:**
- OS: [e.g., macOS 13.0, Ubuntu 22.04]
- Go version: [e.g., 1.21.0]
- remarkable-sync version: [e.g., v0.1.0]
- Tesseract version: [e.g., 5.3.0]

**Logs**
```
Paste relevant log output here (run with --log-level debug)
```

**Additional context**
Any other relevant information.
```

## Feature Requests

We welcome feature requests! When requesting a feature:

1. **Check existing issues**: Someone may have already requested it
2. **Describe the use case**: Explain why you need this feature
3. **Propose a solution**: If you have ideas on implementation
4. **Offer to contribute**: Willing to implement it yourself?

### Feature Request Template

```markdown
**Is your feature request related to a problem?**
A clear description of the problem. Ex. I'm frustrated when [...]

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Alternative solutions or features you've considered.

**Use case**
How would this feature be used? Who would benefit?

**Additional context**
Any other context, screenshots, or examples.
```

## Development Tips

### Debugging

```bash
# Run with debug logging
remarkable-sync sync --log-level debug

# Run with Go race detector
go test -race ./...

# Profile CPU usage
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

### Project Structure

```
remarkable-sync/
├── cmd/
│   └── remarkable-sync/     # CLI entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── converter/           # .rmdoc to PDF conversion
│   ├── daemon/              # Daemon mode
│   ├── logger/              # Structured logging
│   ├── ocr/                 # OCR processing
│   ├── pdfenhancer/         # PDF text layer addition
│   ├── rmclient/            # reMarkable API client
│   ├── state/               # Sync state management
│   └── sync/                # Sync orchestration
├── integration/             # Integration tests
├── testdata/                # Test fixtures
└── examples/                # Example configs and scripts
```

### Useful Make Targets

```bash
make help           # Show all available targets
make build-local    # Build for current platform (uses goreleaser, recommended)
make build-all      # Build for all platforms (uses goreleaser)
make build          # Simple Go build (single platform, no goreleaser)
make test           # Run tests
make test-no-ocr    # Run tests without Tesseract
make lint           # Run linter
make fmt            # Format code
make clean          # Clean build artifacts
make dev            # Run in development mode
```

### Build System Details

The project uses goreleaser for builds. Configuration is in `.goreleaser.yaml`.

**CGO Configuration:**
- The Makefile automatically sets CGO flags using pkg-config
- Flags are exported: `CGO_CFLAGS`, `CGO_CXXFLAGS`, `CGO_LDFLAGS`
- Libraries detected: Tesseract OCR and Leptonica

**Troubleshooting builds:**
```bash
# Verify CGO flags are set correctly
scripts/check-cgo-flags.sh

# Manual CGO flag export (if needed)
export CGO_CFLAGS="$(pkg-config --cflags tesseract lept)"
export CGO_LDFLAGS="$(pkg-config --libs tesseract lept)"
make build-local
```

## Getting Help

- **Documentation**: Check [README.md](README.md) and [FAQ.md](FAQ.md)
- **Issues**: Search existing issues on GitHub
- **Discussions**: Start a discussion for questions
- **Email**: Contact maintainers for private matters

## Recognition

Contributors are recognized in:
- Git commit history
- Release notes
- GitHub contributors page

Thank you for contributing to reMarkable Sync!
