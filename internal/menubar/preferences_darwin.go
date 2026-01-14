//go:build darwin
// +build darwin

package menubar

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

typedef struct {
    const char *daemonAddr;
    const char *syncInterval;
    int ocrEnabled;
    const char *daemonConfigFile;
    int saved;
} PreferencesResult;

void *createPreferencesController(const char *daemonAddr,
                                  const char *syncInterval,
                                  int ocrEnabled,
                                  const char *daemonConfigFile);
void showPreferencesWindow(void *controller);
int isPreferencesWindowVisible(void *controller);
PreferencesResult getPreferencesResult(void *controller);
void releasePreferencesController(void *controller);
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/platinummonkey/legible/internal/logger"
)

// ShowNativePreferences shows the native macOS preferences window
func (a *App) ShowNativePreferences() bool {
	// Convert Go strings to C strings
	cDaemonAddr := C.CString(a.menuBarConfig.DaemonAddr)
	cSyncInterval := C.CString(a.menuBarConfig.SyncInterval)
	cDaemonConfigFile := C.CString(a.menuBarConfig.DaemonConfigFile)
	defer C.free(unsafe.Pointer(cDaemonAddr))
	defer C.free(unsafe.Pointer(cSyncInterval))
	defer C.free(unsafe.Pointer(cDaemonConfigFile))

	cOCREnabled := C.int(0)
	if a.menuBarConfig.OCREnabled {
		cOCREnabled = C.int(1)
	}

	// Create controller
	controller := C.createPreferencesController(
		cDaemonAddr,
		cSyncInterval,
		cOCREnabled,
		cDaemonConfigFile,
	)
	if controller == nil {
		logger.Error("Failed to create preferences controller")
		return false
	}
	defer C.releasePreferencesController(controller)

	// Show window (non-blocking)
	C.showPreferencesWindow(controller)

	// Wait for window to close
	for C.isPreferencesWindowVisible(controller) != 0 {
		time.Sleep(100 * time.Millisecond)
	}

	// Get result
	result := C.getPreferencesResult(controller)
	defer func() {
		C.free(unsafe.Pointer(result.daemonAddr))
		C.free(unsafe.Pointer(result.syncInterval))
		C.free(unsafe.Pointer(result.daemonConfigFile))
	}()

	// Check if saved
	if result.saved == 0 {
		logger.Info("Preferences canceled by user")
		return false
	}

	// Update configuration
	a.menuBarConfig.DaemonAddr = C.GoString(result.daemonAddr)
	a.menuBarConfig.SyncInterval = C.GoString(result.syncInterval)
	a.menuBarConfig.OCREnabled = (result.ocrEnabled != 0)
	a.menuBarConfig.DaemonConfigFile = C.GoString(result.daemonConfigFile)

	// Save to file
	if err := SaveMenuBarConfig(a.menuBarConfig, a.configPath); err != nil {
		logger.Error("Failed to save configuration", "error", err)
		a.showErrorDialog("Failed to save settings")
		return false
	}

	logger.Info("Configuration saved successfully")
	return true
}
