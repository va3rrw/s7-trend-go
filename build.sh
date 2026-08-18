#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# ── Read version from VERSION (you set it; build does not bump) ─────
VERSION_FILE="$ROOT/VERSION"
if [[ ! -f "$VERSION_FILE" ]]; then
  echo "1.0.0" > "$VERSION_FILE"
fi

VER="$(tr -d '[:space:]' < "$VERSION_FILE")"
if [[ ! "$VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid VERSION '$VER' (expected MAJOR.MINOR.PATCH in $VERSION_FILE)" >&2
  exit 1
fi
echo "Version: v$VER (from VERSION)"

# ── Sync version into Go / Wails / frontend ─────────────────────────
cat > "$ROOT/src/version.go" <<EOF
package main

// AppVersion is synced from VERSION by build.sh (not auto-bumped).
const AppVersion = "$VER"
EOF

cat > "$ROOT/src/frontend/src/version.ts" <<EOF
/** Synced from VERSION by build.sh — set version in the project VERSION file. */
export const APP_VERSION = '$VER'
EOF

EXE_BASENAME="s7-trend-go-v${VER}"
EXE_NAME="${EXE_BASENAME}.exe"

# wails.json productVersion + versioned output filename
if command -v python3 >/dev/null 2>&1; then
  python3 - "$ROOT/src/wails.json" "$VER" "$EXE_BASENAME" <<'PY'
import json, sys
path, ver, outname = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path, encoding="utf-8") as f:
    data = json.load(f)
data["outputfilename"] = outname
data.setdefault("info", {})["productVersion"] = ver
data["info"]["productName"] = "S7 Trend Go"
with open(path, "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY
else
  sed -i "s/\"productVersion\": \"[^\"]*\"/\"productVersion\": \"$VER\"/" "$ROOT/src/wails.json"
  sed -i "s/\"outputfilename\": \"[^\"]*\"/\"outputfilename\": \"$EXE_BASENAME\"/" "$ROOT/src/wails.json"
fi

# ── Build Windows executable ────────────────────────────────────────
echo "Building Windows executable ($EXE_NAME)..."
cd "$ROOT/src"
# Locate wails binary
if [[ -n "${WAILS_BIN:-}" && -x "$WAILS_BIN" ]]; then
  : # Using user-provided WAILS_BIN
elif command -v wails >/dev/null 2>&1; then
  WAILS_BIN="$(command -v wails)"
elif [[ -x "$(go env GOPATH 2>/dev/null)/bin/wails" ]]; then
  WAILS_BIN="$(go env GOPATH)/bin/wails"
elif [[ -x "$HOME/go/bin/wails" ]]; then
  WAILS_BIN="$HOME/go/bin/wails"
elif [[ -x "/tmp/go-bin/wails" ]]; then
  WAILS_BIN="/tmp/go-bin/wails"
else
  echo "wails not found (set WAILS_BIN or install with 'go install github.com/wailsapp/wails/v2/cmd/wails@latest')" >&2
  exit 1
fi
echo "Using Wails binary: $WAILS_BIN"

"$WAILS_BIN" build -platform windows/amd64

echo "Moving executable to dist/ folder..."
mkdir -p "$ROOT/dist"
BUILT="$(ls -1 build/bin/*.exe 2>/dev/null | head -1 || true)"
if [[ -z "$BUILT" ]]; then
  echo "No .exe found in build/bin/" >&2
  exit 1
fi
mv -f "$BUILT" "$ROOT/dist/$EXE_NAME"

echo "Build complete: S7 Trend Go v$VER → dist/$EXE_NAME"
ls -la "$ROOT/dist/$EXE_NAME"
