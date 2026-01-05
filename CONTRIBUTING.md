# Contributing to Legible

Thank you for your interest in contributing to Legible! This document provides guidelines and instructions for contributing to the project.

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
- Ollama (for local OCR testing, optional)
- Git
- Make (optional, but recommended)
- golangci-lint (for linting)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/legible.git
   cd legible
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/platinummonkey/legible.git
   ```

## Development Setup

### Install Dependencies

```bash
# Download Go module dependencies
go mod download

# Install goreleaser (macOS)
brew install goreleaser

# Install Ollama for local OCR testing (optional, macOS)
brew install ollama
ollama pull llava  # or mistral-small3.1

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Build the Project

We use goreleaser for consistent, production-quality builds:

```bash
# Build for current platform (recommended)
make build-local

# The binary will be in:
# dist/legible_<os>_<arch>/legible

# Build for all platforms
make build-all
```

**Why goreleaser?**
- Same configuration for local dev and releases
- Consistent builds across platforms
- Automatic version injection
- No CGO dependencies required

### Verify Installation

```bash
# Run tests
make test

# Run linter
make lint

# Verify Ollama is running (optional, for OCR tests)
ollama list
curl http://localhost:11434/api/tags
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
- legible version: [e.g., v0.1.0]
- LLM Provider: [e.g., ollama, openai, anthropic]
- Ollama version (if applicable): [e.g., 0.1.20]

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
legible sync --log-level debug

# Run with Go race detector
go test -race ./...

# Profile CPU usage
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

### Project Structure

```
legible/
├── cmd/
│   └── legible/     # CLI entry point
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
make lint           # Run linter
make fmt            # Format code
make clean          # Clean build artifacts
make dev            # Run in development mode
```

### Generating and Validating SBOMs

Software Bill of Materials (SBOMs) are automatically generated by goreleaser during releases. To test SBOM generation locally:

**Install syft (SBOM generator):**

```bash
# macOS
brew install syft

# Linux
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Windows
scoop install syft
```

**Generate SBOMs manually:**

```bash
# Generate SPDX format SBOM
syft packages . -o spdx-json=sbom.spdx.json

# Generate CycloneDX format SBOM
syft packages . -o cyclonedx-json=sbom.cyclonedx.json

# Generate from a built binary
syft packages ./dist/legible_linux_amd64/legible -o spdx-json=binary.sbom.json
```

**Test with goreleaser:**

```bash
# Build with goreleaser in snapshot mode (includes SBOM generation)
goreleaser build --snapshot --clean --single-target

# SBOMs will be in dist/ directory
ls -la dist/*.sbom.spdx.json
```

**Validate SBOM contents:**

```bash
# Inspect SBOM
syft sbom.spdx.json

# Extract package list
jq '.packages[].name' sbom.spdx.json | sort | uniq

# Verify required fields
jq '.spdxVersion, .dataLicense, .SPDXID, .name, .documentNamespace' sbom.spdx.json
```

**Scan for vulnerabilities:**

```bash
# Install grype (vulnerability scanner)
brew install grype

# Scan using SBOM
grype sbom:./sbom.spdx.json

# Scan binary directly
grype ./dist/legible_linux_amd64/legible

# Scan container image
grype ghcr.io/platinummonkey/legible:latest
```

**SBOM Best Practices:**

- Always generate SBOMs for release builds
- Include both source and binary SBOMs
- Validate SBOM format and completeness
- Scan for vulnerabilities before releasing
- Document SBOM availability in release notes

**Testing Container SBOMs:**

```bash
# Build Docker image locally with SBOM
docker buildx build --sbom=true --provenance=true -t legible:test .

# Inspect SBOM metadata
docker buildx imagetools inspect legible:test --format "{{ json .SBOM }}"

# Export SBOM to file
docker buildx imagetools inspect legible:test --format "{{ json .SBOM }}" > container.sbom.json
```

### Build System Details

The project uses goreleaser for builds. Configuration is in `.goreleaser.yaml`.

**CGO Configuration:**
- CGO is disabled (`CGO_ENABLED=0`) since we don't use native dependencies
- Builds are pure Go and cross-compile easily
- No external libraries required for OCR (uses Ollama HTTP API)

**Troubleshooting builds:**
```bash
# Verify build works
make build-local

# Clean and rebuild if issues occur
make clean
go mod tidy
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

Thank you for contributing to Legible!
