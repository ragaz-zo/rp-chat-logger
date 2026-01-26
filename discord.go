package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const discordMessageLimit = 2000

func sendToDiscord(webhookURL, pingUser, sender, message string) error {
	timestamp := time.Now().Format("15:04:05")
	if pingUser != "" {
		pingUser = "<@" + pingUser + "> "
	}
	base := fmt.Sprintf("%s**[%s] %s:** \n", pingUser, timestamp, sender)

	// Split the message into chunks
	chunks := splitMessage(base, message, discordMessageLimit-len(base))

	for _, chunk := range chunks {
		payload := map[string]string{
			"content": chunk,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("Discord API returned non-204 status code: %d", resp.StatusCode)
		}
	}

	return nil
}

func extractChunk(msg string, maxLength int) (string, string) {
	// If the entire message is shorter than maxLength, return it as is without '...'
	if len(msg) < maxLength {
		return msg, ""
	}

	// If it's exactly the maxLength, return '...' and the message as the remainder.
	if len(msg) == maxLength {
		return "...", "... " + msg
	}

	// Deduct 4 for " ..."
	end := maxLength - 4

	// Find the last space within the maxLength.
	for end > 0 && msg[end] != ' ' {
		end--
	}

	// If no space was found in the maxLength, return '...' and the message as the remainder.
	if end == 0 {
		return "...", "... " + msg
	}

	chunk := msg[:end] + " ..."
	remainder := "... " + msg[end+1:]

	return chunk, remainder
}

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
