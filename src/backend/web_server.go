package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	defaultWebServerHost = "0.0.0.0"
	defaultWebServerPort = 8080
	minWebServerPort     = 1024
	maxWebServerPort     = 65535
	webClientMarker      = `<script>window.__S7_WEB_SERVER__ = true;</script>`
)

// GetWebServerStatus returns the current status of the embedded HTTP/WebSocket server.
func (a *App) GetWebServerStatus() WebServerStatus {
	a.webMu.Lock()
	defer a.webMu.Unlock()

	host := a.webServerHost
	if strings.TrimSpace(host) == "" {
		host = defaultWebServerHost
	}
	port := a.webServerPort
	if port <= 0 {
		port = defaultWebServerPort
	}
	urlStr := ""
	if a.webRunning {
		urlStr = buildWebServerURL(host, port)
	}
	return WebServerStatus{
		Enabled:          a.webServerEnabled,
		Running:          a.webRunning,
		Host:             host,
		Port:             port,
		URL:              urlStr,
		Error:            a.webLastErr,
		ConnectedClients: int(a.webClients.Load()),
	}
}

// StartWebServer starts (or restarts on a new address) the embedded HTTP/WebSocket server.
func (a *App) StartWebServer(host string, port int) (WebServerStatus, error) {
	return a.startWebServer(host, port, false)
}

func (a *App) startWebServer(host string, port int, allowEphemeral bool) (WebServerStatus, error) {
	a.webOpMu.Lock()
	defer a.webOpMu.Unlock()

	a.webMu.Lock()
	if strings.TrimSpace(host) == "" {
		host = a.webServerHost
		if strings.TrimSpace(host) == "" {
			host = defaultWebServerHost
		}
	}
	if port == 0 && !allowEphemeral {
		port = a.webServerPort
		if port <= 0 {
			port = defaultWebServerPort
		}
	}
	if (!allowEphemeral || port != 0) && (port < minWebServerPort || port > maxWebServerPort) {
		a.webMu.Unlock()
		return a.GetWebServerStatus(), fmt.Errorf("web server port must be between %d and %d", minWebServerPort, maxWebServerPort)
	}
	if a.webRunning && a.webServerHost == host && port != 0 && a.webServerPort == port {
		a.webMu.Unlock()
		return a.GetWebServerStatus(), nil
	}
	wasRunning := a.webRunning
	a.webMu.Unlock()

	if wasRunning {
		a.stopWebServerLocked(false)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		a.webMu.Lock()
		a.webRunning = false
		a.webLastErr = err.Error()
		a.webMu.Unlock()
		a.refreshTrayWebServer()
		st := a.GetWebServerStatus()
		a.emitAppEvent("app:web-server-state", st)
		return st, err
	}

	actualPort := port
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok && tcpAddr.Port > 0 {
		actualPort = tcpAddr.Port
	}

	connCtx, connCancel := context.WithCancel(context.Background())
	srv := &http.Server{
		Handler:           a.newWebHandler(connCtx),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	a.webMu.Lock()
	a.webServerEnabled = true
	a.webServerHost = host
	a.webServerPort = actualPort
	a.webServer = srv
	a.webListener = ln
	a.webRunning = true
	a.webLastErr = ""
	a.webConnCancel = connCancel
	a.webMu.Unlock()

	go func() {
		serveErr := srv.Serve(ln)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			a.webMu.Lock()
			if a.webServer == srv {
				a.webRunning = false
				a.webLastErr = serveErr.Error()
				a.webServer = nil
				a.webListener = nil
				a.webConnCancel = nil
			}
			a.webMu.Unlock()
			a.refreshTrayWebServer()
			a.emitAppEvent("app:web-server-state", a.GetWebServerStatus())
		}
	}()

	a.persistWebServerState()
	a.refreshTrayWebServer()
	st := a.GetWebServerStatus()
	a.emitAppEvent("app:web-server-state", st)
	return st, nil
}

