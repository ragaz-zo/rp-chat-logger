package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// StartWebUI starts the web UI HTTP server. This blocks until the server
// is shut down or encounters an error.
func (a *App) StartWebUI() error {
	mux := http.NewServeMux()

	// Serve embedded static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("creating static sub-filesystem: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Page routes
	mux.HandleFunc("GET /", a.handleIndex)

	// API routes for HTMX
	mux.HandleFunc("GET /api/config", a.handleGetConfig)
	mux.HandleFunc("PUT /api/config", a.handleUpdateConfig)
	mux.HandleFunc("POST /api/server/start", a.handleStartServer)
	mux.HandleFunc("POST /api/server/stop", a.handleStopServer)
	mux.HandleFunc("GET /api/server/status", a.handleServerStatus)

	// SSE endpoints
	mux.HandleFunc("GET /api/logs/stream", a.handleSSEStream)
	mux.HandleFunc("GET /api/failures/stream", a.handleFailureStream)

	// Shutdown endpoint
	mux.HandleFunc("POST /api/shutdown", a.handleShutdown)

	// Update endpoints
	mux.HandleFunc("GET /api/update/info", a.handleUpdateInfo)
	mux.HandleFunc("POST /api/update/check", a.handleUpdateCheck)
	mux.HandleFunc("POST /api/update/apply", a.handleUpdateApply)

	// Dialog endpoints
	mux.HandleFunc("GET /api/dialog/select-folder", a.handleSelectFolder)

	a.webServer = &http.Server{
		Addr:    a.webAddr,
		Handler: mux,
	}

	log.Printf("Web UI started at http://%s/", a.webAddr)
	a.logger.Log("info", fmt.Sprintf("Web UI available at http://%s/", a.webAddr))
	return a.webServer.ListenAndServe()
}

func (a *App) parseTemplates(files ...string) (*template.Template, error) {
	return template.ParseFS(templateFS, files...)
}

// handleIndex renders the main page.
func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	a.configMu.RLock()
	cfg := *a.config
	a.configMu.RUnlock()

	updateInfo := a.updater.GetInfo()
	data := map[string]interface{}{
		"Config":          cfg,
		"Running":         a.ingestionRunning.Load(),
		"Message":         a.statusMessage(),
		"Version":         Version,
		"UpdateAvailable": updateInfo.Available,
		"UpdateInfo":      updateInfo,
	}

	tmpl, err := a.parseTemplates(
		"templates/layout.html",
		"templates/index.html",
		"templates/partials/config_form.html",
		"templates/partials/status.html",
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Template render error: %v", err)
	}
}

// handleGetConfig returns the current config as an HTML partial.
func (a *App) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.RLock()
	cfg := *a.config
	a.configMu.RUnlock()

	data := map[string]interface{}{
		"Config": cfg,
	}

	tmpl, err := a.parseTemplates("templates/partials/config_form.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "config-form", data); err != nil {
		log.Printf("Template render error: %v", err)
	}
}

// handleUpdateConfig processes config form submission via HTMX.
func (a *App) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("debug", fmt.Sprintf("Config update request from %s", r.RemoteAddr))

	if err := r.ParseForm(); err != nil {
		a.logger.Log("debug", fmt.Sprintf("Failed to parse form: %v", err))
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	a.configMu.Lock()
	a.config.WebhookURL = r.FormValue("webhookURL")
	a.config.EnableDiscord = r.FormValue("enableDiscord") == "on"
	a.config.EnableLocalSave = r.FormValue("enableLocalSave") == "on"
	a.config.Path = r.FormValue("path")
	a.config.FileFormat = r.FormValue("fileFormat")
	a.config.ListenAddr = r.FormValue("listenAddr")
	a.config.AutoStart = r.FormValue("autoStart") == "on"
	a.config.DebugMode = r.FormValue("debugMode") == "on"
	cfg := *a.config
	a.configMu.Unlock()

	a.logger.SetDebugMode(cfg.DebugMode)

	a.logger.Log("debug", fmt.Sprintf("Config values: Discord=%v, LocalSave=%v (Path=%s, Format=%s), Listen=%s, AutoStart=%v, Debug=%v",
		cfg.EnableDiscord, cfg.EnableLocalSave, cfg.Path, cfg.FileFormat, cfg.ListenAddr, cfg.AutoStart, cfg.DebugMode))

	data := map[string]interface{}{
		"Config": cfg,
	}

	if err := saveConfiguration(&cfg); err != nil {
		a.logger.Log("error", fmt.Sprintf("Failed to save config: %v", err))
		data["SaveError"] = "Failed to save configuration"
	} else {
		a.logger.Log("info", "Configuration saved")
		a.logger.Log("debug", fmt.Sprintf("Config written to: %s", getConfigPath()))
		data["SaveSuccess"] = true
	}

	tmpl, err := a.parseTemplates("templates/partials/config_form.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "config-form", data); err != nil {
		log.Printf("Template render error: %v", err)
	}
}

