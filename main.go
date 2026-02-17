package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defaultListenAddr = "localhost:3000"
	defaultWebUIAddr  = "127.0.0.1:8080"
)

// App holds all shared application state including config, servers, and logging.
type App struct {
	config   *AppConfig
	configMu sync.RWMutex

	ingestionServer  *http.Server
	ingestionMu      sync.Mutex
	ingestionWg      sync.WaitGroup
	ingestionRunning atomic.Bool

	webServer     *http.Server
	sseBroker     *SSEBroker
	failureBroker *SSEBroker
	logger        *SSELogger
	discordQueue  *DiscordQueue
	updater       *Updater
	webAddr       string
}

// NewApp creates a new App with the given config and web UI address.
func NewApp(config *AppConfig, webAddr string) *App {
	broker := NewSSEBroker()
	failureBroker := NewSSEBroker()
	logger := NewSSELogger(broker, failureBroker)
	logger.SetDebugMode(config.DebugMode)
	discordQueue := NewDiscordQueue(logger)
	updater := NewUpdater(logger)

	return &App{
		config:        config,
		sseBroker:     broker,
		failureBroker: failureBroker,
		logger:        logger,
		discordQueue:  discordQueue,
		updater:       updater,
		webAddr:       webAddr,
	}
}

// Shutdown gracefully shuts down both servers and the SSE broker.
func (a *App) Shutdown() {
	if a.ingestionRunning.Load() {
		if err := a.StopIngestionServer(); err != nil {
			log.Printf("Error stopping ingestion server: %v", err)
		}
	}

	if a.webServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.webServer.Shutdown(ctx); err != nil {
			log.Printf("Error stopping web server: %v", err)
		}
	}

	a.sseBroker.Stop()
	a.failureBroker.Stop()
	a.discordQueue.Stop()
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func main() {
	configPath := flag.String("config", "", "path to config file (default: ~/.config/rp-chat-logger/config.json)")
	webAddr := flag.String("web-addr", defaultWebUIAddr, "web UI listen address")
	flag.Parse()

	if *configPath != "" {
		setConfigPath(*configPath)
	}

	config, err := loadConfiguration()
	if err != nil {
		log.Printf("Unable to load configuration: %v. Using default values.", err)
		config = &AppConfig{
			ListenAddr: defaultListenAddr,
			FileFormat: "txt",
		}
	}

	if config.ListenAddr == "" {
		config.ListenAddr = defaultListenAddr
	}
	if config.FileFormat == "" {
		config.FileFormat = "txt"
	}

	application := NewApp(config, *webAddr)

	// Cleanup old binary from previous update (Windows)
	CleanupOldBinary()

	// Check for updates in background
	go func() {
		if err := application.updater.CheckForUpdate(); err != nil {
			log.Printf("Update check failed: %v", err)
		}
	}()

	// Auto-start ingestion server if configured
	if config.AutoStart {
		if err := application.StartIngestionServer(); err != nil {
			log.Printf("Auto-start failed: %v", err)
		}
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start web UI server in a goroutine
	go func() {
		if err := application.StartWebUI(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Web UI server failed: %v", err)
		}
	}()

	// Open browser after a short delay to ensure server is ready
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:8080")
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	application.Shutdown()
}
