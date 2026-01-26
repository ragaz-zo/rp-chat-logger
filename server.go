package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var server *http.Server

func startServer(config *AppConfig) {
	http.DefaultServeMux = new(http.ServeMux)
	http.HandleFunc("/message", createHandler(config))

	addr := fmt.Sprintf("%s:%d", hostname, config.Port)
	log.Printf("Server started at http://%s/", addr)

	server = &http.Server{
		Addr: addr,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", addr, err)
	}
}

func serverShutdown() {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server Shutdown Failed:%+v", err)
		}
		log.Println("Server stopped.")
	}
}

func createHandler(config *AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, message := parseMessage(r)
		if message != "" {
			logMessage := fmt.Sprintf("Received message from %s: %s", sender, message)
			if globalLogArea != nil {
				appendToLiveLogWithLevel(globalLogArea, "debug", logMessage)
			}
			
			if config.EnableDiscord {
				err := sendToDiscord(config.WebhookURL, config.DiscordID, sender, message)
				if err != nil {
					log.Printf("Failed to send message to Discord: %v", err)
					errorMsg := fmt.Sprintf("Failed to send to Discord: %v", err)
					if globalLogArea != nil {
						appendToLiveLogWithLevel(globalLogArea, "error", errorMsg)
					}
					http.Error(w, "Failed to send message to Discord", http.StatusInternalServerError)
					return
				} else {
					if globalLogArea != nil {
						appendToLiveLogWithLevel(globalLogArea, "debug", "Message sent to Discord successfully")
					}
				}
			}
			
			if config.EnableLocalSave {
				err := logToFile(config, sender, message)
				if err != nil {
					log.Printf("Failed to log message to file: %v", err)
					errorMsg := fmt.Sprintf("Failed to save to file: %v", err)
					if globalLogArea != nil {
						appendToLiveLogWithLevel(globalLogArea, "error", errorMsg)
					}
				} else {
					if globalLogArea != nil {
						appendToLiveLogWithLevel(globalLogArea, "debug", fmt.Sprintf("Message logged to %s file", config.FileFormat))
					}
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
		json.NewEncoder(w).Encode(response)
	}
}

