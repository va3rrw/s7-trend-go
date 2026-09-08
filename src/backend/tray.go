package backend

import (
	"context"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/gogpu/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const trayTooltip = "S7 Trend Go"

type trayController struct {
	app  *App
	ctx  context.Context
	icon []byte

	mu            sync.Mutex
	tray          *systray.SystemTray
	done          chan struct{}
	stopRequested bool
}

func newTrayController(app *App, ctx context.Context, icon []byte) *trayController {
	return &trayController{
		app:  app,
		ctx:  ctx,
		icon: append([]byte(nil), icon...),
		done: make(chan struct{}),
	}
}

func (a *App) startTray(ctx context.Context) {
	a.trayMu.Lock()
	if a.tray != nil {
		a.trayMu.Unlock()
		return
	}
	controller := newTrayController(a, ctx, a.trayIcon)
	a.tray = controller
	a.trayMu.Unlock()
	controller.start()
}

func (a *App) stopTray() {
	a.trayMu.Lock()
	controller := a.tray
	a.tray = nil
	a.trayMu.Unlock()
	if controller != nil {
		controller.stop()
	}
}

func (t *trayController) start() {
	go t.run()
}

func (t *trayController) run() {
	// Win32 message windows and their message loops are thread-affine.
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	tray := systray.New()
	menu := systray.NewMenu()
	menu.Add("Restore", t.restoreWindow)
	menu.AddSeparator()
	menu.Add("Exit", t.exitApp)

	tray.SetIcon(t.icon).
		SetTooltip(trayTooltip).
		SetMenu(menu).
		OnClick(t.restoreWindow).
		OnDoubleClick(t.restoreWindow).
		Show()

	t.mu.Lock()
	if t.stopRequested {
		t.mu.Unlock()
		tray.Remove()
		close(t.done)
		return
	}
	t.tray = tray
	t.mu.Unlock()

	_ = tray.Run()

	t.mu.Lock()
	if t.tray == tray {
		t.tray = nil
	}
	close(t.done)
	t.mu.Unlock()
}

func (t *trayController) stop() {
	t.mu.Lock()
	t.stopRequested = true
	tray := t.tray
	done := t.done
	t.mu.Unlock()

	if tray != nil {
		tray.Remove()
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func (t *trayController) restoreWindow() {
	runtime.WindowShow(t.ctx)
	runtime.WindowUnminimise(t.ctx)
}

func (t *trayController) exitApp() {
	t.app.requestExitFromTray()
}
