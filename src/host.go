package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"s7-trend-go/backend"
)

type wailsHost struct {
	app                *application.App
	window             *application.WebviewWindow
	tray               *application.SystemTray
	trayMenu           *application.Menu
	restoreItem        *application.MenuItem
	toggleSamplingItem *application.MenuItem
	webServerItem      *application.MenuItem
	notifier           *safeNotificationService
	trayMu             sync.Mutex
	isSampling         bool
	connectedPlcs      int
	totalPlcs          int
	webRunning         bool
	webAddr            string
	lastTooltip        string
	lastToggleLabel    string
	lastWebLabel       string
	lastWebChecked     bool
	webItemInit        bool
	lastRestoreEnabled bool
	restoreItemInit    bool
}

var _ backend.DesktopHost = (*wailsHost)(nil)

func newWailsHost(app *application.App, window *application.WebviewWindow) *wailsHost {
	return &wailsHost{
		app:    app,
		window: window,
	}
}

func (h *wailsHost) attachTray(tray *application.SystemTray, menu *application.Menu, restoreItem, toggleItem, webServerItem *application.MenuItem) {
	h.trayMu.Lock()
	h.tray = tray
	h.trayMenu = menu
	h.restoreItem = restoreItem
	h.toggleSamplingItem = toggleItem
	h.webServerItem = webServerItem
	h.trayMu.Unlock()
}

func (h *wailsHost) attachNotifier(notifier *safeNotificationService) {
	h.notifier = notifier
}

func (h *wailsHost) updateTrayTooltipLocked() {
	if h.tray == nil {
		return
	}
	tooltip := "S7 Trend Go — Idle"
	if h.isSampling {
		if h.totalPlcs > 0 {
			tooltip = fmt.Sprintf("S7 Trend Go — Sampling (%d/%d PLCs)", h.connectedPlcs, h.totalPlcs)
		} else {
			tooltip = "S7 Trend Go — Sampling"
		}
	}
	if h.webRunning && h.webAddr != "" {
		tooltip += " | Web " + h.webAddr
	}
	if tooltip != h.lastTooltip {
		h.tray.SetTooltip(tooltip)
		h.lastTooltip = tooltip
	}
}

func (h *wailsHost) UpdateTrayStatus(isSampling bool, connectedPlcs, totalPlcs int) {
	label := "Start Sampling"
	if isSampling {
		label = "Stop Sampling"
	}

	h.trayMu.Lock()
	defer h.trayMu.Unlock()

	h.isSampling = isSampling
	h.connectedPlcs = connectedPlcs
	h.totalPlcs = totalPlcs
	h.updateTrayTooltipLocked()

	if h.toggleSamplingItem != nil && h.trayMenu != nil && label != h.lastToggleLabel {
		h.toggleSamplingItem.SetLabel(label)
		h.trayMenu.Update()
		h.lastToggleLabel = label
	}
}

func (h *wailsHost) UpdateTrayWebServer(running bool, addr string) {
	label := "Web Server"
	if running && addr != "" {
		label = fmt.Sprintf("Web Server (%s)", addr)
	}

	h.trayMu.Lock()
	defer h.trayMu.Unlock()

	h.webRunning = running
	h.webAddr = addr
	h.updateTrayTooltipLocked()

	if h.webServerItem != nil && h.trayMenu != nil {
		dirty := false
		if !h.webItemInit || running != h.lastWebChecked {
			h.webServerItem.SetChecked(running)
			h.lastWebChecked = running
			dirty = true
		}
		if label != h.lastWebLabel {
			h.webServerItem.SetLabel(label)
			h.lastWebLabel = label
			dirty = true
		}
		h.webItemInit = true
		if dirty {
			h.trayMenu.Update()
		}
	}
}

func (h *wailsHost) Notify(title, body string) {
	if h.notifier != nil {
		h.notifier.send(title, body)
	}
}

func (h *wailsHost) EmitEvent(name string, data ...any) {
	if h.app != nil {
		h.app.Event.Emit(name, data...)
	}
}

func (h *wailsHost) Quit() {
	if h.app != nil {
		h.app.Quit()
	}
}

func (h *wailsHost) setRestoreEnabled(enabled bool) {
	h.trayMu.Lock()
	defer h.trayMu.Unlock()

	if h.restoreItem == nil || h.trayMenu == nil {
		return
	}
	if !h.restoreItemInit || enabled != h.lastRestoreEnabled {
		h.restoreItem.SetEnabled(enabled)
		h.lastRestoreEnabled = enabled
		h.restoreItemInit = true
		h.trayMenu.Update()
	}
}

func (h *wailsHost) syncTrayRestoreState() {
	if h.window == nil {
		return
	}
	canRestore := !h.window.IsVisible() || h.window.IsMinimised()
	h.setRestoreEnabled(canRestore)
}

func (h *wailsHost) ShowWindow() {
	if h.window == nil {
		return
	}
	h.window.Show()
	if h.window.IsMinimised() {
		h.window.UnMinimise()
	}
	h.window.Focus()
	h.setRestoreEnabled(false)
}

func (h *wailsHost) HideWindow() {
	if h.window != nil {
		h.window.Hide()
		h.setRestoreEnabled(true)
	}
}

func (h *wailsHost) GetWindowBounds() (width, height, x, y int, ok bool) {
	if h.window == nil || !h.window.IsVisible() || h.window.IsMinimised() {
		return 0, 0, 0, 0, false
	}
	width, height = h.window.Size()
	x, y = h.window.Position()
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	return width, height, x, y, true
}

func (h *wailsHost) SetWindowBounds(width, height, x, y int, setPos bool) {
	if h.window == nil || width <= 0 || height <= 0 {
		return
	}
	h.window.SetSize(width, height)
	if setPos {
		h.window.SetPosition(x, y)
	}
}

func (h *wailsHost) OpenFileDialog(title, defaultDir, filterName, filterPattern string) (string, error) {
	if h.app == nil {
		return "", nil
	}
	dlg := h.app.Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(true)
	if defaultDir != "" {
		dlg.SetDirectory(defaultDir)
	}
	if filterName != "" && filterPattern != "" {
		dlg.AddFilter(filterName, filterPattern)
	}
	if h.window != nil {
		dlg.AttachToWindow(h.window)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		if isDialogCancelled(err) {
			return "", nil
		}
		return "", err
	}
	return path, nil
}

func (h *wailsHost) SaveFileDialog(title, defaultDir, defaultFilename, filterName, filterPattern string) (string, error) {
	if h.app == nil {
		return "", nil
	}
	dlg := h.app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:                title,
		Filename:             defaultFilename,
		Directory:            defaultDir,
		CanCreateDirectories: true,
	})
	if filterName != "" && filterPattern != "" {
		dlg.AddFilter(filterName, filterPattern)
	}
	if h.window != nil {
		dlg.AttachToWindow(h.window)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		if isDialogCancelled(err) {
			return "", nil
		}
		return "", err
	}
	return path, nil
}

func isDialogCancelled(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "cancel")
}
