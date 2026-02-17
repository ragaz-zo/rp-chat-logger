package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const discordMessageLimit = 2000

var discordClient = &http.Client{
	Timeout: 10 * time.Second,
}

// QueuedMessage represents a message waiting to be sent to Discord.
type QueuedMessage struct {
	WebhookURL string
	Sender     string
	Message    string
	RetryAt    time.Time
	Attempts   int
}

// DiscordQueue manages rate-limited Discord messages with automatic retry.
type DiscordQueue struct {
	messages   []QueuedMessage
	mu         sync.Mutex
	notify     chan struct{}
	done       chan struct{}
	logger     *SSELogger
	maxRetries int
}

// NewDiscordQueue creates a new Discord message queue with background processing.
func NewDiscordQueue(logger *SSELogger) *DiscordQueue {
	q := &DiscordQueue{
		messages:   make([]QueuedMessage, 0),
		notify:     make(chan struct{}, 1),
		done:       make(chan struct{}),
		logger:     logger,
		maxRetries: 5,
	}
	go q.processLoop()
	return q
}

// Add queues a message for sending to Discord.
func (q *DiscordQueue) Add(msg QueuedMessage) {
	q.mu.Lock()
	q.messages = append(q.messages, msg)
	count := len(q.messages)
	q.mu.Unlock()

	if q.logger != nil {
		q.logger.Log("info", fmt.Sprintf("Message queued for Discord retry (queue size: %d)", count))
	}

	// Non-blocking notify
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// QueueSize returns the current number of queued messages.
func (q *DiscordQueue) QueueSize() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}

// Stop shuts down the queue processor.
func (q *DiscordQueue) Stop() {
	close(q.done)
}

func (q *DiscordQueue) processLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.done:
			return
		case <-q.notify:
			q.processMessages()
		case <-ticker.C:
			q.processMessages()
		}
	}
}

func (q *DiscordQueue) processMessages() {
	q.mu.Lock()
	if len(q.messages) == 0 {
		q.mu.Unlock()
		return
	}

	now := time.Now()
	var ready []QueuedMessage
	var pending []QueuedMessage

	for _, msg := range q.messages {
		if msg.RetryAt.Before(now) || msg.RetryAt.IsZero() {
			ready = append(ready, msg)
		} else {
			pending = append(pending, msg)
		}
	}

	q.messages = pending
	q.mu.Unlock()

	for _, msg := range ready {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		retryAfter, err := sendToDiscordWithRetry(ctx, msg.WebhookURL, msg.Sender, msg.Message)
		cancel()

		if err != nil {
			msg.Attempts++
			if retryAfter > 0 && msg.Attempts < q.maxRetries {
				// Rate limited - re-queue with retry time
				msg.RetryAt = time.Now().Add(retryAfter)
				q.Add(msg)
				if q.logger != nil {
					q.logger.Log("info", fmt.Sprintf("Discord rate limited, will retry in %v (attempt %d/%d)", retryAfter, msg.Attempts, q.maxRetries))
				}
			} else if msg.Attempts >= q.maxRetries {
				// Max retries exceeded
				log.Printf("Discord send failed after %d attempts: %v", msg.Attempts, err)
				if q.logger != nil {
					q.logger.Log("error", fmt.Sprintf("Discord send failed after %d attempts: %v", msg.Attempts, err))
					q.logger.LogFailure(msg.Sender, msg.Message, "discord", fmt.Sprintf("max retries exceeded: %v", err))
				}
			} else {
				// Non-rate-limit error
				log.Printf("Discord send failed: %v", err)
				if q.logger != nil {
					q.logger.Log("error", fmt.Sprintf("Discord send failed: %v", err))
					q.logger.LogFailure(msg.Sender, msg.Message, "discord", err.Error())
				}
			}
		} else if q.logger != nil {
			q.logger.Log("info", fmt.Sprintf("Queued message sent to Discord successfully (attempt %d)", msg.Attempts+1))
		}
	}
}

// sendToDiscordWithRetry sends a message and returns retry duration if rate limited.
// Returns (0, nil) on success, (retryAfter, error) on rate limit, (0, error) on other errors.
func sendToDiscordWithRetry(ctx context.Context, webhookURL, sender, message string) (time.Duration, error) {
	timestamp := time.Now().Format("15:04:05")
	base := fmt.Sprintf("**[%s] %s:** \n", timestamp, sender)

	chunks := splitMessage(base, message, discordMessageLimit-len(base))
	log.Printf("[DEBUG] Discord: sending %d chunk(s), message length=%d", len(chunks), len(message))

	for i, chunk := range chunks {
		payload := map[string]string{
			"content": chunk,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return 0, fmt.Errorf("marshaling discord payload: %w", err)
		}

		log.Printf("[DEBUG] Discord: chunk %d/%d, payload size=%d bytes", i+1, len(chunks), len(jsonData))

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return 0, fmt.Errorf("creating discord request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := discordClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("sending discord request: %w", err)
		}
		resp.Body.Close()

		log.Printf("[DEBUG] Discord: chunk %d/%d response status=%d", i+1, len(chunks), resp.StatusCode)

		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limited - extract Retry-After header
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfter := 5 * time.Second // default
			if retryAfterStr != "" {
				if seconds, err := strconv.ParseFloat(retryAfterStr, 64); err == nil {
					retryAfter = time.Duration(seconds*1000) * time.Millisecond
				}
			}
			return retryAfter, fmt.Errorf("rate limited by Discord")
		}

		if resp.StatusCode != http.StatusNoContent {
			return 0, fmt.Errorf("discord API returned status code: %d", resp.StatusCode)
		}
	}

	log.Printf("[DEBUG] Discord: all chunks sent successfully")
	return 0, nil
}

// sendToDiscord sends a chat message to a Discord webhook. If the message
// exceeds Discord's character limit, it is split into multiple chunks.
// Returns (rateLimited, retryAfter, error). If rateLimited is true, the caller
// should queue the message for retry after retryAfter duration.
func sendToDiscord(ctx context.Context, webhookURL, sender, message string) (bool, time.Duration, error) {
	retryAfter, err := sendToDiscordWithRetry(ctx, webhookURL, sender, message)
	if err != nil {
		if retryAfter > 0 {
			return true, retryAfter, err
		}
		return false, 0, err
	}
	return false, 0, nil
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
