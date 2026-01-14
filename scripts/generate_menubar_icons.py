#!/usr/bin/env python3
"""
Generate menu bar icons for Legible app - Dark mode compatible version.
Design: Tablet with pen/stylus + colored status indicator
The main icon is template-compatible (black with alpha) for macOS tinting
Status dots have white halos for dark mode visibility
"""
from PIL import Image, ImageDraw
import io

def create_icon(status_color):
    """
    Create a 22x22 menu bar icon with tablet+pen design and status dot.
    Template-compatible: Main icon in black, macOS will tint for light/dark mode
    Status dot uses color with white halo for visibility in dark mode

    Args:
        status_color: Tuple (R, G, B) for the status indicator dot

    Returns:
        PNG bytes
    """
    # Create 22x22 image with transparency
    size = (22, 22)
    img = Image.new('RGBA', size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Template color (black - macOS will tint this for light/dark mode)
    icon_color = (0, 0, 0, 255)

    # Draw tablet outline (rounded rectangle)
    tablet_rect = [2, 4, 15, 18]
    draw.rounded_rectangle(tablet_rect, radius=2, outline=icon_color, width=1)

    # Draw pen/stylus across the tablet (diagonal line with tip)
    pen_coords = [(6, 3), (17, 14)]
    draw.line(pen_coords, fill=icon_color, width=2)

    # Pen tip (small circle at end)
    tip_pos = (17, 14)
    draw.ellipse([tip_pos[0]-1, tip_pos[1]-1, tip_pos[0]+1, tip_pos[1]+1],
                 fill=icon_color)

    # Draw status indicator dot with white halo for dark mode visibility
    dot_center = (18, 5)
    dot_radius = 3

    # White halo (slightly larger, semi-transparent for blending)
    halo_radius = dot_radius + 1
    draw.ellipse([dot_center[0]-halo_radius, dot_center[1]-halo_radius,
                  dot_center[0]+halo_radius, dot_center[1]+halo_radius],
                 fill=(255, 255, 255, 180))

    # Colored status dot on top
    draw.ellipse([dot_center[0]-dot_radius, dot_center[1]-dot_radius,
                  dot_center[0]+dot_radius, dot_center[1]+dot_radius],
                 fill=status_color)

    # Thin black outline for definition
    draw.ellipse([dot_center[0]-dot_radius, dot_center[1]-dot_radius,
                  dot_center[0]+dot_radius, dot_center[1]+dot_radius],
                 outline=icon_color,
                 width=1)

    # Convert to PNG bytes
    byte_arr = io.BytesIO()
    img.save(byte_arr, format='PNG')
    return byte_arr.getvalue()


def bytes_to_go_array(png_bytes):
    """Convert PNG bytes to Go byte array format."""
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
    # Color definitions (vibrant colors that work in both light and dark mode)
    colors = {
        'green': (52, 199, 89, 255),    # Apple system green
        'yellow': (255, 214, 10, 255),  # Bright yellow (more vibrant)
        'red': (255, 69, 58, 255),      # Apple system red
    }

    print("Generating dark mode compatible menu bar icons...\n")

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
        with open(f'/tmp/menubar_icon_{name}_v2.png', 'wb') as f:
            f.write(png_bytes)
        print(f"Saved to: /tmp/menubar_icon_{name}_v2.png")


if __name__ == '__main__':
    main()
