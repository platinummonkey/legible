# Integration Tests

This directory contains end-to-end integration tests for the legible application.

## Overview

Integration tests verify that multiple components work correctly together and that complete user workflows function as expected. Unlike unit tests that test individual components in isolation, integration tests exercise real code paths with actual dependencies.

## Test Organization

### config_state_test.go

Tests integration between configuration management, state persistence, and cross-component logging.

**Tests:**
- `TestConfigStateIntegration` - Config loading → State management → File persistence
- `TestLoggerIntegration` - Logger initialization and usage across components
- `TestRMClientTokenPersistence` - Authentication token persistence across sessions
- `TestMultipleDocumentWorkflow` - Managing multiple documents with filtering
- `TestStateUpdateWorkflow` - Document state transitions through sync lifecycle

**Dependencies:** None (runs without Tesseract)

### converter_test.go

Tests the PDF conversion pipeline from .rmdoc files.

**Tests:**
- `TestConverterPipeline` - Full .rmdoc → PDF conversion using testdata
- `TestConverterWithInvalidInput` - Error handling for invalid inputs
- `TestConverterMultipleDocuments` - Batch processing multiple documents
- `TestConverterOutputDirectory` - Output directory creation and nested paths
- `TestConverterMetadataExtraction` - Metadata extraction from .rmdoc files

**Dependencies:** Requires `testdata/rmdoc/Test.rmdoc` (automatically skips if not found)

### cli_test.go

Tests CLI command structure, flag parsing, and command execution.

**Tests:**
- `TestCLIBuild` - Binary build process
- `TestCLIVersion` - Version command output
- `TestCLIHelp` - Help command and flags
- `TestCLISyncCommandFlags` - Sync command flag parsing
- `TestCLIDaemonCommandFlags` - Daemon command flag parsing
- `TestCLIAuthCommand` - Auth command structure
- `TestCLIConfigFile` - Config file loading
- `TestCLIInvalidCommand` - Error handling

**Dependencies:** None (builds and tests CLI structure only)

**Note:** These tests use `testing.Short()` checks. Run with `-short` flag to skip CLI build tests.

## Running Integration Tests

### Run All Integration Tests

```bash
# From repository root
go test ./integration/... -v

# With short mode (skips CLI build tests)
go test ./integration/... -v -short
```

### Run Specific Test File

```bash
go test ./integration -run TestConfigStateIntegration -v
go test ./integration -run TestConverterPipeline -v
go test ./integration -run TestCLI -v
```

### Run with Coverage

```bash
go test ./integration/... -v -coverprofile=integration-coverage.out
go tool cover -html=integration-coverage.out
```

## Test Dependencies

### Required for All Tests

- Go 1.21+
- Access to `testdata/` directory
- Writable temporary directories (uses `t.TempDir()`)

### Optional Dependencies

- **testdata/rmdoc/Test.rmdoc**: Required for converter tests (tests skip if not found)
- **Tesseract OCR**: Not required for current integration tests (OCR tests to be added)

## Test Data

Integration tests use fixtures from the `testdata/` directory:

- `testdata/rmdoc/Test.rmdoc` - Sample reMarkable document (2 pages)
- `testdata/rmdoc/*.metadata` - Document metadata JSON
- `testdata/rmdoc/*.content` - Document content JSON

See [testdata/README.md](../testdata/README.md) for details on test fixtures.

## Integration Test Patterns

### 1. Configuration-Driven Tests

Tests that load configuration from files and verify component behavior:

```go
configPath := filepath.Join(tmpDir, "config.yaml")
os.WriteFile(configPath, []byte(configYAML), 0644)
os.Setenv("LEGIBLE_CONFIG", configPath)
cfg, _ := config.Load()
```

### 2. State Persistence Tests

Tests that verify data persistence across component lifecycle:

```go
mgr := state.NewManager(stateFile)
mgr.AddDocument(doc)
mgr.Save()

mgr2 := state.NewManager(stateFile)
mgr2.Load()
// Verify data persisted correctly
```

