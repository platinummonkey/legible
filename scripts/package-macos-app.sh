#!/usr/bin/env bash
#
# Package macOS menu bar app into .app bundle
# Usage: package-macos-app.sh <binary-path> <output-dir> <version>
#

set -e

BINARY_PATH="$1"
OUTPUT_DIR="$2"
VERSION="$3"

if [ -z "$BINARY_PATH" ] || [ -z "$OUTPUT_DIR" ] || [ -z "$VERSION" ]; then
    echo "Usage: $0 <binary-path> <output-dir> <version>"
    exit 1
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    exit 1
fi

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

APP_NAME="Legible.app"
APP_BUNDLE="$OUTPUT_DIR/$APP_NAME"

echo "Creating macOS app bundle: $APP_BUNDLE"

# Create app bundle structure
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

# Copy binary
cp "$BINARY_PATH" "$APP_BUNDLE/Contents/MacOS/legible-menubar"
chmod +x "$APP_BUNDLE/Contents/MacOS/legible-menubar"

# Process Info.plist (replace version placeholder)
sed "s/{{.Version}}/$VERSION/g" "$PROJECT_ROOT/assets/macos-app/Info.plist" > "$APP_BUNDLE/Contents/Info.plist"

# Copy icons if they exist
if [ -f "$PROJECT_ROOT/assets/macos-app/AppIcon.icns" ]; then
    cp "$PROJECT_ROOT/assets/macos-app/AppIcon.icns" "$APP_BUNDLE/Contents/Resources/"
fi

echo "✓ App bundle created successfully: $APP_BUNDLE"

# Create a zip for distribution
cd "$OUTPUT_DIR"
zip -r "${APP_NAME%.app}.zip" "$APP_NAME"
echo "✓ Created distribution zip: $OUTPUT_DIR/${APP_NAME%.app}.zip"
