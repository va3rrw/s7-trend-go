# Plan: Port Old-Tool Advanced Options into S7 Trend Go

> **Status:** Planning only for advanced features. First incremental task: regroup existing menus to match the old tool (File / Edit / Tools / Trace / Help). Many old-tool options are intentionally deferred as low value.

## Sources analyzed

| Artifact | Findings |
|---|---|
| `s7-trend-values/S7 Trend Values.pdf` (v2.0, 4 pages) | Core workflows: PLC links, channels, start/stop, Y/X axis dialogs, interpolation, notifications, force value |
| `s7-trend-values/S7TrendValues.exe` (UPX-packed Delphi/Win32, product **2.0.4.0**) | Full menu tree + form captions recovered after UPX decompress; binary DFMs for all dialogs |
| `s7-trend-values/FFT.dll` | Exports: `_fft_calculation`, `_fft_Ncalculation`, `_hamming`, `_hi_pass_filter`, `_low_pass_filter`, `_exportMat_to_matlab` |
| Current Go app (`src/models.go`, `App.vue`, dialogs, `chart.ts`) | Baseline feature set for gap analysis |

The user-facing “advanced settings page with many options” is the old menu item:

**Edit → Experts Plot options …** (`ShowallPlotoptions1Click`)

That opens the **Iocomp TiPlot component editor** (`TIPLOTEDITORFORM` / `TIPLOTPREVIEWSETTINGSFORM`) — a multi-tab property sheet covering nearly every visual and interaction knob of the chart control. It is far larger than the simple Y/X dialogs shown in the PDF.

---

## 1. What the old tool exposes

### 1.1 Main menu map (from EXE captions)

```
File
  Load PLC and Channels …
  Load PLC Settings … / Save PLC Settings …
  Load Channels settings … / Save Channel Settings …
  Load records … / Save records …
  Print …
  Close

Edit
  PLC Settings …
  Channel Settings …
  X Settings …
  Y(s) Settings …
  Jump at date time …
  Experts Plot options …     ← advanced page

Tools
  FFT…
  Export CSV …
  Export XLS …
  Run another instance

Trace
  Set Timer Interval …       (50–5000 ms in old; Go uses 20–60000)
  START / PAUSE / STOP
  Samples interpolation … → Line | Differential
  Notifications…
    Binary → On positive edge | On negative edge | Both edges
    Analog → On Low value | On High Value | Both values
    Deactivate all
    Remove all from trend
  X Scroll … → Smooth | Page
  Clear Records …
  Auto save long recording sessions …  (menu present, disabled in this build)

Help [PDF]
```

### 1.2 Experts Plot options (advanced editor tabs)

Top-level sections recovered from the iPlot editor DFM:

| Tab | Sub-areas / notable options |
|---|---|
| **Control / Plot** | General (update frame rate, auto frame rate, outer margin, border style, clip annotations), Title (text/font/visible), Background (color + gradient start/stop/direction), Print (document name, orientation, show dialog, paper size, margins), Hints (pause / hide pause), File I/O (properties load/save, data/log path, buffer size, activate/deactivate) |
| **Channels** | Per-channel: title, ring buffer size, data style, layer, X/Y tracking enabled, visible in legend, popup enabled, fast draw, user can move data points, clip to axes; **line** style/width; **interpolation style**; **marker** size/style/turn-off limit/use channel color; **bar**, **high/low**, **candlestick**, **fill**, **digital** (ref high/low/style); **stats** (point count, memory, X/Y min/max/mean, max data points, point capacity); per-channel file I/O |
| **Annotations** | Defaults (brush/pen), layout, add/remove, reference position/size, clip to axes |
| **Cursors** | Dual pointers, snap to data point, channel filter (allow all / individual), show in legend, hint hide-on-release, orientation, limits (line positions, fill, user can move), labels |
| **X-Axes / Y-Axes** | Add/remove axes; general (span, min, desired increment/start, reverse scale, scale type linear/log, restore on resume, master UI input, grid lines visible); title; labels (precision, date-time format, rotation, min length auto-adjust); cursor/legend format; tracking (scroll smooth/page, scroll compress max, align first, increment style); scroll min/max; scale ticks (major/minor length/count, margins, stacking, force stacking); Cartesian child ref |
| **Legend** | Visibility, margins, selected item font/color, background transparent/color, channel name max width/color style, line column size; columns (X/Y title, marker, X/Y value, line, Y max/min/mean, titles); wrap (col/row auto count & spacing); tables |
| **ToolBar** | Zoom in/out factor; show buttons (edit, copy, save, print, resume, pause, axes mode, zoom in/out, zoom box, cursor, select, preview); flat/small buttons, flat border |
| **Data View** | Background; axes control (wheel/mouse style); grid lines (major/minor style/color/width, per-side show, X/Y majors/minors custom) |

