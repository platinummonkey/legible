#!/usr/bin/env python3
"""
Generate menu bar icons for the Legible app.

Creates simple, clean circular status indicators optimized for macOS menu bar:
- Green: Idle/sync complete
- Yellow: Syncing in progress
- Red: Error state

Icons are 22x22 pixels (standard menu bar size) with transparency.
"""

from PIL import Image, ImageDraw
import sys

def create_circle_icon(color, output_path, size=22):
    """
    Create a simple circular status icon.

    Args:
        color: RGB tuple (r, g, b)
        output_path: Path to save the PNG file
        size: Icon size in pixels (default 22x22)
    """
    # Create image with transparency
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Calculate circle dimensions (with some padding)
    padding = 3
    circle_size = size - (padding * 2)

    # Draw circle
    draw.ellipse(
        [padding, padding, padding + circle_size, padding + circle_size],
        fill=color + (255,),  # Add alpha channel (fully opaque)
        outline=None
    )

    # Save the image
    img.save(output_path, 'PNG')
    print(f"Created: {output_path}")

def create_document_icon(color, output_path, size=22):
    """
    Create a document/paper icon with a colored status dot.

    Args:
        color: RGB tuple for the status dot
        output_path: Path to save the PNG file
        size: Icon size in pixels
    """
    # Create image with transparency
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Document color (light gray for visibility on both themes)
    doc_color = (160, 160, 160, 255)

    # Draw document outline (simplified paper shape)
    # Main rectangle
    doc_padding = 4
    doc_width = size - (doc_padding * 2)
    doc_height = size - (doc_padding * 2)

    # Document body
    draw.rectangle(
        [doc_padding, doc_padding, doc_padding + doc_width, doc_padding + doc_height],
        fill=doc_color,
        outline=None
    )

    # Fold corner (small triangle in top-right)
    fold_size = 4
    fold_points = [
        (size - doc_padding - fold_size, doc_padding),
        (size - doc_padding, doc_padding),
        (size - doc_padding, doc_padding + fold_size)
    ]
    draw.polygon(fold_points, fill=(100, 100, 100, 255))

    # Status dot (bottom-right corner)
    dot_size = 6
    dot_x = size - doc_padding - dot_size - 1
    dot_y = size - doc_padding - dot_size - 1

    draw.ellipse(
        [dot_x, dot_y, dot_x + dot_size, dot_y + dot_size],
        fill=color + (255,),
        outline=None
    )

    # Save the image
    img.save(output_path, 'PNG')
    print(f"Created: {output_path}")

def main():
    """Generate all status icons."""
    # Colors for each state
    colors = {
        'green': (52, 199, 89),    # Apple's system green
        'yellow': (255, 204, 0),   # Apple's system yellow/orange
        'red': (255, 59, 48),      # Apple's system red
    }

    print("Generating menu bar icons...")
    print("Style: Simple circles (clean and minimal)")

    # Generate simple circle icons (recommended for menu bar)
    for color_name, rgb in colors.items():
        output = f"icon-{color_name}-22.png"
        create_circle_icon(rgb, output, size=22)

    print("\nGenerating document-style icons (alternative)...")

    # Generate document icons (alternative style)
    for color_name, rgb in colors.items():
        output = f"icon-{color_name}-doc-22.png"
        create_document_icon(rgb, output, size=22)

    print("\nAll icons generated successfully!")
    print("\nUsage:")
    print("- Use the simple circle icons (icon-*-22.png) for best menu bar visibility")
    print("- Document icons available as alternative if preferred")
    print("- To use in Go: read PNG file and embed in icons.go")

if __name__ == '__main__':
    try:
        from PIL import Image, ImageDraw
        main()
    except ImportError:
        print("Error: PIL/Pillow is required")
        print("Install with: pip3 install Pillow")
        sys.exit(1)
