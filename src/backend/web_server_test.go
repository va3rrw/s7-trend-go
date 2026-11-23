package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func isolateConfigDir(t *testing.T) {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("APPDATA", cfgDir)
	t.Setenv("HOME", cfgDir)
}

func reserveFreeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", msg)
}

func doRPC(t *testing.T, baseURL, method string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal RPC body: %v", err)
		}
		payload = bytes.NewReader(raw)
	} else {
		payload = strings.NewReader("{}")
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/rpc/"+method, payload)
	if err != nil {
		t.Fatalf("build RPC request %s: %v", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("execute RPC %s: %v", method, err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read RPC %s response: %v", method, err)
	}
	return resp, respBytes
}

func TestWebServer_LifecycleAndTrayUpdates(t *testing.T) {
	isolateConfigDir(t)
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	SetWebAssets(app, fstest.MapFS{
		"frontend/dist/index.html": {Data: []byte("<html><head><title>Test</title></head><body>ok</body></html>")},
	})

	initial := app.GetWebServerStatus()
	if initial.Enabled || initial.Running || initial.Port != defaultWebServerPort {
		t.Fatalf("unexpected initial status: %+v", initial)
	}

	started, err := app.startWebServer("127.0.0.1", 0, true)
	if err != nil {
		t.Fatalf("startWebServer failed: %v", err)
	}
	if !started.Running || !started.Enabled || started.Host != "127.0.0.1" || started.Port <= 0 || started.URL == "" {
		t.Fatalf("unexpected started status: %+v", started)
	}

	host.mu.Lock()
	if len(host.webUpdates) == 0 || !host.webUpdates[len(host.webUpdates)-1].running {
		host.mu.Unlock()
		t.Fatalf("expected UpdateTrayWebServer running=true update, got %+v", host.webUpdates)
	}
	host.mu.Unlock()

	if _, err := app.SetWebServerPort(0); err == nil {
		t.Error("expected error for port 0")
	}
	if _, err := app.SetWebServerPort(70000); err == nil {
		t.Error("expected error for port 70000")
	}

	newPort := reserveFreeTCPPort(t)
	restarted, err := app.SetWebServerPort(newPort)
	if err != nil {
		t.Fatalf("SetWebServerPort live restart failed: %v", err)
	}
	if !restarted.Running || restarted.Port != newPort {
		t.Fatalf("expected running server on port %d, got %+v", newPort, restarted)
	}

	stopped, err := app.ToggleWebServer()
	if err != nil {
		t.Fatalf("ToggleWebServer off failed: %v", err)
	}
	if stopped.Running || stopped.Enabled {
		t.Fatalf("expected stopped & disabled after toggle off, got %+v", stopped)
	}

	stoppedPort := reserveFreeTCPPort(t)
	configured, err := app.SetWebServerPort(stoppedPort)
	if err != nil {
		t.Fatalf("SetWebServerPort while stopped failed: %v", err)
	}
	if configured.Running || configured.Port != stoppedPort {
		t.Fatalf("expected stopped server with port %d, got %+v", stoppedPort, configured)
	}

	toggledOn, err := app.ToggleWebServer()
	if err != nil {
		t.Fatalf("ToggleWebServer on failed: %v", err)
	}
	if !toggledOn.Running || !toggledOn.Enabled || toggledOn.Port != stoppedPort {
		t.Fatalf("expected running server on configured port %d, got %+v", stoppedPort, toggledOn)
	}

	afterStop := app.StopWebServer()
	if afterStop.Running || afterStop.Enabled {
		t.Fatalf("expected StopWebServer to stop and disable, got %+v", afterStop)
	}
}

func TestWebServer_HTTPAndRPCEndpoints(t *testing.T) {
	isolateConfigDir(t)
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	tagID := uuid.New()
	settings := CreateDefaultSettings()
	settings.Tags = []TagSettings{
		{
			Id:       tagID,
			Name:     "Pressure",
			PlcLink:  "PLC1",
			Address:  "DB1.DBD0",
			DataType: DataTypeReal,
			YAxis:    "Y-Axis 1",
			Color:    "#112233",
			Enabled:  true,
		},
	}
	app.SaveSettings(settings)
	ts := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	app.RecordSample(tagID.String(), ts, 42.5)

	SetWebAssets(app, fstest.MapFS{
		"frontend/dist/index.html": {Data: []byte("<!doctype html><html><head><title>S7</title></head><body>ok</body></html>")},
		"frontend/dist/app.js":     {Data: []byte("console.log('ready');")},
	})

	st, err := app.startWebServer("127.0.0.1", 0, true)
	if err != nil {
		t.Fatalf("startWebServer failed: %v", err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", st.Port)
	client := &http.Client{Timeout: 5 * time.Second}

	// GET /health
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("unexpected /health response %d: %s", resp.StatusCode, string(body))
	}

	// GET /api/status
	resp, err = client.Get(baseURL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status failed: %v", err)
	}
	var snap ServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		resp.Body.Close()
		t.Fatalf("decode /api/status failed: %v", err)
	}
	resp.Body.Close()
	if snap.Version != AppVersion || snap.EnabledTags != 1 || !snap.WebServer.Running {
		t.Fatalf("unexpected /api/status snapshot: %+v", snap)
	}

	// GET /api/export/csv
	resp, err = client.Get(baseURL + "/api/export/csv")
	if err != nil {
		t.Fatalf("GET /api/export/csv failed: %v", err)
	}
	csvBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(csvBody), "Pressure,DB1.DBD0,42.5") {
		t.Fatalf("unexpected CSV response %d: %s", resp.StatusCode, string(csvBody))
	}

	// GET / injects window.__S7_WEB_SERVER__ before </head>
	resp, err = client.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	indexBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	indexHTML := string(indexBody)
	markerIdx := strings.Index(indexHTML, webClientMarker)
	headCloseIdx := strings.Index(strings.ToLower(indexHTML), "</head>")
	if resp.StatusCode != http.StatusOK || markerIdx < 0 || markerIdx > headCloseIdx {
		t.Fatalf("expected marker injected before </head>, got: %s", indexHTML)
	}

	// Static asset /app.js
	resp, err = client.Get(baseURL + "/app.js")
	if err != nil {
		t.Fatalf("GET /app.js failed: %v", err)
	}
	jsBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(jsBody), "ready") {
		t.Fatalf("unexpected /app.js response %d: %s", resp.StatusCode, string(jsBody))
	}

	// RPC GetVersion, GetSettings, SaveSettings, IsPolling, ImportSettingsJSON, GetHistoryRange
	rResp, rBody := doRPC(t, baseURL, "GetVersion", nil, nil)
	if rResp.StatusCode != http.StatusOK || !strings.Contains(string(rBody), AppVersion) {
		t.Fatalf("GetVersion unexpected %d: %s", rResp.StatusCode, string(rBody))
	}

	rResp, rBody = doRPC(t, baseURL, "GetSettings", nil, nil)
	if rResp.StatusCode != http.StatusOK {
		t.Fatalf("GetSettings failed %d: %s", rResp.StatusCode, string(rBody))
	}

	updatedSettings := CreateDefaultSettings()
	updatedSettings.PollIntervalMs = 250
	rResp, _ = doRPC(t, baseURL, "SaveSettings", updatedSettings, map[string]string{
		"Origin": baseURL,
	})
	if rResp.StatusCode != http.StatusOK || app.GetSettings().PollIntervalMs != 250 {
		t.Fatalf("SaveSettings via same-origin failed %d (pollMs=%d)", rResp.StatusCode, app.GetSettings().PollIntervalMs)
	}

	rResp, rBody = doRPC(t, baseURL, "IsPolling", nil, nil)
	if rResp.StatusCode != http.StatusOK || strings.TrimSpace(string(rBody)) != "false" {
		t.Fatalf("IsPolling unexpected %d: %s", rResp.StatusCode, string(rBody))
	}

	imported := CreateDefaultSettings()
	imported.TimeWindowSeconds = 600
	importedJSON, _ := json.Marshal(imported)
	rResp, rBody = doRPC(t, baseURL, "ImportSettingsJSON", map[string]string{"rawJSON": string(importedJSON)}, nil)
	if rResp.StatusCode != http.StatusOK || app.GetSettings().TimeWindowSeconds != 600 {
		t.Fatalf("ImportSettingsJSON unexpected %d: %s", rResp.StatusCode, string(rBody))
	}

	rResp, rBody = doRPC(t, baseURL, "GetHistoryRange", map[string]any{
		"tagIds":    []string{tagID.String()},
		"startMs":   ts - 1000,
		"endMs":     ts + 1000,
		"maxPoints": 100,
	}, nil)
	if rResp.StatusCode != http.StatusOK || !strings.Contains(string(rBody), "42.5") {
		t.Fatalf("GetHistoryRange unexpected %d: %s", rResp.StatusCode, string(rBody))
	}

	// Cancelled request context propagates into GetHistoryRange
	cancelledCtx, cancelReq := context.WithCancel(context.Background())
	cancelReq()
	cancelBody, _ := json.Marshal(map[string]any{
		"tagIds":    []string{tagID.String()},
		"startMs":   ts - 1000,
		"endMs":     ts + 1000,
		"maxPoints": 100,
	})
	cancelReqHTTP := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/rpc/GetHistoryRange", bytes.NewReader(cancelBody)).WithContext(cancelledCtx)
	cancelReqHTTP.Header.Set("Content-Type", "application/json")
	cancelRec := httptest.NewRecorder()
	app.newWebHandler(context.Background()).ServeHTTP(cancelRec, cancelReqHTTP)
	if cancelRec.Code != 499 {
		t.Fatalf("expected status 499 for canceled GetHistoryRange RPC context, got %d (%s)", cancelRec.Code, cancelRec.Body.String())
	}

	// Cross-origin protection (Origin mismatch and Sec-Fetch-Site: cross-site)
	rResp, _ = doRPC(t, baseURL, "GetVersion", nil, map[string]string{
		"Origin": "http://evil.example",
	})
	if rResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-origin Origin header, got %d", rResp.StatusCode)
	}
	rResp, _ = doRPC(t, baseURL, "GetVersion", nil, map[string]string{
		"Sec-Fetch-Site": "cross-site",
	})
	if rResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for Sec-Fetch-Site: cross-site, got %d", rResp.StatusCode)
	}

	// Blocked admin/desktop-only RPC methods return 404
	for _, blocked := range []string{
		"StartWebServer",
		"StopWebServer",
		"ToggleWebServer",
		"SetWebServerPort",
		"QuitApp",
		"OpenStorageFolder",
	} {
		rResp, _ = doRPC(t, baseURL, blocked, nil, nil)
		if rResp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for blocked RPC %s, got %d", blocked, rResp.StatusCode)
		}
	}
}

