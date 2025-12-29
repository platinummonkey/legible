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

Using Tesseract OCR, the application:

- Processes rendered PDF pages from reMarkable documents
- Performs optical character recognition on handwritten and typed content
- Generates text data with positional information (bounding boxes)

### 3. PDF Enhancement

The final step adds value to the synchronized documents by:

- Adding a hidden text layer to the PDF file with OCR'd text
- Positioning text accurately based on OCR bounding box coordinates
- Preserving the original visual appearance (the text layer is invisible)
- Making previously unsearchable handwritten notes fully searchable in PDF viewers

## Technology Stack

- **Language**: Go
- **reMarkable API**: [rmapi](https://github.com/ddvk/rmapi) - Go client for the reMarkable cloud API
- **OCR Engine**: Tesseract - Industry-standard open-source OCR engine
- **PDF Processing**: Go PDF libraries for adding text layers to existing PDFs

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
┌─────────────────┐
│  OCR Processor  │
│  (Tesseract)    │
│ - Render pages  │
│ - Extract text  │
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
