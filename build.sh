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

EXE_BASENAME="s7-trend-go-v${VER}"
EXE_NAME="${EXE_BASENAME}.exe"

# ── Render Wails v3 & Windows build metadata from tracked templates ─
sed "s/@VERSION@/$VER/g" "$ROOT/src/build/config.yml.tmpl" > "$ROOT/src/build/config.yml"
sed "s/@VERSION@/$VER/g" "$ROOT/src/build/windows/info.json.tmpl" > "$ROOT/src/build/windows/info.json"
sed "s/@VERSION@/$VER/g" "$ROOT/src/build/windows/wails.exe.manifest.tmpl" > "$ROOT/src/build/windows/wails.exe.manifest"

# ── Locate wails3 binary ────────────────────────────────────────────
GOPATH_DIR="$(go env GOPATH 2>/dev/null || true)"
if [[ -n "$GOPATH_DIR" ]] && command -v cygpath >/dev/null 2>&1; then
  GOPATH_DIR="$(cygpath -u "$GOPATH_DIR")"
fi

if [[ -n "${WAILS3_BIN:-}" && -x "$WAILS3_BIN" ]]; then
  : # Using user-provided WAILS3_BIN
elif [[ -n "${WAILS_BIN:-}" && -x "$WAILS_BIN" ]]; then
  WAILS3_BIN="$WAILS_BIN"
elif command -v wails3 >/dev/null 2>&1; then
  WAILS3_BIN="$(command -v wails3)"
elif command -v wails3.exe >/dev/null 2>&1; then
  WAILS3_BIN="$(command -v wails3.exe)"
elif [[ -n "$GOPATH_DIR" && -x "$GOPATH_DIR/bin/wails3.exe" ]]; then
  WAILS3_BIN="$GOPATH_DIR/bin/wails3.exe"
elif [[ -n "$GOPATH_DIR" && -x "$GOPATH_DIR/bin/wails3" ]]; then
  WAILS3_BIN="$GOPATH_DIR/bin/wails3"
elif [[ -x "$HOME/go/bin/wails3.exe" ]]; then
  WAILS3_BIN="$HOME/go/bin/wails3.exe"
elif [[ -x "$HOME/go/bin/wails3" ]]; then
  WAILS3_BIN="$HOME/go/bin/wails3"
elif [[ -x "/usr/local/bin/wails3" ]]; then
  WAILS3_BIN="/usr/local/bin/wails3"
elif [[ -x "/tmp/go-bin/wails3" ]]; then
  WAILS3_BIN="/tmp/go-bin/wails3"
else
  echo "wails3 not found (set WAILS3_BIN or install with 'go install github.com/wailsapp/wails/v3/cmd/wails3@latest')" >&2
  exit 1
fi
echo "Using Wails v3 binary: $WAILS3_BIN"

# ── Generate Wails v3 bindings before frontend tests & build ────────
echo "Generating Wails v3 TypeScript bindings..."
(cd "$ROOT/src" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "$WAILS3_BIN" generate bindings -clean=true -ts -i)

# ── Run test suites & build frontend assets ────────────────────────
echo "Running frontend tests..."
(
  cd "$ROOT/src/frontend"
  if command -v pnpm >/dev/null 2>&1; then
    if [[ ! -d "node_modules" ]]; then
      echo "Installing frontend dependencies with pnpm..."
      pnpm install --frozen-lockfile 2>/dev/null || pnpm install
    fi
    pnpm test
    echo "Building frontend assets..."
    pnpm run build
  else
    if [[ ! -d "node_modules" ]]; then
      echo "Installing frontend dependencies with npm..."
      npm install
    fi
    npm test
    echo "Building frontend assets..."
    npm run build
  fi
)

echo "Running Go tests..."
(
  cd "$ROOT/src"
  if command -v gcc >/dev/null 2>&1; then
    go test -race ./backend/...
  else
    CGO_ENABLED=0 go test ./backend/...
  fi
)

# ── Build Windows executable ────────────────────────────────────────
echo "Building Windows executable ($EXE_NAME)..."
cd "$ROOT/src"
PKG_MGR="npm"
if command -v pnpm >/dev/null 2>&1; then
  PKG_MGR="pnpm"
fi
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "$WAILS3_BIN" build APP_NAME="$EXE_BASENAME" APP_VERSION="$VER" PACKAGE_MANAGER="$PKG_MGR"

echo "Moving executable to dist/ folder..."
mkdir -p "$ROOT/dist"
if [[ -f "bin/$EXE_NAME" ]]; then
  BUILT="bin/$EXE_NAME"
else
  BUILT="$(ls -1 bin/*.exe 2>/dev/null | head -1 || true)"
fi
if [[ -z "$BUILT" ]]; then
  echo "No .exe found in bin/" >&2
  exit 1
fi
mv -f "$BUILT" "$ROOT/dist/$EXE_NAME"

echo "Build complete: S7 Trend Go v$VER → dist/$EXE_NAME"
ls -la "$ROOT/dist/$EXE_NAME"
