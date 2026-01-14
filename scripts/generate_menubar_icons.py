#!/usr/bin/env python3
"""
Generate menu bar icons for Legible app.
Design: Tablet with pen/stylus + colored status indicator
"""
from PIL import Image, ImageDraw
import io

def create_icon(status_color):
    """
    Create a 22x22 menu bar icon with tablet+pen design and status dot.

    Args:
        status_color: Tuple (R, G, B) for the status indicator dot

    Returns:
        PNG bytes
    """
    # Create 22x22 image with transparency
    size = (22, 22)
    img = Image.new('RGBA', size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Colors (for template mode, use grayscale for main icon)
    icon_color = (0, 0, 0, 255)  # Black for template (macOS will tint)

    # Draw tablet outline (rounded rectangle)
    # Tablet body: smaller to leave room for status dot
    tablet_rect = [2, 4, 15, 18]
    draw.rounded_rectangle(tablet_rect, radius=2, outline=icon_color, width=1)

    # Draw pen/stylus across the tablet (diagonal line with tip)
    # Pen body
    pen_coords = [(6, 3), (17, 14)]
    draw.line(pen_coords, fill=icon_color, width=2)

    # Pen tip (small circle at end)
    tip_pos = (17, 14)
    draw.ellipse([tip_pos[0]-1, tip_pos[1]-1, tip_pos[0]+1, tip_pos[1]+1],
                 fill=icon_color)

    # Draw status indicator dot in top-right corner
    dot_center = (18, 5)
    dot_radius = 3
    draw.ellipse([dot_center[0]-dot_radius, dot_center[1]-dot_radius,
                  dot_center[0]+dot_radius, dot_center[1]+dot_radius],
                 fill=status_color,
                 outline=icon_color,
                 width=1)

    # Convert to PNG bytes
    byte_arr = io.BytesIO()
    img.save(byte_arr, format='PNG')
    return byte_arr.getvalue()


def bytes_to_go_array(png_bytes):
    """Convert PNG bytes to Go byte array format."""
    hex_bytes = ', '.join(f'0x{b:02X}' for b in png_bytes)

    # Format as Go code with proper wrapping
    lines = []
    line = "\t\t"
    for i, byte in enumerate(png_bytes):
        line += f'0x{byte:02X}, '
        if (i + 1) % 12 == 0:  # 12 bytes per line
            lines.append(line.rstrip())
            line = "\t\t"

    if line.strip():
        lines.append(line.rstrip(', '))

    return '\n'.join(lines)


def main():
    # Color definitions (Apple system colors)
    colors = {
        'green': (52, 199, 89, 255),    # System green
        'yellow': (255, 204, 0, 255),   # System yellow
        'red': (255, 59, 48, 255),      # System red
    }

    print("Generating menu bar icons...\n")

    for name, color in colors.items():
        png_bytes = create_icon(color)
        go_array = bytes_to_go_array(png_bytes)

        print(f"// icon{name.capitalize()} - {len(png_bytes)} bytes")
        print(f"func icon{name.capitalize()}() []byte {{")
        print("\treturn []byte{")
        print(go_array)
        print("\t}")
        print("}\n")

        # Also save to file for inspection
        with open(f'/tmp/menubar_icon_{name}.png', 'wb') as f:
            f.write(png_bytes)
        print(f"Saved to: /tmp/menubar_icon_{name}.png")


if __name__ == '__main__':
    main()
