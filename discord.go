package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const discordMessageLimit = 2000

var discordClient = &http.Client{
	Timeout: 10 * time.Second,
}

// sendToDiscord sends a chat message to a Discord webhook. If the message
// exceeds Discord's character limit, it is split into multiple chunks.
func sendToDiscord(ctx context.Context, webhookURL, pingUser, sender, message string) error {
	timestamp := time.Now().Format("15:04:05")
	if pingUser != "" {
		pingUser = "<@" + pingUser + "> "
	}
	base := fmt.Sprintf("%s**[%s] %s:** \n", pingUser, timestamp, sender)

	chunks := splitMessage(base, message, discordMessageLimit-len(base))

	for _, chunk := range chunks {
		payload := map[string]string{
			"content": chunk,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshaling discord payload: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("creating discord request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := discordClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending discord request: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("discord API returned non-204 status code: %d", resp.StatusCode)
		}
	}

	return nil
}

// extractChunk splits a message at a word boundary within maxLength characters.
// It returns the chunk and any remaining text. If the message fits within
// maxLength, the remainder is empty.
func extractChunk(msg string, maxLength int) (string, string) {
	if len(msg) < maxLength {
		return msg, ""
	}

	if len(msg) == maxLength {
		return "...", "... " + msg
	}

	// Deduct 4 for " ..."
	end := maxLength - 4

	// Find the last space within the maxLength.
	for end > 0 && msg[end] != ' ' {
		end--
	}

	if end == 0 {
		return "...", "... " + msg
	}

	chunk := msg[:end] + " ..."
	remainder := "... " + msg[end+1:]

	return chunk, remainder
}

// splitMessage divides a message into Discord-safe chunks, each prefixed
// with the given base string (containing timestamp and sender info).
func splitMessage(base string, msg string, messageSize int) []string {
	var chunks []string
	remainingMessage := msg

	for len(remainingMessage) > 0 {
		chunk, remainder := extractChunk(remainingMessage, messageSize)
		chunks = append(chunks, base+chunk)

		if remainder != "" && !strings.HasPrefix(remainder, "...") {
			remainingMessage = "..." + remainder
		} else {
			remainingMessage = remainder
		}
	}
	return chunks
}
