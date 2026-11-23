package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"s7-trend-go/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func init() {
	application.RegisterEvent[backend.PollUpdate]("poll_update")
	application.RegisterEvent[backend.PollTiming]("poll_timing")
	application.RegisterEvent[application.Void]("app:request-exit")
}

func main() {
	appService := backend.NewApp()

	wailsApp := application.New(application.Options{
		Name:        "S7 Trend Go",
		Description: "S7 PLC Trend App built with Go and Wails",
		Icon:        trayIcon,
		Services: []application.Service{
			application.NewService(appService),
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

	host := newWailsHost(wailsApp, mainWindow)
	backend.AttachHost(appService, host)
	backend.SetTrayEnabled(appService, true)

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if backend.HandleWindowClosing(appService) {
			e.Cancel()
		}
	})

	systemTray := wailsApp.SystemTray.New()
	systemTray.SetIcon(trayIcon)
	systemTray.SetTooltip("S7 Trend Go")

	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("Restore").OnClick(func(_ *application.Context) {
		backend.RestoreWindow(appService)
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

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