### 3. Pipeline Tests

Tests that exercise complete workflows end-to-end:

```go
// .rmdoc → PDF conversion pipeline
result, err := conv.ConvertRmdoc(inputPath, outputPath)
// Verify output file exists and has expected properties
```

### 4. CLI Integration Tests

Tests that build and execute CLI commands:

```go
cmd := exec.Command(binaryPath, "version")
output, err := cmd.CombinedOutput()
// Verify command output
```

## Adding New Integration Tests

When adding new integration tests:

1. **Choose the appropriate file** based on the component being tested
2. **Use descriptive test names** that explain the workflow being tested
3. **Use `t.TempDir()`** for temporary files and directories
4. **Skip gracefully** if required test data is missing:
   ```go
   if _, err := os.Stat(requiredFile); os.IsNotExist(err) {
       t.Skip("Required file not found, skipping test")
   }
   ```
5. **Test error paths** in addition to happy paths
6. **Log useful information** for debugging:
   ```go
   t.Logf("Conversion result: %+v", result)
   ```

## Future Integration Tests

The following integration tests are planned but not yet implemented due to Tesseract dependencies:

### Full Sync Workflow (Requires Tesseract)

```go
// Test: auth → list → download → convert → OCR → enhance
// Status: Pending Tesseract installation in CI
```

### OCR Pipeline (Requires Tesseract)

```go
// Test: PDF → images → OCR → text with bounding boxes
// Status: Pending Tesseract installation in CI
```

### PDF Enhancement (Requires Tesseract)

```go
// Test: Add searchable text layer to PDF
// Status: Pending Tesseract installation in CI
```

### Daemon Lifecycle (Requires Tesseract)

```go
// Test: Startup → periodic sync → graceful shutdown
// Status: Pending Tesseract installation in CI
```

## CI/CD Integration

Integration tests run in GitHub Actions CI pipeline:

```yaml
- name: Run integration tests
  run: go test ./integration/... -v -short
```

**Note:** Short mode is used in CI to skip CLI build tests that may be slow or resource-intensive.

## Troubleshooting

### Test Skips

If tests are skipping:

```
--- SKIP: TestConverterPipeline (0.00s)
    converter_test.go:XX: Test.rmdoc not found in testdata
```

**Solution:** Ensure `testdata/rmdoc/Test.rmdoc` exists. Run from repository root.

### Build Failures

If CLI build tests fail:

```
FAIL: TestCLIBuild - Failed to build CLI
```

**Solution:**
1. Check that `cmd/legible/main.go` exists
2. Verify all dependencies are available: `go mod download`
3. Try building manually: `go build ./cmd/legible`

### Permission Errors

If tests fail with permission errors:

```
FAIL: Permission denied when creating state file
```

**Solution:** Ensure test has write access to temporary directories. This should work automatically with `t.TempDir()`, but check filesystem permissions if issues persist.

## Best Practices

1. **Isolation**: Each test should be independent and not affect other tests
2. **Cleanup**: Use `t.TempDir()` for automatic cleanup, or `defer` for manual cleanup
3. **Determinism**: Tests should produce consistent results across runs
4. **Speed**: Keep integration tests reasonably fast (< 30s per test)
5. **Clarity**: Test names and error messages should clearly indicate what failed

## Coverage Goals

Target coverage for integration tests:

- **Config + State**: 95%+ of integration paths
- **Converter**: 90%+ of conversion pipeline
- **CLI**: 80%+ of command structure and flag parsing
- **Full workflows**: 100% of critical user paths (pending Tesseract)

Current integration test coverage can be viewed with:

```bash
go test ./integration/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Related Documentation

- [Unit Test Coverage](../TEST_COVERAGE.md) - Unit test status and guidelines
- [Test Data](../testdata/README.md) - Test fixture documentation
- [Development Guide](../README.md) - General development information
