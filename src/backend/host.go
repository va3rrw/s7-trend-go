package backend

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
	Notify(title, body string)
}

// AttachHost connects a DesktopHost implementation to app.
func AttachHost(a *App, host DesktopHost) {
	a.mu.Lock()
	a.host = host
	a.mu.Unlock()
	a.refreshTrayStatus()
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
// menu and notifies the frontend so reactive controls stay synchronized.
func ToggleSamplingFromTray(a *App) {
	var nextSampling bool
	if a.IsPolling() {
		a.StopPolling()
		nextSampling = false
	} else {
		a.StartPolling(a.GetSettings())
		nextSampling = true
	}

	if host := a.getHost(); host != nil {
		host.EmitEvent("app:sampling-state", SamplingStateEvent{IsSampling: nextSampling})
	}
}

// LoadSettingsFromPath loads settings from a filesystem path (such as a
// dropped .json file or second-instance CLI argument), restarts active polling
// if needed, and notifies the frontend.
func LoadSettingsFromPath(a *App, filePath string) error {
	settings, err := a.loadSettingsFromFilePath(filePath)
	host := a.getHost()
	if err != nil {
		if host != nil {
			host.EmitEvent("settings:load-error", err.Error())
		}
		return err
	}

	if a.IsPolling() {
		a.StartPolling(*settings)
	} else {
		a.refreshTrayStatus()
	}

	if host != nil {
		host.EmitEvent("settings:loaded", *settings)
	}
	return nil
}

// NotifySettingsLoadError emits a settings load error message to the frontend.
func NotifySettingsLoadError(a *App, message string) {
	if host := a.getHost(); host != nil {
		host.EmitEvent("settings:load-error", message)
	}
}
