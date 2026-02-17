package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Logger defines the interface for application-level logging.
type Logger interface {
	Log(level, message string)
}

// StartIngestionServer creates and starts the message ingestion HTTP server.
func (a *App) StartIngestionServer() error {
	a.ingestionMu.Lock()
	defer a.ingestionMu.Unlock()

	if a.ingestionRunning.Load() {
		return fmt.Errorf("ingestion server already running")
	}

	a.configMu.RLock()
	addr := a.config.ListenAddr
	enableDiscord := a.config.EnableDiscord
	enableLocalSave := a.config.EnableLocalSave
	a.configMu.RUnlock()

	// Prevent starting if neither output option is enabled
	if !enableDiscord && !enableLocalSave {
		return fmt.Errorf("cannot start server: no output options are enabled. Enable either Discord notifications or file logging")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/message", createHandler(a))

	a.ingestionServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	a.ingestionWg.Add(1)
	a.ingestionRunning.Store(true)

	go func() {
		defer a.ingestionWg.Done()
		log.Printf("Ingestion server started at http://%s/", addr)
		a.logger.Log("info", fmt.Sprintf("Ingestion server started on %s", addr))
		if err := a.ingestionServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Could not listen on %s: %v", addr, err)
			a.logger.Log("error", fmt.Sprintf("Server failed: %v", err))
		}
		a.ingestionRunning.Store(false)
	}()

	return nil
}

// StopIngestionServer gracefully shuts down the message ingestion server.
func (a *App) StopIngestionServer() error {
	a.ingestionMu.Lock()
	srv := a.ingestionServer
	a.ingestionMu.Unlock()

	if srv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutting down ingestion server: %w", err)
	}

	a.ingestionWg.Wait()
	a.logger.Log("info", "Ingestion server stopped")
	log.Println("Ingestion server stopped.")
	return nil
}

// createHandler returns an HTTP handler that processes incoming chat messages
// and routes them to Discord and/or local file logging based on the config.
func createHandler(a *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Log incoming request details
		if a.logger != nil {
			a.logger.Log("debug", fmt.Sprintf("HTTP %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr))
			a.logger.Log("debug", fmt.Sprintf("User-Agent: %s", r.UserAgent()))
		}

		// Snapshot config under read lock to avoid races with UI writes.
		a.configMu.RLock()
		cfg := *a.config
		a.configMu.RUnlock()

		if a.logger != nil {
			a.logger.Log("debug", fmt.Sprintf("Config: Discord=%v, LocalSave=%v, Path=%s, Format=%s",
				cfg.EnableDiscord, cfg.EnableLocalSave, cfg.Path, cfg.FileFormat))
		}

		sender, message := parseMessage(r)
		if a.logger != nil {
			a.logger.Log("debug", fmt.Sprintf("Parsed: sender=%q, message=%q", sender, message))
		}

		if message != "" {
			a.logger.Log("info", fmt.Sprintf("Message from %s: %s", sender, message))

			if cfg.EnableDiscord {
				if a.logger != nil {
					// Redact webhook URL for security, show only host
					a.logger.Log("debug", "Sending to Discord webhook")
				}
				rateLimited, retryAfter, err := sendToDiscord(ctx, cfg.WebhookURL, sender, message)
				if err != nil {
					if rateLimited {
						// Queue for retry
						a.discordQueue.Add(QueuedMessage{
							WebhookURL: cfg.WebhookURL,
							Sender:     sender,
							Message:    message,
							RetryAt:    time.Now().Add(retryAfter),
							Attempts:   1,
						})
						if a.logger != nil {
							a.logger.Log("info", fmt.Sprintf("Discord rate limited, message queued for retry in %v", retryAfter))
						}
					} else {
						log.Printf("Failed to send message to Discord: %v", err)
						if a.logger != nil {
							a.logger.Log("error", fmt.Sprintf("Discord send failed: %v", err))
							a.logger.LogFailure(sender, message, "discord", err.Error())
						}
					}
					// Don't return error - game crashes on non-200 responses
				} else if a.logger != nil {
					a.logger.Log("debug", "Discord webhook returned success")
				}
			}

			if cfg.EnableLocalSave {
				fullPath := generateLogFilename(cfg.Path, cfg.FileFormat)
				if a.logger != nil {
					a.logger.Log("debug", fmt.Sprintf("Writing to file: %s", fullPath))
				}
				err := logToFile(&cfg, sender, message)
				if err != nil {
					log.Printf("Failed to log message to file: %v", err)
					if a.logger != nil {
						a.logger.Log("error", fmt.Sprintf("File write failed: %v", err))
						a.logger.LogFailure(sender, message, "file", err.Error())
					}
				} else if a.logger != nil {
					a.logger.Log("debug", fmt.Sprintf("Wrote to %s successfully", fullPath))
				}
			}
		} else if a.logger != nil {
			a.logger.Log("debug", "No message content, skipping processing")
		}

		// Always responds 200 OK to prevent the game from crashing, even if there are internal errors.
		response := map[string]string{"status": "ok"}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}
}
