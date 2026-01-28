package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntry represents a single chat log record with a timestamp,
// sender name, and message body.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Sender    string `json:"sender"`
	Message   string `json:"message"`
}

// generateLogFilename returns the full file path for today's log file
// in the given format (e.g. "txt", "csv", "json", "docx").
func generateLogFilename(basePath, format string) string {
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("ConanExiles_log_%s.%s", date, format)
	return filepath.Join(basePath, filename)
}

// logToFile writes a chat message to a local file in the format
// specified by the config (txt, csv, json, or docx).
func logToFile(config *AppConfig, sender, message string) error {
	if !config.EnableLocalSave || config.Path == "" {
		return nil
	}

	logEntry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Sender:    sender,
		Message:   message,
	}

	switch config.FileFormat {
	case "txt":
		return logToTxt(config.Path, logEntry)
	case "csv":
		return logToCsv(config.Path, logEntry)
	case "json":
		return logToJson(config.Path, logEntry)
	case "docx":
		return logToDocx(config.Path, logEntry)
	default:
		return logToTxt(config.Path, logEntry)
	}
}

// logToTxt appends a log entry as a plain-text line to a .txt file.
func logToTxt(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "txt")

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening txt log file: %w", err)
	}
	defer file.Close()

	logLine := fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Sender, entry.Message)
	if _, err = file.WriteString(logLine); err != nil {
		return fmt.Errorf("writing to txt log file: %w", err)
	}
	return nil
}

// logToCsv appends a log entry as a CSV row, creating the header row
// if the file does not yet exist.
func logToCsv(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "csv")

	fileExists := true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fileExists = false
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening csv log file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {
		if err := writer.Write([]string{"Timestamp", "Sender", "Message"}); err != nil {
			return fmt.Errorf("writing csv header: %w", err)
		}
	}

	if err := writer.Write([]string{entry.Timestamp, entry.Sender, entry.Message}); err != nil {
		return fmt.Errorf("writing csv row: %w", err)
	}
	return nil
}

// logToJson appends a log entry to a JSON array file. Existing entries
// are read first and the new entry is appended.
func logToJson(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "json")

	var entries []LogEntry

	if data, err := os.ReadFile(filename); err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("parsing existing json log file: %w", err)
		}
	}

	entries = append(entries, entry)

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating json log file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		return fmt.Errorf("encoding json log entries: %w", err)
	}
	return nil
}

// logToDocx appends a log entry as a plain-text line to a .docx file.
func logToDocx(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "txt")
	filename = strings.Replace(filename, ".txt", ".docx", 1)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening docx log file: %w", err)
	}
	defer file.Close()

	logLine := fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Sender, entry.Message)
	if _, err = file.WriteString(logLine); err != nil {
		return fmt.Errorf("writing to docx log file: %w", err)
	}
	return nil
}
