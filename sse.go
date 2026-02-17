package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SSEBroker manages SSE client connections and broadcasts log events.
type SSEBroker struct {
	clients    map[chan string]struct{}
	mu         sync.RWMutex
	register   chan chan string
	unregister chan chan string
	broadcast  chan string
	done       chan struct{}
}

// NewSSEBroker creates and starts a new SSE broker.
func NewSSEBroker() *SSEBroker {
	b := &SSEBroker{
		clients:    make(map[chan string]struct{}),
		register:   make(chan chan string),
		unregister: make(chan chan string),
		broadcast:  make(chan string, 256),
		done:       make(chan struct{}),
	}
	go b.run()
	return b
}

func (b *SSEBroker) run() {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = struct{}{}
			count := len(b.clients)
			b.mu.Unlock()
			log.Printf("[DEBUG] SSE: client registered, total clients=%d", count)
		case client := <-b.unregister:
			b.mu.Lock()
			delete(b.clients, client)
			close(client)
			count := len(b.clients)
			b.mu.Unlock()
			log.Printf("[DEBUG] SSE: client unregistered, total clients=%d", count)
		case msg := <-b.broadcast:
			b.mu.RLock()
			clientCount := len(b.clients)
			skipped := 0
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					skipped++
				}
			}
			b.mu.RUnlock()
			if skipped > 0 {
				log.Printf("[DEBUG] SSE: broadcast to %d clients, %d slow clients skipped", clientCount, skipped)
			}
		case <-b.done:
			log.Printf("[DEBUG] SSE: broker shutting down")
			return
		}
	}
}

// Subscribe returns a channel that receives log events.
func (b *SSEBroker) Subscribe() chan string {
	ch := make(chan string, 64)
	b.register <- ch
	return ch
}

// Unsubscribe removes a client channel.
func (b *SSEBroker) Unsubscribe(ch chan string) {
	b.unregister <- ch
}

// Publish sends a message to all subscribers.
func (b *SSEBroker) Publish(msg string) {
	b.broadcast <- msg
}

// Stop shuts down the broker goroutine.
func (b *SSEBroker) Stop() {
	close(b.done)
}

// FailureEntry represents a failed message processing attempt.
type FailureEntry struct {
	Timestamp   string
	Sender      string
	Message     string
	FailureType string // "discord", "file", "other"
	Error       string
}

// SSELogger implements the Logger interface, broadcasting logs
// to all connected SSE clients and keeping a ring buffer of recent history.
type SSELogger struct {
	broker        *SSEBroker
	failureBroker *SSEBroker
	debugMode     atomic.Bool
	history       []string
	historyMu     sync.RWMutex
	maxHistory    int
	failures      []FailureEntry
	failuresMu    sync.RWMutex
	maxFailures   int
}

// NewSSELogger creates a new SSE-backed logger.
func NewSSELogger(broker *SSEBroker, failureBroker *SSEBroker) *SSELogger {
	return &SSELogger{
		broker:        broker,
		failureBroker: failureBroker,
		maxHistory:    500,
		history:       make([]string, 0, 500),
		maxFailures:   100,
		failures:      make([]FailureEntry, 0, 100),
	}
}

// Log implements the Logger interface. It formats the message
// and broadcasts it via SSE to all connected clients.
func (l *SSELogger) Log(level, message string) {
	if l == nil {
		return
	}
	if !l.debugMode.Load() && level == "debug" {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	levelTag := ""
	switch level {
	case "error":
		levelTag = "[ERROR] "
	case "warning":
		levelTag = "[WARNING] "
	case "info":
		levelTag = "[INFO] "
	case "debug":
		levelTag = "[DEBUG] "
	}

	logLine := fmt.Sprintf("[%s] %s%s", timestamp, levelTag, message)

	l.historyMu.Lock()
	if len(l.history) >= l.maxHistory {
		l.history = l.history[1:]
	}
	l.history = append(l.history, logLine)
	l.historyMu.Unlock()

	l.broker.Publish(logLine)
}

// SetDebugMode updates whether debug-level messages are shown.
func (l *SSELogger) SetDebugMode(enabled bool) {
	l.debugMode.Store(enabled)
}

// GetHistory returns recent log lines for newly connected clients.
func (l *SSELogger) GetHistory() []string {
	l.historyMu.RLock()
	defer l.historyMu.RUnlock()
	result := make([]string, len(l.history))
	copy(result, l.history)
	return result
}

// GetHistoryText returns recent log history as a single joined string.
func (l *SSELogger) GetHistoryText() string {
	lines := l.GetHistory()
	return strings.Join(lines, "\n")
}

// LogFailure records a failed message processing attempt.
func (l *SSELogger) LogFailure(sender, message, failureType, errMsg string) {
	if l == nil {
		return
	}

	entry := FailureEntry{
		Timestamp:   time.Now().Format("15:04:05"),
		Sender:      sender,
		Message:     message,
		FailureType: failureType,
		Error:       errMsg,
	}

	l.failuresMu.Lock()
	if len(l.failures) >= l.maxFailures {
		l.failures = l.failures[1:]
	}
	l.failures = append(l.failures, entry)
	l.failuresMu.Unlock()

	// Broadcast formatted failure to SSE clients
	failureLine := fmt.Sprintf("[%s] %s | %s: %s | Error: %s",
		entry.Timestamp, entry.FailureType, entry.Sender, truncateMessage(entry.Message, 100), entry.Error)
	l.failureBroker.Publish(failureLine)
}

// GetFailures returns recent failure entries for newly connected clients.
func (l *SSELogger) GetFailures() []FailureEntry {
	l.failuresMu.RLock()
	defer l.failuresMu.RUnlock()
	result := make([]FailureEntry, len(l.failures))
	copy(result, l.failures)
	return result
}

// truncateMessage shortens a message to maxLen characters with ellipsis.
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}
