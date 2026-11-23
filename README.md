# S7 Trend Go

Siemens S7 PLC data acquisition and trend visualization desktop application built with Go, Wails v3, and Vue 3.

## Overview

S7 Trend Go connects to Siemens S7 PLCs (S7-300, S7-400, S7-1200, S7-1500) over Ethernet using native S7 protocol communication. It records real-time tag data and plots multi-axis trend charts.

Based on a very old but useful tool **S7 Trend Values**.

## Key Features

- **Multi-PLC Communication**: Connect to multiple S7 PLCs simultaneously with configurable IP address, Rack, and Slot settings.
- **S7 Tag Support**: Read DB, Memory (M), Inputs (I), and Outputs (Q) memory areas supporting `Bool`, `Byte`, `Word`, `Int`, `DWord`, `DInt`, `Real`, `LReal`, and `String` data types.
- **High-Performance Real-Time Charting**: High-frequency polling (10ms to 60,000ms) with customizable time windows (30s to 86,400s), delivering batched tag and timing frames over a Wails v3 `JSONStream` without main-thread event bridge overhead.
- **Multi-Axis Y Scaling**: Configure independent Y-axes with automatic scaling or explicit minimum and maximum range bounds.
- **Sample Interpolation**: Choose between `Line` and `Step` interpolation modes for trend visualization.
- **Import, Drag-and-Drop & Export**: Save and reload PLC configurations (including dragging `.json` settings files onto the window or passing a `.json` file path via launch arguments), tag lists, Y-axes setups, and exported CSV sample records.
- **Cancellable Trend History**: Store timestamped samples in a local SQLite database with indexed range queries backed by Go `context.Context` + Wails v3 `CancellablePromise` so rapid pan/zoom cancels superseded queries automatically.
- **Per-Tag Storage Cadence**: Numeric tags can use a slower storage interval; Bool tags store only their initial state and changes.
- **Single Instance & Dynamic System Tray**: Ensures a single desktop instance (restoring/focusing the existing window and loading any `.json` CLI arg on relaunch). Closing the window hides to the Windows system tray via Wails v3 native `SystemTray` with **Restore**, live **Start/Stop Sampling** toggle, **Exit**, and a live PLC connection status tooltip.
- **Native Windows Toast Alerts**: Sends Windows Toast notifications on PLC connection loss and recovery transitions via Wails v3 `notifications`, focusing the main window when clicked.
- **Localization**: Built-in support for English (`en`) and Simplified Chinese (`zh`).

## Architecture & Tech Stack

- **Backend**: Go 1.25+, [Wails v3](https://v3.wails.io/), [gos7](https://github.com/robinson/gos7) S7 protocol library.
- **Frontend**: Vue 3 (TypeScript), Vite, UnoCSS, HTML5 Canvas trend rendering, Vue I18n, `@wailsio/runtime`.
- **Target OS**: Windows (`windows/amd64`).

## Trend Data Storage

Trend history is stored in `history.db` under the same default application configuration directory used by `app_state.json`. The directory and database schema are created automatically on first launch, so the standalone executable does not need an installer or a separate database service.

SQLite was selected for this desktop workload because it is embedded, file-based, transactional, and supports indexed time-range queries without requiring a server. Samples use Unix millisecond timestamps and are indexed by tag and timestamp. The frontend requests history through the Wails history API, which reads from SQLite rather than an in-memory ring buffer.

Each tag has a **Storage interval (ms)** setting. `0` follows the global PLC poll interval. A numeric tag interval cannot be lower than the global poll interval. Bool tags ignore repeated values and persist only the first observed state and subsequent transitions.

## System Tray

The standalone application starts with a native Wails v3 system tray icon using the same source artwork as the application icon. Closing the main window hides it while PLC polling and history storage continue in the background. Left-clicking or double-clicking the tray icon, launching a second instance, clicking a Windows Toast notification, or choosing **Restore** shows the window again. You can also toggle **Start Sampling** / **Stop Sampling** directly from the tray menu, inspect live connection counts in the tray tooltip, or choose **Exit** (using the app's themed exit confirmation when settings are dirty).

## Prerequisites

- **Go**: 1.25 or later
- **Node.js**: 22+ and **pnpm** package manager
- **Wails v3 CLI**: `v3.1.0` or later (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)

## Getting Started

### Development Mode

Run the application in live-reload development mode:

```bash
cd src
wails3 dev
```

This starts the Go backend and Vite development server, launching the application window with live reloading and automatic TypeScript bindings regeneration.

### Building for Production

Use the included build script to compile the versioned standalone Windows executable:

```bash
./build.sh
```

The script syncs the version string from the `VERSION` file into `src/backend/version.go`, `src/frontend/src/version.ts`, `src/build/config.yml`, `src/build/windows/info.json`, and `src/build/windows/wails.exe.manifest`, generates Wails v3 TypeScript bindings, runs frontend & Go tests, compiles the binary via `wails3 build`, and outputs the final `.exe` into the `dist/` directory (for example, `dist/s7-trend-go-v1.3.4.exe`).

## Project Layout

```text
s7-trend-go/
├── VERSION                   # Single source of truth for app version
├── build.sh                  # Build script for Windows executable
├── dist/                     # Destination folder for compiled binaries
├── src/
│   ├── main.go               # Wails v3 entry point, stream, single-instance & tray setup
│   ├── host.go               # Wails v3 DesktopHost adapter
│   ├── notifications_service.go # Resilient Windows Toast notification wrapper
│   ├── Taskfile.yml          # Wails v3 root build tasks
│   ├── build/                # Wails v3 build config, platform tasks & Windows metadata
│   ├── backend/              # Go backend packages
│   │   ├── app.go            # App lifecycle, tray status & cancellable RPC methods
│   │   ├── host.go           # DesktopHost interface & close/tray/file coordination
│   │   ├── telemetry.go      # Non-blocking TelemetryFrame pub/sub hub for streams
│   │   ├── app_polling.go    # S7 PLC polling, stream batching & alert transitions
│   │   ├── app_files.go      # Config import/export, drop/arg loading & CSV export
│   │   ├── codec.go          # S7 byte encoding/decoding logic
│   │   └── models.go         # Data structures and settings schemas
│   └── frontend/             # Vue 3 frontend project
│       ├── bindings/         # Generated Wails v3 TypeScript bindings
│       ├── src/
│       │   ├── App.vue       # Main UI component
│       │   ├── chart.ts      # Trend charting & canvas rendering
│       │   ├── components/   # Dialogs (PLC, Tags, Axes, About)
│       │   └── locales/      # i18n translation files (en, zh)
│       └── package.json
└── s7-trend-values/          # Reference documents and legacy files
```

## License

This project is licensed under the [MIT License](LICENSE).
