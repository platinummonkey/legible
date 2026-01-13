# Menu Bar Status Icons

macOS menu bar icons for the Legible sync application.

## Overview

Simple circular status indicators optimized for the macOS menu bar. The icons use Apple's system colors for visual consistency with macOS design patterns.

## Icon Specifications

- **Format**: PNG with transparency (RGBA)
- **Size**: 22x22 pixels (standard macOS menu bar icon size)
- **Style**: Simple filled circles with padding
- **Theme Compatibility**: Visible in both light and dark modes

## Color Palette

Icons use Apple's standard system colors for consistency:

| State | Color | RGB Values | Meaning |
|-------|-------|------------|---------|
| Green | Apple System Green | (52, 199, 89) | Idle/sync complete, no errors |
| Yellow | Apple System Yellow | (255, 204, 0) | Actively syncing/processing |
| Red | Apple System Red | (255, 59, 48) | Error state, sync failed, daemon offline |

## Design Rationale

### Why Circles?

1. **Simplicity**: Circles are instantly recognizable at small sizes
2. **Universal**: No language or cultural interpretation needed
3. **macOS Convention**: Many menu bar apps use circular indicators
4. **Clarity**: Works well in both light and dark themes

### Why These Colors?

- **System Colors**: Using Apple's standard colors ensures:
  - Consistency with macOS design language
  - Accessibility (tested for color blindness)
  - Familiarity to macOS users
  - Automatic adaptation to display profiles

### Why Not Template Images?

Template images (monochrome, theme-adaptive) are ideal for most menu bar icons, but we need **specific colors** to convey status at a glance. Users should be able to see green/yellow/red without opening the menu.

## Alternative Designs

The `generate_icons.py` script also generates document-style icons with status badges:
- `icon-*-doc-22.png` - Document outline with colored status dot

These are available as alternatives if the simple circles feel too minimal.

## Regenerating Icons

If you need to regenerate or modify the icons:

```bash
cd assets/menubar-icons

# Install dependencies (if not already installed)
pip3 install Pillow

# Generate icons
python3 generate_icons.py

# Convert to Go code
python3 png_to_go.py > icons_generated.go

# Copy the function bodies to internal/menubar/icons.go
```

## Files

- `generate_icons.py` - Icon generation script
- `png_to_go.py` - PNG to Go byte array converter
- `icon-green-22.png` - Green status icon
- `icon-yellow-22.png` - Yellow status icon
- `icon-red-22.png` - Red status icon
- `icon-*-doc-22.png` - Alternative document-style icons

## Usage in Code

Icons are embedded as byte arrays in `internal/menubar/icons.go`:

```go
// Get icon for current state
var icon []byte
switch state {
case StateIdle:
    icon = iconGreen()
case StateSyncing:
    icon = iconYellow()
case StateError:
    icon = iconRed()
}

// Set in menu bar
systray.SetIcon(icon)
```

## Visual Examples

### Icon Sizes

```
┌───┐
│ ● │  22x22 pixels (actual size in menu bar)
└───┘
```

### Icon Padding

```
┌────────────────────┐
│   ┌──────────┐    │  22px total
│   │          │    │
│   │    ●     │    │  16px circle
│   │          │    │
│   └──────────┘    │  3px padding on all sides
└────────────────────┘
```

## Design Iterations

**v1.0** (Current):
- Simple filled circles
- 22x22 pixels
- 3px padding
- Apple system colors

**Future Considerations**:
- Animated syncing icon (optional)
- Badge with sync count (if requested)
- Custom icon set for branding

## Accessibility

- **Color Blind Safe**: Colors chosen from Apple's palette which is tested for accessibility
- **High Contrast**: Good visibility in both light and dark modes
- **Size**: 22x22 is standard and readable on all macOS displays

## References

- [Apple Human Interface Guidelines - Menu Bar](https://developer.apple.com/design/human-interface-guidelines/macos/menus/menu-bar-menus/)
- [macOS Menu Bar Icon Design](https://developer.apple.com/design/human-interface-guidelines/macos/icons-and-images/system-icons/)
- [Apple System Colors](https://developer.apple.com/design/human-interface-guidelines/ios/visual-design/color/)
