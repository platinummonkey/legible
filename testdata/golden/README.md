# Golden Files

This directory contains expected output files for comparison testing.

## Purpose

Golden files are reference outputs that tests compare against to verify correctness. They represent the expected behavior of the system.

## Structure

```
golden/
├── converter/
│   ├── test-document-page1.txt       # Expected text from page 1
│   ├── test-document-metadata.json   # Expected parsed metadata
│   └── test-document-content.json    # Expected parsed content
├── ocr/
│   ├── simple-text-hocr.xml         # Expected HOCR output
│   ├── simple-text-words.json       # Expected word extraction
│   └── simple-text-confidence.json  # Expected confidence scores
└── sync/
    ├── state-after-sync.json        # Expected state file after sync
    └── sync-result.json             # Expected sync result summary
```

## Usage

Golden files are used in tests for comparison:

```go
func TestConverter(t *testing.T) {
    result := converter.Convert(input)

    // Compare with golden file
    golden, _ := os.ReadFile("testdata/golden/converter/output.json")
    if !bytes.Equal(result, golden) {
        t.Errorf("output differs from golden file")
    }
}
```

## Updating Golden Files

When expected behavior changes (intentionally), update golden files:

1. Run tests to see failures
2. Verify new output is correct
3. Copy new output to golden file
4. Re-run tests to confirm

Many test frameworks provide `-update` flags to automate this:

```bash
go test -update-golden ./...
```

## Best Practices

- **Keep files small**: Only include minimal data needed for validation
- **Use stable formats**: JSON, text, or other deterministic formats
- **Document changes**: Note why golden files were updated in commits
- **Version control**: Always commit golden file changes with code changes
- **Review carefully**: Golden file changes can hide bugs

## File Naming Convention

- Use descriptive names: `{test-case}-{aspect}.{ext}`
- Examples:
  - `simple-text-ocr-result.json`
  - `multi-page-document-page-count.txt`
  - `metadata-parsing-output.json`

## Maintenance

Golden files should be reviewed periodically to ensure they:
- Still represent correct behavior
- Use current format versions
- Are minimal and focused
- Have corresponding tests