// StopWebServer stops the embedded HTTP/WebSocket server and disables auto-start.
func (a *App) StopWebServer() WebServerStatus {
	a.webOpMu.Lock()
	defer a.webOpMu.Unlock()
	return a.stopWebServerLocked(true)
}

func (a *App) shutdownWebServer() {
	a.webOpMu.Lock()
	defer a.webOpMu.Unlock()
	a.stopWebServerLocked(false)
}

func (a *App) stopWebServerLocked(persistDisable bool) WebServerStatus {
	a.webMu.Lock()
	srv := a.webServer
	ln := a.webListener
	cancel := a.webConnCancel
	if persistDisable {
		a.webServerEnabled = false
	}
	a.webServer = nil
	a.webListener = nil
	a.webConnCancel = nil
	a.webRunning = false
	a.webLastErr = ""
	a.webMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if srv != nil {
		ctx, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
		_ = srv.Shutdown(ctx)
		cancelTimeout()
	} else if ln != nil {
		_ = ln.Close()
	}

	if persistDisable {
		a.persistWebServerState()
	}
	a.refreshTrayWebServer()
	st := a.GetWebServerStatus()
	a.emitAppEvent("app:web-server-state", st)
	return st
}

// ToggleWebServer toggles the embedded HTTP/WebSocket server on or off.
func (a *App) ToggleWebServer() (WebServerStatus, error) {
	a.webMu.Lock()
	running := a.webRunning
	a.webMu.Unlock()

	if running {
		return a.StopWebServer(), nil
	}
	return a.StartWebServer("", 0)
}

// SetWebServerPort updates the configured web server TCP port and restarts the
// embedded server cleanly if it is currently running.
func (a *App) SetWebServerPort(port int) (WebServerStatus, error) {
	if port < minWebServerPort || port > maxWebServerPort {
		return a.GetWebServerStatus(), fmt.Errorf("web server port must be between %d and %d", minWebServerPort, maxWebServerPort)
	}

	a.webOpMu.Lock()
	a.webMu.Lock()
	wasRunning := a.webRunning
	host := a.webServerHost
	if strings.TrimSpace(host) == "" {
		host = defaultWebServerHost
	}
	if !wasRunning {
		a.webServerPort = port
		a.webLastErr = ""
	}
	a.webMu.Unlock()
	a.webOpMu.Unlock()

	if wasRunning {
		return a.StartWebServer(host, port)
	}

	a.persistWebServerState()
	a.refreshTrayWebServer()
	st := a.GetWebServerStatus()
	a.emitAppEvent("app:web-server-state", st)
	return st, nil
}

func (a *App) persistWebServerState() {
	a.mu.RLock()
	lastFile := a.lastSettingsPath
	a.mu.RUnlock()
	a.saveAppState(lastFile)
}

func (a *App) refreshTrayWebServer() {
	host := a.getHost()
	if host == nil {
		return
	}
	st := a.GetWebServerStatus()
	host.UpdateTrayWebServer(st.Running, formatTrayWebAddr(st.Host, st.Port))
}

func formatTrayWebAddr(host string, port int) string {
	if port <= 0 {
		port = defaultWebServerPort
	}
	trimmed := strings.TrimSpace(host)
	if trimmed == "" || trimmed == "0.0.0.0" || trimmed == "::" {
		return fmt.Sprintf(":%d", port)
	}
	return net.JoinHostPort(trimmed, fmt.Sprintf("%d", port))
}

func buildWebServerURL(host string, port int) string {
	if port <= 0 {
		return ""
	}
	displayHost := strings.TrimSpace(host)
	if displayHost == "" || displayHost == "0.0.0.0" || displayHost == "::" {
		displayHost = firstLocalIPv4()
	}
	return "http://" + net.JoinHostPort(displayHost, fmt.Sprintf("%d", port))
}

func firstLocalIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

