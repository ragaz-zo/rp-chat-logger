package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardMessage(t *testing.T) {
	tests := []struct {
		name         string
		sender       string
		message      string
		scene        string
		serverStatus int
		expectError  bool
	}{
		{
			name:         "successful forward",
			sender:       "TestUser",
			message:      "Hello World",
			scene:        "A dark tavern",
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:         "server returns error status",
			sender:       "TestUser",
			message:      "Hello World",
			scene:        "",
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received map[string]string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				received = make(map[string]string)
				if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
					t.Errorf("failed to decode payload: %v", err)
				}

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			err := forwardMessage(context.Background(), server.URL, tt.sender, tt.message, tt.scene)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if received["sender"] != tt.sender {
				t.Errorf("sender: expected %q, got %q", tt.sender, received["sender"])
			}
			if received["message"] != tt.message {
				t.Errorf("message: expected %q, got %q", tt.message, received["message"])
			}
			if received["scene"] != tt.scene {
				t.Errorf("scene: expected %q, got %q", tt.scene, received["scene"])
			}
		})
	}
}

func TestForwardMessage_UnreachableHost(t *testing.T) {
	err := forwardMessage(context.Background(), "http://127.0.0.1:1", "User", "msg", "scene")
	if err == nil {
		t.Error("expected error for unreachable host, got nil")
	}
}