Also: **theme schemes** (Open/Save Settings, scheme name/category maintenance).

### 1.3 Other dialogs beyond the PDF

| Dialog / feature | Options |
|---|---|
| **Y Axis Settings** | Min, Max, Label, Scale Type **Linear / Log**, Auto scale related channels, Apply; File Open/Save settings |
| **X Axis Settings** | Hours / Minutes / Seconds window |
| **Item Properties** | Source Type (DATABLOCK/MERKER/INPUT/OUTPUT), DB, DW/bit, Data Type, PLC Topic, Y Axis, Notify low/high + thresholds |
| **Force Value** | Write selected channel value to PLC (bool toggle via legend double-click) |
| **FFT** | Channel select, Y log scale, spectrum chart, max Hz |
| **Go to Date Time** | Jump view to absolute timestamp |
| **Channels Config** | Activate all / deactivate all / selected; Enable on create; Force Value; Load/Save channel settings |

### 1.4 Toolbar (main window)

Play, pause, channels, stop, clear, pan X/Y, zoom in/out, zoom box, Y settings, X settings, cursor/select modes, FFT entry.

Keyboard help strings in EXE: arrows scroll; Ctrl+arrows zoom X/Y.

---

## 2. Current Go tool baseline

Already present (partial parity):

| Area | Go status |
|---|---|
| 4 PLC links (IP/rack/slot/CPU type), connect/test | Yes (`PlcDialog`, gos7) |
| Tags: name, address, type, color, Y axis, enable, notification enum | Yes (low/high limits exist in model but not edited/used fully) |
| Poll interval + time window | Yes (prompt dialogs) |
| Interpolation Line / Differential | Yes (chart) |
| Y axes min/max + autoScale | Yes (no log scale) |
| Boolean band under chart | Yes (basic) |
| Start / pause chart / stop, CSV export | Yes |
| Load/save PLCs & tags as JSON | Yes |
| i18n EN/ZH | Yes |

Missing or incomplete vs old tool:

- **Experts Plot options** UI and persistence entirely absent  
- Log Y scale, X scroll modes (smooth/page), jump-to-datetime  
- Data cursors / measurement, zoom box, richer pan/zoom toolbar  
- Notifications: model fields exist; **no runtime edge/threshold alerts**, no global Trace → Notifications menu  
- Force write to PLC (app is read-only today)  
- FFT + filters  
- Auto-save long sessions, load/save raw records, print, XLS export  
- Per-channel thresholds in tag editor UI (fields exist, not in form)  
- Chart legend with min/max like old bottom strip  
- Ring buffer / max points / frame-rate controls  

---

## 3. Design principles for the Go port

1. **Do not clone the full Iocomp editor 1:1.** Hundreds of TeeChart-style properties are low value in a Vue + Chart.js stack. Port **user-visible behavior**, not Delphi property sheets.
2. **Group options into one “Advanced settings” dialog** (and a few focused menus), mapping old Expert tabs onto a smaller, maintainable model.
3. **Persist everything in `AppSettings` (JSON)** so load/save of PLC/tags/session can grow to full project files later.
4. **Phase by user value**: chart interaction and notification behavior first; write/FFT later (behavior/safety decisions).
5. Keep Wails boundaries: Go owns PLC I/O, buffering policy, export; Vue owns plot chrome and option UI.