// handleStartServer starts the message ingestion server.
func (a *App) handleStartServer(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("debug", fmt.Sprintf("Start server request from %s", r.RemoteAddr))

	if a.ingestionRunning.Load() {
		a.logger.Log("debug", "Server already running, ignoring start request")
		a.renderStatus(w, true, "Already running")
		return
	}

	a.configMu.RLock()
	cfg := *a.config
	a.configMu.RUnlock()

	if !cfg.EnableDiscord && !cfg.EnableLocalSave {
		a.logger.Log("debug", "Start rejected: no output options enabled")
		a.renderStatus(w, false, "Enable at least one output option")
		return
	}
	if cfg.EnableDiscord && cfg.WebhookURL == "" {
		a.logger.Log("debug", "Start rejected: Discord enabled but no webhook URL")
		a.renderStatus(w, false, "Discord webhook URL required")
		return
	}
	if cfg.EnableLocalSave && cfg.Path == "" {
		a.logger.Log("debug", "Start rejected: Local save enabled but no path")
		a.renderStatus(w, false, "File path required for local save")
		return
	}

	a.logger.Log("debug", fmt.Sprintf("Starting ingestion server on %s", cfg.ListenAddr))
	if err := a.StartIngestionServer(); err != nil {
		a.logger.Log("debug", fmt.Sprintf("Start failed: %v", err))
		a.renderStatus(w, false, fmt.Sprintf("Failed to start: %v", err))
		return
	}

	a.renderStatus(w, true, fmt.Sprintf("Running on %s", cfg.ListenAddr))
}

// handleStopServer stops the message ingestion server.
func (a *App) handleStopServer(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("debug", fmt.Sprintf("Stop server request from %s", r.RemoteAddr))

	if !a.ingestionRunning.Load() {
		a.logger.Log("debug", "Server not running, ignoring stop request")
		a.renderStatus(w, false, "Not running")
		return
	}

	a.logger.Log("debug", "Stopping ingestion server...")
	if err := a.StopIngestionServer(); err != nil {
		a.logger.Log("debug", fmt.Sprintf("Stop failed: %v", err))
		a.renderStatus(w, false, fmt.Sprintf("Failed to stop: %v", err))
		return
	}

	a.renderStatus(w, false, "Stopped")
}

// handleServerStatus returns the current server status as HTML partial.
func (a *App) handleServerStatus(w http.ResponseWriter, r *http.Request) {
	a.renderStatus(w, a.ingestionRunning.Load(), a.statusMessage())
}

func (a *App) statusMessage() string {
	if a.ingestionRunning.Load() {
		a.configMu.RLock()
		addr := a.config.ListenAddr
		a.configMu.RUnlock()
		return fmt.Sprintf("Running on %s", addr)
	}
	return "Stopped"
}

// renderStatus renders the status partial for HTMX.
func (a *App) renderStatus(w http.ResponseWriter, running bool, message string) {
	data := map[string]interface{}{
		"Running": running,
		"Message": message,
	}

	tmpl, err := a.parseTemplates("templates/partials/status.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "status-indicator", data); err != nil {
		log.Printf("Template render error: %v", err)
	}
}

// handleSSEStream serves a Server-Sent Events stream of log messages.
func (a *App) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("debug", fmt.Sprintf("SSE client connected from %s", r.RemoteAddr))

	flusher, ok := w.(http.Flusher)
	if !ok {
		a.logger.Log("debug", "SSE streaming not supported by client")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send recent history first
	history := a.logger.GetHistory()
	a.logger.Log("debug", fmt.Sprintf("Sending %d history lines to SSE client", len(history)))
	for _, line := range history {
		fmt.Fprintf(w, "data: <div class=\"log-line\">%s</div>\n\n", template.HTMLEscapeString(line))
	}
	flusher.Flush()

	// Subscribe to live updates
	ch := a.sseBroker.Subscribe()
	defer func() {
		a.sseBroker.Unsubscribe(ch)
		a.logger.Log("debug", fmt.Sprintf("SSE client disconnected: %s", r.RemoteAddr))
	}()

	ctx := r.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: <div class=\"log-line\">%s</div>\n\n", template.HTMLEscapeString(msg))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// handleFailureStream serves a Server-Sent Events stream of failure messages.
