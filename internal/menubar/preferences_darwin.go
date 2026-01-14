//go:build darwin
// +build darwin

package menubar

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

void *createPreferencesController(const char *daemonAddr,
                                  const char *syncInterval,
                                  int ocrEnabled,
                                  const char *daemonConfigFile,
                                  void *context);
void showPreferencesWindow(void *controller);
void releasePreferencesController(void *controller);
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/platinummonkey/legible/internal/logger"
)

var (
	preferencesCallbacks = make(map[uintptr]func(string, string, bool, string))
	preferencesCallbackMu sync.Mutex
	preferencesCallbackID uintptr
)

//export preferencesGoCallback
func preferencesGoCallback(cDaemonAddr *C.char, cSyncInterval *C.char, cOCREnabled C.int, cDaemonConfigFile *C.char, context unsafe.Pointer) {
	contextID := uintptr(context)

	preferencesCallbackMu.Lock()
	callback, ok := preferencesCallbacks[contextID]
	delete(preferencesCallbacks, contextID)
	preferencesCallbackMu.Unlock()

	if !ok {
		logger.Error("Preferences callback not found", "context_id", contextID)
		return
	}

	// Convert C strings to Go (must do before goroutine since C strings may be freed)
	daemonAddr := C.GoString(cDaemonAddr)
	syncInterval := C.GoString(cSyncInterval)
	ocrEnabled := (cOCREnabled != 0)
	daemonConfigFile := C.GoString(cDaemonConfigFile)

	// Call the Go callback in a goroutine to avoid blocking Objective-C main thread
	go callback(daemonAddr, syncInterval, ocrEnabled, daemonConfigFile)
}

// ShowNativePreferences shows the native macOS preferences window
func (a *App) ShowNativePreferences() {
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

	// Create callback context
	preferencesCallbackMu.Lock()
	preferencesCallbackID++
	contextID := preferencesCallbackID
	preferencesCallbacks[contextID] = func(daemonAddr, syncInterval string, ocrEnabled bool, daemonConfigFile string) {
		// Update configuration
		a.menuBarConfig.DaemonAddr = daemonAddr
		a.menuBarConfig.SyncInterval = syncInterval
		a.menuBarConfig.OCREnabled = ocrEnabled
		a.menuBarConfig.DaemonConfigFile = daemonConfigFile

		// Save to file
		if err := SaveMenuBarConfig(a.menuBarConfig, a.configPath); err != nil {
			logger.Error("Failed to save configuration", "error", err)
			a.showErrorDialog("Failed to save settings")
			return
		}

		logger.Info("Configuration saved successfully")

		// Show restart prompt
		a.showRestartPrompt()
	}
	preferencesCallbackMu.Unlock()

	// Create controller with callback context
	// The Objective-C code will call preferencesGoCallback directly with this context
	controller := C.createPreferencesController(
		cDaemonAddr,
		cSyncInterval,
		cOCREnabled,
		cDaemonConfigFile,
		unsafe.Pointer(contextID),
	)
	if controller == nil {
		logger.Error("Failed to create preferences controller")
		preferencesCallbackMu.Lock()
		delete(preferencesCallbacks, contextID)
		preferencesCallbackMu.Unlock()
		return
	}

	// Show window (non-blocking, callback will be called when saved)
	C.showPreferencesWindow(controller)

	// Note: We don't release the controller immediately because the window is still open
	// The controller will be released when the window closes naturally
	// This is safe because the window retains the controller
}