---

## 4. Proposed settings model extension

Extend `AppSettings` / `types.ts` (illustrative):

```ts
interface AdvancedPlotSettings {
  // Control
  backgroundColor: string;
  grid: {
    showXMajor: boolean; showXMinor: boolean;
    showYMajor: boolean; showYMinor: boolean;
    majorColor: string; minorColor: string;
  };
  showLegend: boolean;
  legend: {
    showYMin: boolean; showYMax: boolean; showYMean: boolean;
    showXValue: boolean; showYValue: boolean;
  };
  updateFrameRateMs: number;   // UI redraw throttle
  maxPointsPerTag: number;     // ring buffer capacity
  clipToAxes: boolean;

  // Interaction
  xScrollMode: 'smooth' | 'page';
  zoomFactor: number;
  enableZoomBox: boolean;
  enableDataCursor: boolean;
  cursorSnapToPoint: boolean;
  restoreAxisOnResume: boolean;

  // Trace behavior
  autoSaveRecording: boolean;
  autoSaveDirectory: string;
  autoSaveIntervalSec: number;
  notificationsEnabled: boolean; // master
  notificationSound: boolean;

  // Optional later
  printOrientation: 'portrait' | 'landscape';
}

interface YAxisSettings {
  // existing +
  scaleType: 'linear' | 'log';
  label: string; // display title (old Label field)
}
```

Tag-level keep/extend: `lowLimit` / `highLimit` (expose in UI), `notification`, optional `lineWidth`, `markerStyle` (subset).

---

## 5. UI plan

### 5.1 New dialog: **Advanced Settings** (maps “Experts Plot options”)

Single modal with **tabs** (not 10 Delphi pages — 5–6 focused tabs):

1. **Plot** — background, grid majors/minors/colors, outer margins (if easy), title optional  
2. **Series** — default line width, interpolation default, markers on/off, max points per tag, clip to axes  
3. **Axes** — default restore-on-resume; links to Y-axis log scale; X scroll mode; jump-to-datetime helper  
4. **Legend & cursor** — legend column toggles (min/max/mean/value); enable dual data cursor + snap  
5. **Recording** — auto-save long sessions, directory, interval; export format defaults (CSV separator)  
6. **Performance** — update frame rate / UI throttle, buffer size  

Menu placement:

- **Trace → Advanced settings…** (primary)  
- Optional shortcut from toolbar gear icon  

Also enhance existing dialogs:

| Dialog | Additions |
|---|---|
| **YAxesDialog** | Scale type Linear/Log; optional label; “Apply” live preview |
| **TagEditorDialog** | Low/High limits when notification is analog; show thresholds in main grid |
| **Trace menu** | Notifications submenu (global apply + deactivate all); X Scroll Smooth/Page; Jump to date/time |
| **Toolbar** | Zoom in/out, zoom box, cursor mode (phased) |

### 5.2 What we intentionally de-scope (v1)

- Full annotation editor, candlestick/OHLC series styles, Cartesian child axes stacking  
- OPC browser forms (Iocomp leftovers; not S7-specific)  
- Theme “scheme” maintenance panel (replace with simple import/export of advanced JSON)  
- Print engine parity (browser print of chart canvas is enough later)  
- XLS export (CSV first; XLS as later optional lib)  
- Porting `FFT.dll` binary (reimplement FFT in pure Go/JS if needed)

---

## 6. Implementation phases

### Phase A — Settings shell + persistence (foundation)

**Goal:** Advanced dialog exists; values save/load with settings; no behavior change yet.

- Extend `AppSettings` in Go + TS; defaults in `CreateDefaultSettings` / `defaultSettings`  
- `AdvancedSettingsDialog.vue` with tabs + i18n keys (en/zh)  
- Wire menu **Trace → Advanced settings**  
- Persist via existing `SaveSettings` / future full project file  
- Unit tests for default merge when loading old JSON missing new fields  