func (a *App) handleFailureStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send recent failure history first
	failures := a.logger.GetFailures()
	for _, f := range failures {
		line := fmt.Sprintf("[%s] %s | %s: %s | Error: %s",
			f.Timestamp, f.FailureType, f.Sender, truncateForDisplay(f.Message, 100), f.Error)
		fmt.Fprintf(w, "data: <div class=\"failure-line\">%s</div>\n\n", template.HTMLEscapeString(line))
	}
	flusher.Flush()

	// Subscribe to live failure updates
	ch := a.failureBroker.Subscribe()
	defer a.failureBroker.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: <div class=\"failure-line\">%s</div>\n\n", template.HTMLEscapeString(msg))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// truncateForDisplay shortens a string for display purposes.
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// handleShutdown handles the shutdown request, returning a shutdown page
// and then exiting the application after a brief delay.
func (a *App) handleShutdown(w http.ResponseWriter, r *http.Request) {
	a.logger.Log("info", "Shutdown requested via web UI")

	// Return shutdown page HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	shutdownHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RP Chat Logger - Shutdown</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #1a1a2e;
            color: #e0e0e0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .shutdown-container {
            text-align: center;
            padding: 40px;
            background: #16213e;
            border-radius: 12px;
            max-width: 500px;
        }
        h1 { color: #f87171; margin-bottom: 20px; }
        p { margin-bottom: 12px; color: #b0b0b0; }
        .instructions { margin-top: 24px; padding: 16px; background: #0f3460; border-radius: 8px; }
        .instructions p { margin-bottom: 8px; }
        .instructions p:last-child { margin-bottom: 0; }
    </style>
</head>
<body>
    <div class="shutdown-container">
        <h1>Application Shutdown</h1>
        <p>RP Chat Logger has been shut down.</p>
        <div class="instructions">
            <p>To start again, re-run the executable.</p>
            <p>You can close this browser tab.</p>
        </div>
    </div>
</body>
</html>`
	w.Write([]byte(shutdownHTML))

	// Shutdown application after a brief delay to allow response to be sent
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.Shutdown()
		os.Exit(0)
	}()
}

// handleUpdateInfo returns the current update information as an HTML partial.
func (a *App) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
	info := a.updater.GetInfo()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if info.Available {
		fmt.Fprintf(w, `<div class="update-badge">
			<span>v%s available</span>
			<button class="btn btn-update btn-small" hx-post="/api/update/apply" hx-swap="innerHTML" hx-target="body" hx-confirm="This will download and apply the update, then restart the application. Continue?">Update</button>
		</div>`, info.LatestVersion)
	} else {
		fmt.Fprintf(w, `<span class="update-check-result">Up to date (v%s)</span>`, info.CurrentVersion)
	}
}

// handleUpdateCheck triggers a check for updates and returns the result.
func (a *App) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if err := a.updater.CheckForUpdate(); err != nil {
		a.logger.Log("error", fmt.Sprintf("Update check failed: %v", err))
	}

	// Return the update info partial
	a.handleUpdateInfo(w, r)
}

// handleUpdateApply downloads and applies the update, then restarts.
func (a *App) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	info := a.updater.GetInfo()
	if !info.Available {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<div class="alert error">No update available</div>`))
		return
	}

	// Return updating page HTML first
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	updatingHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RP Chat Logger - Updating</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #1a1a2e;
            color: #e0e0e0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .update-container {
            text-align: center;
            padding: 40px;
            background: #16213e;
            border-radius: 12px;
            max-width: 500px;
        }
        h1 { color: #4ade80; margin-bottom: 20px; }
        p { margin-bottom: 12px; color: #b0b0b0; }
        .spinner {
            width: 40px;
            height: 40px;
            border: 4px solid #334155;
            border-top-color: #4ade80;
            border-radius: 50%%;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .instructions { margin-top: 24px; padding: 16px; background: #0f3460; border-radius: 8px; }
    </style>
    <script>
        // Auto-refresh after a delay to see if the new version is running
        setTimeout(function() {
            window.location.reload();
        }, 5000);
    </script>
</head>
<body>
    <div class="update-container">
        <h1>Updating...</h1>
        <div class="spinner"></div>
        <p>Downloading version %s</p>
        <div class="instructions">
            <p>The application will restart automatically.</p>
            <p>This page will refresh in a few seconds.</p>
        </div>
    </div>
</body>
</html>`, info.LatestVersion)
	w.Write([]byte(updatingHTML))

	// Perform update in background after response is sent
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := a.updater.PerformUpdate(); err != nil {
			a.logger.Log("error", fmt.Sprintf("Update failed: %v", err))
		}
	}()
}

// handleSelectFolder opens a native folder picker and returns the selected path.
func (a *App) handleSelectFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Use os/exec to open the native file picker
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows: Use PowerShell to open folder picker
		cmd = exec.CommandContext(r.Context(),
			"powershell", "-NoProfile", "-Command",
			`Add-Type -AssemblyName System.Windows.Forms; `+
				`$folder = New-Object System.Windows.Forms.FolderBrowserDialog; `+
				`if ($folder.ShowDialog() -eq 'OK') { Write-Host $folder.SelectedPath }`)
	case "darwin":
		// macOS: Use osascript to open file picker
		cmd = exec.CommandContext(r.Context(),
			"osascript", "-e",
			`tell application "System Events" to choose folder`)
	case "linux":
		// Linux: Use zenity if available
		cmd = exec.CommandContext(r.Context(), "zenity", "--file-selection", "--directory")
	default:
		fmt.Fprintf(w, `{"error":"Folder picker not supported on %s"}`, runtime.GOOS)
		return
	}

	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(w, `{"error":"User cancelled or error occurred"}`)
		return
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		fmt.Fprintf(w, `{"error":"No folder selected"}`)
		return
	}

	// On macOS, osascript returns a file URL, convert to path
	if runtime.GOOS == "darwin" && strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}

	fmt.Fprintf(w, `{"path":"%s"}`, path)
}
