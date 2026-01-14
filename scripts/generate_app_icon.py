#!/usr/bin/env python3
"""
Generate application icon for Legible app.
Creates a larger, more detailed tablet+pen design for the .app bundle icon.
"""
from PIL import Image, ImageDraw
import os

def create_app_icon(size):
    """
    Create an app icon at the specified size.
    Design: Tablet with pen/stylus and subtle details

    Args:
        size: Icon size (square)

    Returns:
        PIL Image
    """
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Scale factor for different sizes
    scale = size / 512.0

    # Colors
    tablet_color = (60, 60, 60, 255)        # Dark gray tablet
    screen_color = (240, 240, 240, 255)     # Light screen
    pen_color = (100, 100, 100, 255)        # Gray pen
    accent_color = (52, 199, 89, 255)       # Green accent (brand color)

    # Padding and dimensions
    padding = int(40 * scale)

    # Tablet outline (rounded rectangle)
    tablet_rect = [
        padding,
        padding,
        size - padding,
        size - padding
    ]
    radius = int(40 * scale)

    # Draw tablet body with shadow effect
    shadow_offset = int(6 * scale)
    shadow_rect = [
        tablet_rect[0] + shadow_offset,
        tablet_rect[1] + shadow_offset,
        tablet_rect[2] + shadow_offset,
        tablet_rect[3] + shadow_offset
    ]
    draw.rounded_rectangle(shadow_rect, radius=radius, fill=(0, 0, 0, 60))

    # Tablet body
    draw.rounded_rectangle(tablet_rect, radius=radius, fill=tablet_color)

    # Screen (inner rectangle)
    screen_padding = int(20 * scale)
    screen_rect = [
        tablet_rect[0] + screen_padding,
        tablet_rect[1] + screen_padding,
        tablet_rect[2] - screen_padding,
        tablet_rect[3] - screen_padding
    ]
    screen_radius = int(20 * scale)
    draw.rounded_rectangle(screen_rect, radius=screen_radius, fill=screen_color)

    # Draw some handwriting strokes on the screen
    stroke_color = (100, 100, 100, 180)
    stroke_width = max(2, int(4 * scale))

    # Wavy line like handwriting
    y_start = screen_rect[1] + int(60 * scale)
    x_start = screen_rect[0] + int(40 * scale)
    x_end = screen_rect[2] - int(40 * scale)

    for i in range(3):
        y = y_start + i * int(50 * scale)
        if y + int(20 * scale) < screen_rect[3]:
            # Wavy line
            points = []
            segments = 20
            for j in range(segments + 1):
                x = x_start + (x_end - x_start) * j / segments
                wave = int(8 * scale) * (1 if (j // 3) % 2 == 0 else -1)
                points.append((x, y + wave))
            draw.line(points, fill=stroke_color, width=stroke_width, joint="curve")

    # Draw pen/stylus across the tablet (larger and more detailed)
    pen_width = int(12 * scale)
    pen_length = int(220 * scale)

    # Pen position (diagonal across)
    pen_start_x = tablet_rect[2] - int(100 * scale)
    pen_start_y = tablet_rect[1] + int(50 * scale)
    pen_end_x = pen_start_x + int(140 * scale)
    pen_end_y = pen_start_y + int(140 * scale)

    # Pen body (thick line)
    draw.line([(pen_start_x, pen_start_y), (pen_end_x, pen_end_y)],
              fill=pen_color, width=pen_width)

    # Pen tip (small circle)
    tip_radius = int(8 * scale)
    draw.ellipse([pen_end_x - tip_radius, pen_end_y - tip_radius,
                  pen_end_x + tip_radius, pen_end_y + tip_radius],
                 fill=(40, 40, 40, 255))

    # Pen grip area (lighter section)
    grip_start_ratio = 0.4
    grip_x = pen_start_x + int((pen_end_x - pen_start_x) * grip_start_ratio)
    grip_y = pen_start_y + int((pen_end_y - pen_start_y) * grip_start_ratio)
    grip_length = int(40 * scale)

    # Brand accent (small green indicator on tablet)
    indicator_size = int(12 * scale)
    indicator_x = tablet_rect[0] + int(30 * scale)
    indicator_y = tablet_rect[1] + int(30 * scale)
    draw.ellipse([indicator_x - indicator_size, indicator_y - indicator_size,
                  indicator_x + indicator_size, indicator_y + indicator_size],
                 fill=accent_color)

    return img


def generate_iconset():
    """Generate all required icon sizes for .icns file."""
    sizes = [16, 32, 64, 128, 256, 512, 1024]
    iconset_dir = "/tmp/Legible.iconset"

    # Create iconset directory
    os.makedirs(iconset_dir, exist_ok=True)

    for size in sizes:
        # Generate regular resolution
        img = create_app_icon(size)
        img.save(f"{iconset_dir}/icon_{size}x{size}.png")
        print(f"Generated icon_{size}x{size}.png")

        # Generate @2x resolution for retina displays (except 1024)
        if size <= 512:
            img_2x = create_app_icon(size * 2)
            img_2x.save(f"{iconset_dir}/icon_{size}x{size}@2x.png")
            print(f"Generated icon_{size}x{size}@2x.png")

    print(f"\nIconset created at: {iconset_dir}")
    print("\nTo convert to .icns, run:")
    print(f"  iconutil -c icns {iconset_dir}")


if __name__ == '__main__':
    generate_iconset()
