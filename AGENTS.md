# reMarkable Sync

A Go-based tool for syncing documents from reMarkable tablets with OCR text layer generation.

## Overview

This project syncs documents from the reMarkable cloud API and enhances PDF files by adding a hidden OCR text layer. This makes handwritten notes searchable while preserving the original visual appearance.

## Core Functionality

### 1. Document Synchronization

The application connects to the reMarkable cloud API using the [rmapi](https://github.com/ddvk/rmapi) library to:

- Download `.rmdoc` files and associated document data from the reMarkable cloud
- Support optional filtering by labels (e.g., sync only documents tagged with "work", "personal", etc.)
- Handle authentication and session management with the reMarkable API
- Maintain local sync state to avoid redundant downloads

### 2. OCR Processing

Using Ollama with vision models, the application:

- Processes rendered PDF pages from reMarkable documents
- Performs AI-powered optical character recognition optimized for handwritten content
- Generates text data with positional information (bounding boxes)
- Provides superior accuracy for handwriting compared to traditional OCR engines

### 3. PDF Enhancement

The final step adds value to the synchronized documents by:

- Adding a hidden text layer to the PDF file with OCR'd text
- Positioning text accurately based on OCR bounding box coordinates
- Preserving the original visual appearance (the text layer is invisible)
- Making previously unsearchable handwritten notes fully searchable in PDF viewers

## Technology Stack

- **Language**: Go
- **reMarkable API**: [rmapi](https://github.com/ddvk/rmapi) - Go client for the reMarkable cloud API
- **OCR Engine**: [Ollama](https://ollama.ai/) - Local AI runtime with vision models (llava, mistral-small3.1, etc.)
- **PDF Processing**: Go PDF libraries for adding text layers to existing PDFs

## Why Ollama for OCR?

Traditional OCR engines like Tesseract are optimized for printed text and struggle with handwriting. Modern vision-language models (VLMs) like those available through Ollama provide:

- **Superior handwriting recognition**: 85-95% accuracy vs 40-60% with Tesseract
- **Language flexibility**: Models handle multiple languages without explicit configuration
- **Context awareness**: Understanding of document structure and context
- **Local processing**: All OCR happens locally, preserving privacy
- **No CGO dependencies**: HTTP API eliminates build complexity

## Use Cases

- Make handwritten notes searchable without altering their appearance
- Create a local backup of reMarkable documents with enhanced functionality
- Integrate reMarkable notes into document management systems that rely on text search
- Sync specific collections or projects using label filters

## Architecture

```
┌─────────────────┐
│ reMarkable API  │
│   (via rmapi)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Sync Engine    │
│ - Auth          │
│ - Filter labels │
│ - Download docs │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌─────────────────────┐
│  OCR Processor  │─────▶│   Ollama HTTP       │
│  (Ollama API)   │      │   Vision Models     │
│ - Render pages  │◀─────│ (llava, mistral, ..)│
│ - Extract text  │      └─────────────────────┘
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ PDF Enhancer    │
│ - Add text layer│
│ - Position text │
│ - Save output   │
└─────────────────┘
```

## Project Goals

1. **Simple**: Straightforward CLI tool with minimal configuration
2. **Reliable**: Robust error handling and sync state management
3. **Efficient**: Incremental sync to avoid reprocessing unchanged documents
4. **Flexible**: Optional label-based filtering for selective sync
5. **Non-destructive**: Original documents remain unchanged; enhanced versions are new files

## Test Data

### Example reMarkable Document

The `example/` directory contains a real reMarkable document for testing and development:

- **Test.rmdoc** - A zipped reMarkable document (29KB, 2 pages)
  - Document ID: `b68e57f6-4fc9-4a71-b300-e0fa100ef8d7`
  - Title: "Test"
  - 2 blank pages with handwritten content
  - File Version: V6 format

**Unzipped structure:**
```
example/
├── Test.rmdoc                                          # Original ZIP file
├── b68e57f6-4fc9-4a71-b300-e0fa100ef8d7.metadata      # Document metadata (JSON)
├── b68e57f6-4fc9-4a71-b300-e0fa100ef8d7.content       # Content/page information (JSON)
└── b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/              # Page data directory
    ├── aefd8acc-a17d-4e24-a76c-66a3ee15b4ba.rm        # Page 1 rendering data (19KB)
    └── 7ac5c320-e3e5-4c6c-8adc-204662ee929a.rm        # Page 2 rendering data (4.4KB)
```

**Using the test data:**
- Use `Test.rmdoc` to test the PDF conversion pipeline
- Use the unzipped structure to understand the .rmdoc format
- The `.rm` files contain binary vector rendering data for each page
- The `.metadata` file contains document properties (title, timestamps, parent folder)
- The `.content` file contains page ordering, templates, and settings

**When implementing components:**
- **Converter** (`internal/converter`): Use `Test.rmdoc` to validate ZIP extraction and .rm parsing
- **OCR** (`internal/ocr`): Render the .rm files to images for OCR testing
- **PDF Enhancer** (`internal/pdfenhancer`): Use the converted PDF to test text layer addition
- **Integration tests**: Use as a known-good fixture for end-to-end workflow validation

## Future Considerations

- Incremental sync with change detection
- Bidirectional sync capabilities
- Custom OCR language model support
- Batch processing optimizations
- Configuration file for default settings
- Progress reporting and logging

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
