package main

import (
	"embed"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"s7-trend-go/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func init() {
	application.RegisterEvent[application.Void]("app:request-exit")
	application.RegisterEvent[backend.AppSettings]("settings:loaded")
	application.RegisterEvent[string]("settings:load-error")
	application.RegisterEvent[backend.SamplingStateEvent]("app:sampling-state")
}

func main() {
	appService := backend.NewApp()
	notifier := newSafeNotificationService(func() {
		backend.RestoreWindow(appService)
	})

	wailsApp := application.New(application.Options{
		Name:        "S7 Trend Go",
		Description: "S7 PLC Trend App built with Go and Wails",
		Icon:        trayIcon,
		Services: []application.Service{
			application.NewService(appService),
			application.NewService(notifier),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "ca.cetus.s7-trend-go",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				backend.RestoreWindow(appService)
				if len(data.Args) > 1 {
					if path := backend.FindSettingsJSONArg(data.Args[1:], data.WorkingDir); path != "" {
						_ = backend.LoadSettingsFromPath(appService, path)
					}
				}
			},
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: false,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			backend.SaveWindowState(appService)
		},
	})

	// If launched initially with a .json settings file argument, load it after
	// single-instance checks pass so secondary processes never overwrite app_state.
	if len(os.Args) > 1 {
		if path := backend.FindSettingsJSONArg(os.Args[1:], ""); path != "" {
			_ = backend.LoadSettingsFromPath(appService, path)
		}
	}

	wailsApp.HandleStream("polling", func(c *application.StreamConn) {
		defer c.Close()
		ch, unsubscribe := appService.SubscribeTelemetry(64)
		defer unsubscribe()

		ctx := c.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-ch:
				if !ok {
					return
				}
				payload, err := json.Marshal(frame)
				if err != nil {
					continue
				}
				if err := c.TrySend(payload); err != nil {
					if errors.Is(err, application.ErrStreamFull) {
						continue
					}
					return
				}
			}
		}
	})

	width, height, x, y, posSet := backend.InitialWindowState(appService)
	if width <= 0 || height <= 0 {
		width = 1440
		height = 900
	}

	windowOptions := application.WebviewWindowOptions{
		Name:             "main",
		Title:            "S7 Trend Go v" + backend.AppVersion,
		Width:            width,
		Height:           height,
		MinWidth:         1280,
		MinHeight:        800,
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGBA(245, 245, 247, 255),
		URL:              "/",
		Windows: application.WindowsWindow{
			DisableIcon:                       false,
			DisableFramelessWindowDecorations: false,
			Theme:                             application.Light,
		},
	}
	if posSet {
		windowOptions.InitialPosition = application.WindowXY
		windowOptions.X = x
		windowOptions.Y = y
	}

	mainWindow := wailsApp.Window.NewWithOptions(windowOptions)

	systemTray := wailsApp.SystemTray.New()
	systemTray.SetIcon(trayIcon)
	systemTray.SetTooltip("S7 Trend Go — Idle")

	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("Restore").OnClick(func(_ *application.Context) {
		backend.RestoreWindow(appService)
	})
	toggleSamplingItem := trayMenu.Add("Start Sampling")
	toggleSamplingItem.OnClick(func(_ *application.Context) {
		backend.ToggleSamplingFromTray(appService)
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Exit").OnClick(func(_ *application.Context) {
		backend.RequestExitFromTray(appService)
	})
	systemTray.SetMenu(trayMenu)
	systemTray.OnClick(func() {
		backend.RestoreWindow(appService)
	})
	systemTray.OnDoubleClick(func() {
		backend.RestoreWindow(appService)
	})

	host := newWailsHost(wailsApp, mainWindow)
	host.attachTray(systemTray, trayMenu, toggleSamplingItem)
	host.attachNotifier(notifier)
	backend.AttachHost(appService, host)
	backend.SetTrayEnabled(appService, true)

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if backend.HandleWindowClosing(appService) {
			e.Cancel()
		}
	})

	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		for _, filePath := range files {
			if strings.EqualFold(filepath.Ext(filePath), ".json") {
				_ = backend.LoadSettingsFromPath(appService, filePath)
				return
			}
		}
		backend.NotifySettingsLoadError(appService, "Only .json settings files are supported")
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
