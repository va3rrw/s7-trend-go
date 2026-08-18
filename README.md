# S7 Trend Go

Siemens S7 PLC data acquisition and trend visualization desktop application built with Go, Wails v2, and Vue 3.

## Overview

S7 Trend Go connects to Siemens S7 PLCs (S7-300, S7-400, S7-1200, S7-1500) over Ethernet using native S7 protocol communication. It records real-time tag data and plots multi-axis trend charts.

Based on a very old but useful tool **S7 Trend Values**.

## Key Features

- **Multi-PLC Communication**: Connect to multiple S7 PLCs simultaneously with configurable IP address, Rack, and Slot settings.
- **S7 Tag Support**: Read DB, Memory (M), Inputs (I), and Outputs (Q) memory areas supporting `Bool`, `Byte`, `Word`, `Int`, `DWord`, `DInt`, `Real`, `LReal`, and `String` data types.
- **High-Performance Real-Time Charting**: High-frequency polling (10ms to 60,000ms) with customizable time windows (30s to 3,600s).
- **Multi-Axis Y Scaling**: Configure independent Y-axes with automatic scaling or explicit minimum and maximum range bounds.
- **Sample Interpolation**: Choose between `Line` and `Differential` interpolation modes for trend visualization.
- **Import & Export**: Save and reload PLC configurations, tag lists, Y-axes setups, and exported sample records.
- **Localization**: Built-in support for English (`en`) and Simplified Chinese (`zh`).

## Architecture & Tech Stack

- **Backend**: Go 1.25, [Wails v2](https://wails.io/), [gos7](https://github.com/robinson/gos7) S7 protocol library.
- **Frontend**: Vue 3 (TypeScript), Vite, UnoCSS, HTML5 Canvas / ECharts trend rendering, Vue I18n.
- **Target OS**: Windows (`windows/amd64`).

## Prerequisites

- **Go**: 1.25 or later
- **Node.js**: 22+ and **pnpm** package manager
- **Wails CLI**: `v2.13.0` or later (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Getting Started

### Development Mode

Run the application in live-reload development mode:

```bash
cd src
wails dev
```

This starts the Go backend and Vite development server, launching the application window with instant hot-reloading.

### Building for Production

Use the included build script to compile the versioned standalone Windows executable:

```bash
./build.sh
```

The script syncs the version string from the `VERSION` file into `src/version.go`, updates `wails.json`, compiles the binary, and outputs the final `.exe` into the `dist/` directory (for example, `dist/s7-trend-go-v1.1.0.exe`).

## Project Layout

```text
s7-trend-go/
├── VERSION                   # Single source of truth for app version
├── build.sh                  # Build script for Windows executable
├── dist/                     # Destination folder for compiled binaries
├── src/
│   ├── main.go               # Entry point and Wails runtime setup
│   ├── wails.json            # Wails app configuration
│   ├── backend/              # Go backend packages
│   │   ├── app.go            # App lifecycle and binding methods
│   │   ├── app_polling.go    # S7 PLC polling and worker routines
│   │   ├── app_files.go      # Config import/export and file handlers
│   │   ├── codec.go          # S7 byte encoding/decoding logic
│   │   └── models.go         # Data structures and settings schemas
│   └── frontend/             # Vue 3 frontend project
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

