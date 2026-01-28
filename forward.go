package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var forwardClient = &http.Client{
	Timeout: 10 * time.Second,
}

// forwardMessage sends a chat message as a JSON POST to the given URL.
// The payload contains "sender", "message", and "scene" fields.
func forwardMessage(ctx context.Context, url, sender, message, scene string) error {
	payload := map[string]string{
		"sender":  sender,
		"message": message,
		"scene":   scene,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling forward payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating forward request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := forwardClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending forward request: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("forward target returned status %d", resp.StatusCode)
	}

	return nil
}
