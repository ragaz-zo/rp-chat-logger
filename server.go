package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Logger defines the interface for application-level logging.
type Logger interface {
	Log(level, message string)
}

var (
	server   *http.Server
	serverMu sync.Mutex
	serverWg sync.WaitGroup
)

// startServer creates and starts the HTTP server on the configured address.
// It registers the /message endpoint and blocks until the server is shut down.
func startServer(config *AppConfig, logger Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/message", createHandler(config, logger))

	addr := config.ListenAddr
	log.Printf("Server started at http://%s/", addr)

	serverMu.Lock()
	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	serverMu.Unlock()

	serverWg.Add(1)
	defer serverWg.Done()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Could not listen on %s: %v", addr, err)
		if logger != nil {
			logger.Log("error", fmt.Sprintf("Server failed: %v", err))
		}
	}
}

// serverShutdown gracefully shuts down the HTTP server with a 5-second timeout.
func serverShutdown() error {
	serverMu.Lock()
	srv := server
	serverMu.Unlock()

	if srv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}

	serverWg.Wait()
	log.Println("Server stopped.")
	return nil
}

// createHandler returns an HTTP handler that processes incoming chat messages
// and routes them to Discord and/or local file logging based on the config.
func createHandler(config *AppConfig, logger Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Snapshot config under read lock to avoid races with UI writes.
		configMu.RLock()
		cfg := *config
		configMu.RUnlock()

		sender, message := parseMessage(r)
		if message != "" {
			logMessage := fmt.Sprintf("Received message from %s: %s", sender, message)
			if logger != nil {
				logger.Log("debug", logMessage)
			}

			if cfg.EnableDiscord {
				err := sendToDiscord(ctx, cfg.WebhookURL, cfg.DiscordID, sender, message)
				if err != nil {
					log.Printf("Failed to send message to Discord: %v", err)
					if logger != nil {
						logger.Log("error", fmt.Sprintf("Failed to send to Discord: %v", err))
					}
					http.Error(w, "Failed to send message to Discord", http.StatusInternalServerError)
					return
				}
				if logger != nil {
					logger.Log("debug", "Message sent to Discord successfully")
				}
			}

			if cfg.EnableLocalSave {
				err := logToFile(&cfg, sender, message)
				if err != nil {
					log.Printf("Failed to log message to file: %v", err)
					if logger != nil {
						logger.Log("error", fmt.Sprintf("Failed to save to file: %v", err))
					}
				} else if logger != nil {
					logger.Log("debug", fmt.Sprintf("Message logged to %s file", cfg.FileFormat))
				}
			}

			if cfg.EnableHTTPForward {
				err := forwardMessage(ctx, cfg.ForwardURL, sender, message, cfg.ForwardScene)
				if err != nil {
					log.Printf("Failed to forward message via HTTP: %v", err)
					if logger != nil {
						logger.Log("error", fmt.Sprintf("Failed to forward via HTTP: %v", err))
					}
				} else if logger != nil {
					logger.Log("debug", "Message forwarded via HTTP successfully")
				}
			}
		}

		response := map[string]interface{}{
			"ManifestFileVersion": "000000000000",
			"bIsFileData":         false,
			"AppID":               "000000000000",
			"AppNameString":       "",
			"BuildVersionString":  "",
			"LaunchExeString":     "",
			"LaunchCommand":       "",
			"PrereqIds":           []string{},
			"PrereqName":          "",
			"PrereqPath":          "",
			"PrereqArgs":          "",
			"FileManifestList":    []string{},
			"ChunkHashList":       map[string]string{},
			"ChunkShaList":        map[string]string{},
			"DataGroupList":       map[string]string{},
			"ChunkFilesizeList":   map[string]string{},
			"CustomFields":        map[string]string{},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}
}
