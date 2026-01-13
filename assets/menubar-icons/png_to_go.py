#!/usr/bin/env python3
"""
Convert PNG files to Go byte arrays for embedding in source code.
"""

import sys
import os

def png_to_go_bytes(png_path):
    """Convert PNG file to Go byte array format."""
    with open(png_path, 'rb') as f:
        data = f.read()

    # Format as Go byte array
    bytes_per_line = 12
    lines = []

    for i in range(0, len(data), bytes_per_line):
        chunk = data[i:i+bytes_per_line]
        hex_values = ', '.join(f'0x{b:02X}' for b in chunk)
        lines.append(f'\t\t{hex_values},')

    return '\n'.join(lines)

def main():
    icons = {
        'green': 'icon-green-22.png',
        'yellow': 'icon-yellow-22.png',
        'red': 'icon-red-22.png',
    }

    print("// Generated icon data - DO NOT EDIT MANUALLY")
    print("// Generated from assets/menubar-icons/*.png")
    print()

    for color, filename in icons.items():
        if not os.path.exists(filename):
            print(f"Error: {filename} not found", file=sys.stderr)
            continue

        print(f"// icon{color.capitalize()} returns a {color} status icon.")
        print(f"// 22x22 PNG with transparency, optimized for macOS menu bar.")
        print(f"func icon{color.capitalize()}() []byte {{")
        print(f"\treturn []byte{{")

        bytes_str = png_to_go_bytes(filename)
        print(bytes_str)

        print(f"\t}}")
        print(f"}}")
        print()

if __name__ == '__main__':
    main()