func (a *App) newWebHandler(connCtx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/api/status", a.handleAPIStatus)
	mux.HandleFunc("/api/export/csv", a.handleAPIExportCSV)
	mux.HandleFunc("/api/ws/polling", func(w http.ResponseWriter, r *http.Request) {
		a.handleWSPolling(connCtx, w, r)
	})
	mux.HandleFunc("/api/ws/events", func(w http.ResponseWriter, r *http.Request) {
		a.handleWSEvents(connCtx, w, r)
	})
	mux.HandleFunc("/api/rpc/", a.handleAPIRPC)
	mux.Handle("/", a.newWebAssetHandler())
	return mux
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func (a *App) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, StatusSnapshot(a), r.Method == http.MethodHead)
}

func (a *App) handleAPIExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="trend.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	_ = WriteCSV(a, w)
}

func (a *App) handleWSPolling(connCtx context.Context, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()

	a.webClients.Add(1)
	if connCtx.Err() == nil {
		a.emitAppEvent("app:web-server-state", a.GetWebServerStatus())
	}
	defer func() {
		a.webClients.Add(-1)
		if connCtx.Err() == nil {
			a.emitAppEvent("app:web-server-state", a.GetWebServerStatus())
		}
	}()

	wsCtx, cancel := context.WithCancel(connCtx)
	defer cancel()
	go func() {
		select {
		case <-r.Context().Done():
			cancel()
		case <-wsCtx.Done():
		}
	}()

	readCtx := conn.CloseRead(wsCtx)
	ch, unsubscribe := a.SubscribeTelemetry(64)
	defer unsubscribe()

	for {
		select {
		case <-readCtx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case frame, ok := <-ch:
			if !ok {
				_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
				return
			}
			writeCtx, writeCancel := context.WithTimeout(readCtx, 5*time.Second)
			err := wsjson.Write(writeCtx, conn, frame)
			writeCancel()
			if err != nil {
				return
			}
		}
	}
}

func (a *App) handleWSEvents(connCtx context.Context, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()

	wsCtx, cancel := context.WithCancel(connCtx)
	defer cancel()
	go func() {
		select {
		case <-r.Context().Done():
			cancel()
		case <-wsCtx.Done():
		}
	}()

	readCtx := conn.CloseRead(wsCtx)
	ch, unsubscribe := a.subscribeEvents(64)
	defer unsubscribe()

	for {
		select {
		case <-readCtx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case ev, ok := <-ch:
			if !ok {
				_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
				return
			}
			writeCtx, writeCancel := context.WithTimeout(readCtx, 5*time.Second)
			err := wsjson.Write(writeCtx, conn, ev)
			writeCancel()
			if err != nil {
				return
			}
		}
	}
}

