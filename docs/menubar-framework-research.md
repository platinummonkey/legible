# macOS Menu Bar App Framework Research

**Research Date:** 2026-01-13
**Issue:** remarkable-sync-579

## Executive Summary

For building a macOS menu bar application in Go, there are five viable approaches ranging from cross-platform libraries to native macOS-specific solutions. All require cgo. **Recommendation: fyne.io/systray** for its maintenance, simplicity, and cross-platform potential.

---

## Option 1: fyne.io/systray ⭐ RECOMMENDED

### Overview
A fork of getlantern/systray that removes GTK dependencies while maintaining cross-platform support.

### Pros
- **Active maintenance** (latest v1.12.0, Dec 2025)
- **No GTK dependency** (lighter than original systray)
- **Cross-platform** (Windows, macOS, Linux, BSD)
- **Simple API** - straightforward menu creation
- **Well-documented** with good examples
- **Modern Linux support** via DBus
- **Integration with Fyne UI** (if we want a full GUI later)

### Cons
- Requires cgo (CGO_ENABLED=1)
- Less macOS-native feel (good enough for status indicators)
- Limited to basic menu items (no complex UI)

### Requirements
- Go 1.12+
- cgo enabled
- macOS: Xcode Command Line Tools
- Must bundle in .app structure with Info.plist

### GitHub Stats
- Active development (Fyne team maintains it)
- Part of larger Fyne ecosystem

### Code Example
```go
import "fyne.io/systray"

func main() {
    systray.Run(onReady, onExit)
}

func onReady() {
    systray.SetIcon(iconGreen)
    systray.SetTitle("Legible")
    systray.SetTooltip("reMarkable Sync")

    mStart := systray.AddMenuItem("Start Sync", "Begin syncing")
    mQuit := systray.AddMenuItemCheckbox("Quit", "Exit", false)

    go func() {
        for {
            select {
            case <-mStart.ClickedCh:
                // Handle start
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}
```

---

## Option 2: menuet (macOS-specific)

### Overview
macOS-only library for NSStatusBar applications with native feel.

### Pros
- **Pure macOS focus** - uses native NSStatusBar APIs
- **Clean, Go-idiomatic API**
- **Dynamic menu updates** easy to implement
- **Native alerts and notifications**
- **No Linux/GTK dependencies**
- **301 stars** - decent community

### Cons
- **macOS-only** (no cross-platform)
- **API still changing rapidly** (breaking changes possible)
- **Less active maintenance** (115 commits total)
- Requires cgo
- Smaller community than systray

### Requirements
- macOS/OS X only
- cgo enabled
- Xcode Command Line Tools

### Code Example
```go
import "github.com/caseymrm/menuet"

func main() {
    app := menuet.App()
    app.Label = "com.legible.menubar"

    app.Children = func() []menuet.MenuItem {
        return []menuet.MenuItem{
            {
                Text: "Sync Status: " + getStatus(),
            },
            {
                Text: "Start Sync",
                Clicked: func() {
                    startSync()
                },
            },
        }
    }

    app.RunApplication()
}
```

---

## Option 3: getlantern/systray (original)

### Overview
Original cross-platform systray library with wide adoption.

### Pros
- **Mature** (3.6k stars, 506 forks)
- **Cross-platform** (Windows, macOS, Linux)
- **Well-tested** in production apps (Lantern VPN)
- **Most functions callable from any goroutine**
- **Checkable menu items** on Windows and macOS

### Cons
- **GTK dependency on Linux** (heavier build requirements)
- Requires cgo
- **Superseded by fyne.io/systray fork** (no reason to use original)
- Linux: requires gtk3 and libayatana-appindicator3 dev headers

### Requirements
- cgo enabled
- Linux: gcc, gtk3, libayatana-appindicator3
- macOS: Xcode Command Line Tools, .app bundle structure

### Verdict
**Use fyne.io/systray instead** - same API, fewer dependencies.

---

## Option 4: DarwinKit (formerly MacDriver)

### Overview
Native Objective-C bridge for Go, providing direct access to ~200 Apple frameworks.

### Pros
- **Most native approach** - direct Objective-C API access
- **5.4k stars** - strong community interest
- **Active development** (v0.5.0, July 2024)
- **Full macOS API access** (AppKit, Foundation, WebKit, CoreML)
- **Superior flexibility** - can build complex native UIs
- **20 contributors** - healthy project

