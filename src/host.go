package main

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"s7-trend-go/backend"
)

type wailsHost struct {
	app    *application.App
	window *application.WebviewWindow
}

var _ backend.DesktopHost = (*wailsHost)(nil)

func newWailsHost(app *application.App, window *application.WebviewWindow) *wailsHost {
	return &wailsHost{
		app:    app,
		window: window,
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

func (h *wailsHost) ShowWindow() {
	if h.window == nil {
		return
	}
	h.window.Show()
	if h.window.IsMinimised() {
		h.window.UnMinimise()
	}
	h.window.Focus()
}

func (h *wailsHost) HideWindow() {
	if h.window != nil {
		h.window.Hide()
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