**Files:** `models.go`, `types.ts`, `store.ts`, `App.vue`, new dialog, locales.

### Phase B — Plot appearance (high visual parity)

**Goal:** Background, grid, legend, log Y, series density.

- Apply `backgroundColor` / grid options in `chart.ts` (Chart.js scales + plugins)  
- Y-axis `scaleType: 'log'` with validation (positive domain)  
- Legend strip under chart: color, name, live Y, min, max (old bottom channel table)  
- `maxPointsPerTag` ring trim in `TrendChart` history  
- `updateFrameRateMs` throttle for `requestAnimationFrame` / chart update  

**Files:** `chart.ts`, `TrendChartView.vue`, `YAxesDialog.vue`, CSS.

### Phase C — Interaction (toolbar / cursors / scroll)

**Goal:** Match Trace + toolbar navigation users expect.

- X scroll modes: smooth drag (current) vs page step  
- Zoom in/out buttons + wheel + Ctrl+arrows optional  
- Zoom-box drag mode  
- Dual vertical cursors with Δt / Δy readout (Chart.js annotations or custom plugin)  
- **Jump at date/time** dialog (scroll `viewOffsetSeconds` to target)  
- Pause restores axis range when `restoreAxisOnResume`  

**Files:** `chart.ts`, `App.vue` toolbar, small `JumpToTimeDialog.vue`.

### Phase D — Notifications (runtime behavior)

**Goal:** Make existing `Notification` + limits real.

- On each poll sample, detect binary edges / analog low-high  
- Emit events → status bar message, optional highlight row red (old threshold behavior), optional system beep  
- Trace → Notifications menu: apply mode to selected/all, deactivate all, remove from trend  
- Tag editor: low/high fields; main grid columns Low/High  

**Files:** `app.go` or frontend poll handler, `TagEditorDialog.vue`, grid in `App.vue`.

### Phase E — Recording & export

**Goal:** Long sessions and richer export.

- Auto-save: append/rotate CSV while sampling when enabled  
- Load/save records (`.csv` / internal JSON) into chart history  
- CSV separator option; Export XLS (optional, e.g. `excelize` later)  
- Print: `window.print` or export chart PNG  

**Files:** `app.go` (file I/O), Trace menu, Advanced → Recording tab.

### Phase F — Force write (optional, safety-gated)

**Goal:** Old “Force Value…” parity.

- Go: `WriteTag(tagId, value)` using gos7 write APIs (DB/MB/EB/AB)  
- UI: Force Value dialog; bool toggle on legend/grid double-click when enabled  
- **Safety:** confirm dialog; optional “allow writes” master switch default **off**; document that optimized DBs / safety PLCs need care  

**Files:** `app.go`, `codec.go` write path, dialogs.

### Phase G — FFT (optional advanced tool)

**Goal:** Tools → FFT…

- Pure Go or JS FFT (do not load Windows `FFT.dll`)  
- Dialog: select channel, window (Hamming), Y log, frequency range  
- Uses buffered samples for selected tag; spectrum Chart.js view  

**Files:** new `fft.go` or `frontend/src/fft.ts`, `FftDialog.vue`.

---

## 7. Mapping table: old Expert option → Go action