func TestWebServer_WebSockets(t *testing.T) {
	isolateConfigDir(t)
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	st, err := app.startWebServer("127.0.0.1", 0, true)
	if err != nil {
		t.Fatalf("startWebServer failed: %v", err)
	}

	wsBase := fmt.Sprintf("ws://127.0.0.1:%d", st.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pollConn, _, err := websocket.Dial(ctx, wsBase+"/api/ws/polling", nil)
	if err != nil {
		t.Fatalf("dial /api/ws/polling failed: %v", err)
	}
	defer pollConn.CloseNow()

	eventConn, _, err := websocket.Dial(ctx, wsBase+"/api/ws/events", nil)
	if err != nil {
		t.Fatalf("dial /api/ws/events failed: %v", err)
	}
	defer eventConn.CloseNow()

	waitUntil(t, 2*time.Second, func() bool {
		return app.GetWebServerStatus().ConnectedClients == 1
	}, "connectedClients == 1")

	wsTagID := uuid.New()
	numVal := 99.25
	app.publishTelemetry(TelemetryFrame{
		Updates: []PollUpdate{
			{
				TagId:        wsTagID,
				Timestamp:    "2026-05-01T12:00:00Z",
				Value:        "99.25",
				NumericValue: &numVal,
				Quality:      "Good",
			},
		},
	})

	var gotFrame TelemetryFrame
	if err := wsjson.Read(ctx, pollConn, &gotFrame); err != nil {
		t.Fatalf("read telemetry frame from ws failed: %v", err)
	}
	if len(gotFrame.Updates) != 1 || gotFrame.Updates[0].TagId != wsTagID || gotFrame.Updates[0].Value != "99.25" {
		t.Fatalf("unexpected telemetry frame: %+v", gotFrame)
	}

	app.emitAppEvent("app:sampling-state", SamplingStateEvent{IsSampling: true})
	var gotEv AppEvent
	if err := wsjson.Read(ctx, eventConn, &gotEv); err != nil {
		t.Fatalf("read app event from ws failed: %v", err)
	}
	if gotEv.Name != "app:sampling-state" {
		t.Fatalf("unexpected app event: %+v", gotEv)
	}

	_ = pollConn.Close(websocket.StatusNormalClosure, "done")
	waitUntil(t, 2*time.Second, func() bool {
		return app.GetWebServerStatus().ConnectedClients == 0
	}, "connectedClients == 0 after close")
}

func TestWebServer_PersistenceAndStartup(t *testing.T) {
	isolateConfigDir(t)
	port := reserveFreeTCPPort(t)

	app1 := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history1.db"))
	if _, err := app1.StartWebServer("127.0.0.1", port); err != nil {
		t.Fatalf("app1 StartWebServer failed: %v", err)
	}

	rawState, err := os.ReadFile(getAppStateFilePath())
	if err != nil {
		t.Fatalf("read app_state.json after StartWebServer failed: %v", err)
	}
	var saved AppState
	if err := json.Unmarshal(rawState, &saved); err != nil {
		t.Fatalf("unmarshal app_state.json failed: %v", err)
	}
	if !saved.WebServerEnabled || saved.WebServerHost != "127.0.0.1" || saved.WebServerPort != port {
		t.Fatalf("unexpected persisted web server state: %+v", saved)
	}

	// Desktop quit should stop listener cleanly without clearing WebServerEnabled.
	if err := app1.ServiceShutdown(); err != nil {
		t.Fatalf("app1 ServiceShutdown failed: %v", err)
	}
	if app1.GetWebServerStatus().Running {
		t.Fatal("expected listener stopped after ServiceShutdown")
	}
	rawAfterShutdown, err := os.ReadFile(getAppStateFilePath())
	if err != nil {
		t.Fatalf("read app_state.json after ServiceShutdown failed: %v", err)
	}
	var afterShutdown AppState
	if err := json.Unmarshal(rawAfterShutdown, &afterShutdown); err != nil {
		t.Fatalf("unmarshal app_state.json after ServiceShutdown failed: %v", err)
	}
	if !afterShutdown.WebServerEnabled {
		t.Fatal("expected ServiceShutdown to preserve webServerEnabled=true")
	}

	// Next launch loads state and resumes listening via StartWebServerIfEnabled.
	app2 := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history2.db"))
	defer func() { _ = app2.ServiceShutdown() }()
	app2.loadAppState()

	loaded := app2.GetWebServerStatus()
	if !loaded.Enabled || loaded.Running || loaded.Port != port || loaded.Host != "127.0.0.1" {
		t.Fatalf("unexpected loaded state before startup: %+v", loaded)
	}

	StartWebServerIfEnabled(app2)
	resumed := app2.GetWebServerStatus()
	if !resumed.Running || resumed.Port != port {
		t.Fatalf("expected StartWebServerIfEnabled to listen on port %d, got %+v", port, resumed)
	}
}

func TestWebServer_SettingsSyncBidirectional(t *testing.T) {
	isolateConfigDir(t)
	app := NewAppWithHistoryPath(filepath.Join(t.TempDir(), "history.db"))
	defer func() { _ = app.ServiceShutdown() }()

	host := &mockDesktopHost{}
	AttachHost(app, host)

	st, err := app.startWebServer("127.0.0.1", 0, true)
	if err != nil {
		t.Fatalf("startWebServer failed: %v", err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", st.Port)
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/api/ws/events", st.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventConn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial /api/ws/events failed: %v", err)
	}
	defer eventConn.CloseNow()

	readNextSettingsUpdated := func() AppSettings {
		for {
			var ev struct {
				Name string          `json:"name"`
				Data json.RawMessage `json:"data"`
			}
			if err := wsjson.Read(ctx, eventConn, &ev); err != nil {
				t.Fatalf("read ws event failed: %v", err)
			}
			if ev.Name == "settings:updated" {
				var cfg AppSettings
				if err := json.Unmarshal(ev.Data, &cfg); err != nil {
					t.Fatalf("unmarshal settings:updated payload failed: %v", err)
				}
				return cfg
			}
		}
	}

	// 1. Desktop adds PLC + Tag -> WebSocket client receives settings:updated
	addedTag := TagSettings{
		Id:       uuid.New(),
		Name:     "DesktopAddedTag",
		PlcLink:  "PLC2",
		Address:  "DB1.DBD0",
		DataType: DataTypeReal,
		Enabled:  true,
	}
	desktopCfg := app.GetSettings()
	desktopCfg.PlcLinks = append(desktopCfg.PlcLinks, PlcLinkSettings{
		Name:      "PLC2",
		IpAddress: "10.0.0.2",
		Rack:      0,
		Slot:      1,
	})
	desktopCfg.Tags = []TagSettings{addedTag}
	app.SaveSettings(desktopCfg)

	wsGotAdded := readNextSettingsUpdated()
	if len(wsGotAdded.PlcLinks) != 2 || len(wsGotAdded.Tags) != 1 || wsGotAdded.Tags[0].Name != "DesktopAddedTag" {
		t.Fatalf("unexpected WebSocket settings after desktop add: %+v", wsGotAdded)
	}

	// 2. Remote web client removes PLC + Tag via /api/rpc/SaveSettings -> both desktop host and WS receive update
	remoteCfg := CreateDefaultSettings()
	resp, body := doRPC(t, baseURL, "SaveSettings", remoteCfg, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RPC SaveSettings failed %d: %s", resp.StatusCode, string(body))
	}

	wsGotRemoved := readNextSettingsUpdated()
	if len(wsGotRemoved.PlcLinks) != 1 || len(wsGotRemoved.Tags) != 0 {
		t.Fatalf("unexpected WebSocket settings after remote remove: %+v", wsGotRemoved)
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.emittedEvents) < 2 || host.emittedEvents[len(host.emittedEvents)-1] != "settings:updated" {
		t.Fatalf("expected desktop host to receive settings:updated from remote RPC, got %v", host.emittedEvents)
	}
}