### Cons
- **Complex API** - requires understanding Objective-C patterns
- **Memory management challenges** (hybrid Go/Objective-C)
- **Requires XCode** for framework headers
- **Potential segfaults** from framework exceptions
- **Main thread dispatch required** for GUI operations
- **Overkill for simple menu bar app**

### Requirements
- Go 1.18+
- XCode (full IDE, not just Command Line Tools)
- cgo enabled
- Objective-C knowledge helpful

### Use Case
Best for complex native macOS apps. **Too heavy for our simple status menu.**

---

## Option 5: trayhost

### Overview
Cross-platform tray library targeting web-based UIs.

### Pros
- Cross-platform (Windows, macOS, Linux/GTK+3)
- Simple API
- 283 stars

### Cons
- **Potentially inactive** (no recent activity visible)
- **Limited API** - very basic features
- **Must embed icons as byte arrays** (tooling required)
- **No cross-compilation support**
- Requires cgo
- **Fewer features than systray alternatives**

### Requirements
- cgo on all platforms
- Linux: GTK+ 3.0 dev headers
- Windows: MinGW
- macOS: Xcode Command Line Tools

### Verdict
**Not recommended** - systray alternatives are better maintained and featured.

---

## Comparison Matrix

| Feature | fyne.io/systray | menuet | getlantern/systray | DarwinKit | trayhost |
|---------|----------------|---------|-------------------|-----------|----------|
| **Platform** | Cross | macOS only | Cross | macOS only | Cross |
| **Maintenance** | Active (2025) | Moderate | Active | Active (2024) | Low/Inactive |
| **cgo Required** | Yes | Yes | Yes | Yes | Yes |
| **API Complexity** | Simple | Simple | Simple | Complex | Simple |
| **Native Feel** | Good | Excellent | Good | Excellent | Basic |
| **Dependencies** | Minimal | Minimal | GTK (Linux) | XCode | GTK (Linux) |
| **Stars** | N/A (Fyne) | 301 | 3.6k | 5.4k | 283 |
| **Best For** | Simple menu bars | macOS-only apps | Legacy projects | Complex native UIs | Basic needs |

---

## Recommendation: fyne.io/systray

### Why fyne.io/systray?

1. **Actively maintained** by the Fyne team (Dec 2025 release)
2. **Simple, proven API** - easy to implement green/yellow/red status
3. **Cross-platform** - if we ever want Linux support, it's there
4. **No GTK baggage** - lighter than original systray
5. **Good enough native feel** for a status indicator
6. **Future extensibility** - can integrate full Fyne UI if needed
7. **Strong ecosystem** - part of larger Go UI project

### Implementation Plan

1. Use `fyne.io/systray` for the menu bar application
2. Create three icon sets (green/yellow/red) for status
3. Simple menu with:
   - Status text (last sync time, document count)
   - Start/Stop Sync actions
   - Open output directory
   - Preferences
   - Quit
4. Connect to daemon via HTTP or Unix socket (from remarkable-sync-u9j)
5. Poll for status updates every 2-5 seconds
6. Bundle in macOS .app structure for proper integration

### When to Reconsider

- If **macOS-native feel becomes critical** → use menuet
- If **complex native UI needed** → use DarwinKit
- If **avoiding cgo becomes possible** → revisit in future (unlikely)

---

## Build Considerations

All options require cgo, which means:
- **Build time impact** - slower builds, cross-compilation complex
- **Dependencies** - Xcode Command Line Tools on macOS
- **Distribution** - must bundle in .app for macOS integration
- **CI/CD** - GitHub Actions needs proper macOS runners

The project already uses cgo-dependent libraries (pdfcpu likely uses cgo), so this doesn't add new constraints.

---

## Sources

- [fyne.io/systray - Go Packages](https://pkg.go.dev/fyne.io/systray)
- [getlantern/systray - GitHub](https://github.com/getlantern/systray)
- [caseymrm/menuet - GitHub](https://github.com/caseymrm/menuet)
- [progrium/darwinkit - GitHub](https://github.com/progrium/darwinkit)
- [cratonica/trayhost - GitHub](https://github.com/cratonica/trayhost)
- [Using Go in Native macOS Apps with MacDriver - InfoQ](https://www.infoq.com/news/2021/02/macdriver-go-objc-interop/)
- [Use Mac APIs and build Mac apps with Go - DEV](https://dev.to/progrium/use-mac-apis-and-build-mac-apps-with-go-ap6)

---

**Next Steps:** Proceed with remarkable-sync-07k (Implement macOS menu bar application core) using fyne.io/systray.
