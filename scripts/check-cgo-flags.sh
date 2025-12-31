#!/usr/bin/env bash
# Script to verify pkg-config CGO flags are set correctly
# Used for debugging build issues

set -e

echo "=== Checking pkg-config and CGO configuration ==="
echo

# Check if pkg-config is available
if ! command -v pkg-config &> /dev/null; then
    echo "ERROR: pkg-config not found. Please install pkg-config."
    exit 1
fi
echo "✓ pkg-config found: $(which pkg-config)"
echo

# Check tesseract
echo "Checking tesseract..."
if pkg-config --exists tesseract 2>/dev/null; then
    echo "✓ Tesseract found: $(pkg-config --modversion tesseract)"
    echo "  CFLAGS:  $(pkg-config --cflags tesseract)"
    echo "  LDFLAGS: $(pkg-config --libs tesseract)"
else
    echo "ERROR: pkg-config cannot find tesseract"
    echo "  Please install: brew install tesseract (macOS) or apt-get install libtesseract-dev (Linux)"
    exit 1
fi
echo

# Check leptonica
echo "Checking leptonica..."
if pkg-config --exists lept 2>/dev/null; then
    echo "✓ Leptonica found: $(pkg-config --modversion lept)"
    echo "  CFLAGS:  $(pkg-config --cflags lept)"
    echo "  LDFLAGS: $(pkg-config --libs lept)"

    # Show the corrected include path
    LEPT_CFLAGS_ORIG=$(pkg-config --cflags lept)
    LEPT_CFLAGS_FIXED=$(echo "$LEPT_CFLAGS_ORIG" | sed 's|/leptonica$||')
    echo "  CFLAGS (corrected): $LEPT_CFLAGS_FIXED"
else
    echo "ERROR: pkg-config cannot find leptonica"
    echo "  Please install: brew install leptonica (macOS) or apt-get install libleptonica-dev (Linux)"
    exit 1
fi
echo

# Show combined CGO flags as they would be set by Makefile
echo "=== Combined CGO flags (as used by Makefile) ==="
TESS_CFLAGS=$(pkg-config --cflags tesseract 2>/dev/null)
LEPT_CFLAGS_ORIG=$(pkg-config --cflags lept 2>/dev/null)
LEPT_CFLAGS_FIXED=$(echo "$LEPT_CFLAGS_ORIG" | sed 's|/leptonica$||')
LDFLAGS=$(pkg-config --libs tesseract lept 2>/dev/null)

echo "export CGO_CFLAGS=\"$TESS_CFLAGS $LEPT_CFLAGS_FIXED $LEPT_CFLAGS_ORIG\""
echo "export CGO_CXXFLAGS=\"$TESS_CFLAGS $LEPT_CFLAGS_FIXED $LEPT_CFLAGS_ORIG\""
echo "export CGO_LDFLAGS=\"$LDFLAGS\""
echo "export CGO_ENABLED=1"
echo

echo "=== All checks passed! ==="