func (a *App) handleAPIRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if !authorizeSameOrigin(r) {
		writeJSONError(w, http.StatusForbidden, "cross-origin RPC requests are not permitted")
		return
	}

	method := strings.TrimPrefix(r.URL.Path, "/api/rpc/")
	if method == "" || strings.Contains(method, "/") {
		http.NotFound(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)

	switch method {
	case "GetVersion":
		writeJSON(w, http.StatusOK, a.GetVersion(), false)
	case "GetSystemLanguage":
		writeJSON(w, http.StatusOK, a.GetSystemLanguage(), false)
	case "GetSettings":
		writeJSON(w, http.StatusOK, a.GetSettings(), false)
	case "SaveSettings":
		var settings AppSettings
		if err := decodeRPCBody(r.Body, &settings); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.SaveSettings(settings)
		writeJSON(w, http.StatusOK, nil, false)
	case "StartPolling":
		var settings AppSettings
		if err := decodeRPCBody(r.Body, &settings); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.StartPolling(settings)
		writeJSON(w, http.StatusOK, nil, false)
	case "StopPolling":
		a.StopPolling()
		writeJSON(w, http.StatusOK, nil, false)
	case "IsPolling":
		writeJSON(w, http.StatusOK, a.IsPolling(), false)
	case "CheckStatus":
		var req struct {
			LinkName string `json:"linkName"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, a.CheckStatus(req.LinkName), false)
	case "Connect":
		var req struct {
			LinkName string `json:"linkName"`
			IP       string `json:"ip"`
			Rack     int    `json:"rack"`
			Slot     int    `json:"slot"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := a.Connect(req.LinkName, req.IP, req.Rack, req.Slot); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, nil, false)
	case "Disconnect":
		var req struct {
			LinkName string `json:"linkName"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := a.Disconnect(req.LinkName); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, nil, false)
	case "DisconnectAll":
		a.DisconnectAll()
		writeJSON(w, http.StatusOK, nil, false)
	case "TestConnection":
		var link PlcLinkSettings
		if err := decodeRPCBody(r.Body, &link); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := a.TestConnection(link); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, nil, false)
	case "GetHistoryRange":
		var req struct {
			TagIds    []string `json:"tagIds"`
			StartMs   int64    `json:"startMs"`
			EndMs     int64    `json:"endMs"`
			MaxPoints int      `json:"maxPoints"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		res, err := a.GetHistoryRange(r.Context(), req.TagIds, req.StartMs, req.EndMs, req.MaxPoints)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				writeJSONError(w, 499, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res, false)
	case "ClearHistory":
		a.ClearHistory()
		writeJSON(w, http.StatusOK, nil, false)
	case "RecordSample":
		var req struct {
			TagId       string  `json:"tagId"`
			TimestampMs int64   `json:"timestampMs"`
			Value       float64 `json:"value"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.RecordSample(req.TagId, req.TimestampMs, req.Value)
		writeJSON(w, http.StatusOK, nil, false)
	case "GetTreePanelWidth":
		writeJSON(w, http.StatusOK, a.GetTreePanelWidth(), false)
	case "SetTreePanelWidth":
		var req struct {
			Width int `json:"width"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.SetTreePanelWidth(req.Width)
		writeJSON(w, http.StatusOK, nil, false)
	case "HasSettingsChanged":
		writeJSON(w, http.StatusOK, a.HasSettingsChanged(), false)
	case "GetLastSettingsPath":
		writeJSON(w, http.StatusOK, a.GetLastSettingsPath(), false)
	case "ImportSettingsJSON":
		var req struct {
			RawJSON string `json:"rawJSON"`
		}
		if err := decodeRPCBody(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		settings, err := a.ImportSettingsJSON(req.RawJSON)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, settings, false)
	case "GetWebServerStatus":
		writeJSON(w, http.StatusOK, a.GetWebServerStatus(), false)
	default:
		http.NotFound(w, r)
	}
}

func (a *App) newWebAssetHandler() http.Handler {
	a.webMu.Lock()
	fsys := a.webAssets
	a.webMu.Unlock()

	if fsys == nil {
		return http.NotFoundHandler()
	}

	subFS := fsys
	if _, err := fs.Stat(fsys, "frontend/dist/index.html"); err == nil {
		if sub, err := fs.Sub(fsys, "frontend/dist"); err == nil {
			subFS = sub
		}
	}

	indexBytes, indexErr := fs.ReadFile(subFS, "index.html")
	var patchedIndex []byte
	if indexErr == nil {
		patchedIndex = injectWebClientMarker(indexBytes)
	}
	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/", "/index.html":
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				if indexErr != nil {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				w.WriteHeader(http.StatusOK)
				if r.Method != http.MethodHead {
					_, _ = w.Write(patchedIndex)
				}
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

func injectWebClientMarker(indexHTML []byte) []byte {
	lower := strings.ToLower(string(indexHTML))
	if idx := strings.Index(lower, "</head>"); idx >= 0 {
		out := make([]byte, 0, len(indexHTML)+len(webClientMarker)+1)
		out = append(out, indexHTML[:idx]...)
		out = append(out, []byte(webClientMarker+"\n")...)
		out = append(out, indexHTML[idx:]...)
		return out
	}
	out := make([]byte, 0, len(webClientMarker)+1+len(indexHTML))
	out = append(out, []byte(webClientMarker+"\n")...)
	out = append(out, indexHTML...)
	return out
}

func authorizeSameOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		return strings.EqualFold(u.Host, r.Host)
	}
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	return true
}

func decodeRPCBody(body io.Reader, dst any) error {
	dec := json.NewDecoder(body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any, headOnly bool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if headOnly {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message}, false)
}
