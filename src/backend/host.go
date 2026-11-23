package backend

import "io/fs"

// DesktopHost abstracts desktop runtime capabilities needed by App so package
// backend stays free of GUI framework dependencies and can be tested directly.
type DesktopHost interface {
	EmitEvent(name string, data ...any)
	Quit()
	ShowWindow()
	HideWindow()
	GetWindowBounds() (width, height, x, y int, ok bool)
	SetWindowBounds(width, height, x, y int, setPos bool)
	OpenFileDialog(title, defaultDir, filterName, filterPattern string) (string, error)
	SaveFileDialog(title, defaultDir, defaultFilename, filterName, filterPattern string) (string, error)
	UpdateTrayStatus(isSampling bool, connectedPlcs, totalPlcs int)
	UpdateTrayWebServer(running bool, addr string)
	Notify(title, body string)
}

// AttachHost connects a DesktopHost implementation to app.
func AttachHost(a *App, host DesktopHost) {
	a.mu.Lock()
	a.host = host
	a.mu.Unlock()
	a.refreshTrayStatus()
	a.refreshTrayWebServer()
}

// SetTrayEnabled configures whether closing the window hides to tray instead of exiting.
func SetTrayEnabled(a *App, enabled bool) {
	a.trayMu.Lock()
	a.trayEnabled = enabled
	a.trayMu.Unlock()
}

// InitialWindowState returns the persisted window geometry loaded at startup.
func InitialWindowState(a *App) (width, height, x, y int, posSet bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.windowWidth, a.windowHeight, a.windowX, a.windowY, a.windowPosSet
}

// RestoreWindow asks the attached desktop host to show and focus the main window.
func RestoreWindow(a *App) {
	if host := a.getHost(); host != nil {
		host.ShowWindow()
	}
}

// SaveWindowState persists current window geometry and last-used settings state.
func SaveWindowState(a *App) {
	a.saveWindowState()
}

// HandleWindowClosing handles a window close request and returns true if closing
// should be cancelled (hidden to tray or awaiting exit confirmation).
func HandleWindowClosing(a *App) (prevent bool) {
	a.mu.RLock()
	force := a.forceExit
	host := a.host
	a.mu.RUnlock()

	if force {
		a.saveWindowState()
		return false
	}

	a.trayMu.Lock()
	trayEnabled := a.trayEnabled
	a.trayMu.Unlock()
	if trayEnabled && host != nil {
		// Save bounds before hiding so hidden/invisible windows do not clobber geometry.
		a.saveWindowState()
		host.HideWindow()
		return true
	}

	if !a.HasSettingsChanged() {
		a.saveWindowState()
		return false
	}

	if host != nil {
		host.EmitEvent("app:request-exit")
	}
	return true
}

// RequestExitFromTray handles an Exit command invoked from the system tray menu.
func RequestExitFromTray(a *App) {
	host := a.getHost()
	if host == nil {
		return
	}

	if a.HasSettingsChanged() {
		host.ShowWindow()
		host.EmitEvent("app:request-exit")
		return
	}
	a.QuitApp()
}

// ToggleSamplingFromTray toggles live PLC polling from the native system tray
// menu and relies on StartPolling/StopPolling to emit state transitions.
func ToggleSamplingFromTray(a *App) {
	if a.IsPolling() {
		a.StopPolling()
		return
	}
	a.StartPolling(a.GetSettings())
}

// LoadSettingsFromPath loads settings from a filesystem path (such as a
// dropped .json file or second-instance CLI argument), restarts active polling
// if needed, and notifies both desktop and remote web clients.
func LoadSettingsFromPath(a *App, filePath string) error {
	settings, err := a.loadSettingsFromFilePath(filePath)
	if err != nil {
		a.emitAppEvent("settings:load-error", err.Error())
		return err
	}

	if a.IsPolling() {
		a.StartPolling(*settings)
	} else {
		a.refreshTrayStatus()
	}

	a.emitAppEvent("settings:loaded", *settings)
	return nil
}

// NotifySettingsLoadError emits a settings load error message to clients.
func NotifySettingsLoadError(a *App, message string) {
	a.emitAppEvent("settings:load-error", message)
}

// SetWebAssets configures the embedded frontend asset filesystem served when Web
// Server mode is toggled on.
func SetWebAssets(a *App, fsys fs.FS) {
	a.webMu.Lock()
	a.webAssets = fsys
	a.webMu.Unlock()
}

// StartWebServerIfEnabled starts the embedded web server if it was left enabled
// in the saved application state.
func StartWebServerIfEnabled(a *App) {
	a.webMu.Lock()
	enabled := a.webServerEnabled
	host := a.webServerHost
	port := a.webServerPort
	a.webMu.Unlock()

	if !enabled {
		a.refreshTrayWebServer()
		return
	}
	if _, err := a.StartWebServer(host, port); err != nil {
		if desktop := a.getHost(); desktop != nil {
			desktop.Notify("Web Server Error", err.Error())
		}
	}
}

// ToggleWebServerFromTray toggles the embedded HTTP/WebSocket server from the
// native system tray menu.
func ToggleWebServerFromTray(a *App) {
	if _, err := a.ToggleWebServer(); err != nil {
		if host := a.getHost(); host != nil {
			host.Notify("Web Server Error", err.Error())
		}
	}
}

// StatusSnapshot returns a point-in-time status summary for GET /api/status.
func StatusSnapshot(a *App) ServerStatus {
	a.mu.RLock()
	isPolling := a.isPolling
	pollMs := a.settings.PollIntervalMs
	connectedPlcs, totalPlcs, enabledTags := a.plcConnectionCountsLocked()
	a.mu.RUnlock()

	return ServerStatus{
		Version:        AppVersion,
		IsPolling:      isPolling,
		ConnectedPlcs:  connectedPlcs,
		TotalPlcs:      totalPlcs,
		EnabledTags:    enabledTags,
		PollIntervalMs: pollMs,
		WebServer:      a.GetWebServerStatus(),
	}
}