| Old Experts area | Priority | Phase | Approach in Go |
|---|---|---|---|
| Plot background / gradient | High | B | Solid bg first; gradient optional CSS |
| Grid major/minor X/Y | High | B | Extend `secondGridPlugin` + Chart.js grid |
| Channel line width / markers | Medium | B | Chart.js `borderWidth`, `pointRadius` |
| Interpolation style | Done | — | Already Line/Differential |
| Ring buffer / max points | High | B | `maxPointsPerTag` trim |
| Frame rate | Medium | B | UI throttle |
| Legend min/max/mean | High | B | Bottom legend strip |
| Data cursor + snap | High | C | Custom plugin |
| X/Y tracking / scroll modes | High | C | `xScrollMode` + zoom |
| Zoom box / factor | High | C | Toolbar modes |
| Log scale Y | High | B | Chart.js `type: 'logarithmic'` |
| Reverse scale / multi X-axis | Low | later | Skip multi-X; reverse optional |
| Annotations | Low | later | Skip v1 |
| Candlestick / bar / OHLC | Low | — | Out of scope for PLC trend |
| Digital ref style | Medium | B | Boolean band already; refine colors |
| ToolBar button visibility | Low | C | Fixed useful subset, not full editor |
| File I/O buffer / auto log | High | E | Auto-save recording |
| Print settings | Low | E | Simple print/PNG |
| Theme schemes | Low | A | JSON import/export of advanced block |
| OPC | — | — | Do not port |

---

## 8. Backend / architecture notes

```
┌─────────────────────────────────────────────┐
│ Vue: AdvancedSettingsDialog + chart chrome  │
│  - edits AppSettings.advanced               │
│  - chart.ts consumes appearance/interaction │
└──────────────────┬──────────────────────────┘
                   │ SaveSettings / events
┌──────────────────▼──────────────────────────┐
│ Go App: polling, optional write, auto-save  │
│  - max buffer policy                        │
│  - notification evaluation (or pure FE)     │
│  - ExportCSV / AutoSave / WriteTag          │
└─────────────────────────────────────────────┘
```

- Prefer evaluating **notifications on the frontend** from poll updates (already has edges of history); keeps Go thinner. Use Go only if multi-window/headless logging is needed.  
- **Auto-save** should live in Go (reliable disk I/O on Windows).  
- Chart.js plugins stay in `chart.ts` to avoid scattering DOM logic.

---

## 9. Delivery order (incremental)

**Done / in progress**

0. **Menu regroup only** — File / Edit / Tools / Trace / Help using existing actions (no new advanced features). Many old items stay out until needed.

**Later (only features we decide are useful)**

1. Phase A — Advanced settings shell + model (subset of Experts options)  
2. Phase B — Plot appearance (grid, legend min/max, max points, log Y)  
3. Phase C — Interaction (zoom, cursors, jump-to-time, X scroll)  
4. Phase D — Live notifications + threshold UI  

Defer F/G (writes / FFT) until explicitly requested. Skip options that are not useful in practice.

---

## 10. Verification plan

- Load old-style settings JSON missing `advanced` → defaults filled, no crash  
- Toggle grid/legend/log Y live while sampling  
- Cursor Δt accuracy vs known sample interval  
- Notification: force simulated bool toggle / analog cross of low/high  
- Auto-save files rotate correctly under continuous poll  
- i18n complete for new strings  
- Manual compare against old EXE screenshots (PDF pages 1–4 + Experts editor)  

---

## 11. Open decisions (confirm before coding F/G)

1. **Writes:** Should Force Value be supported, or remain read-only monitor?  
2. **FFT:** Needed for your workflows, or skip?  
3. **Experts fidelity:** Prefer compact Advanced dialog (recommended) vs multi-page clone of Iocomp?  
4. **Auto-save format:** CSV only vs proprietary binary/JSON session?  

Default recommendations: **read-only until asked**, **FFT later**, **compact Advanced dialog**, **CSV auto-save**.

---

## 12. Summary

The old tool’s “many options” page is **Edit → Experts Plot options**, the Iocomp TiPlot editor (Control, Channels, Annotations, Cursors, X/Y Axes, Legend, ToolBar, Data View, Tables, Print, File I/O). Combined with Trace/Tools menus, the old product also offers FFT, force-write, export XLS, record load/save, log Y, scroll modes, and notifications.

The Go port already covers core PLC sampling and basic trace options. The plan is to add a **focused Advanced Settings dialog** plus phased chart/interaction/notification/recording work that recreates the **useful** Expert options on Chart.js, without porting the entire Delphi component editor or OPC stack.
