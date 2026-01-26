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

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Sender    string `json:"sender"`
	Message   string `json:"message"`
}

func generateLogFilename(basePath, format string) string {
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("ConanExiles_log_%s.%s", date, format)
	return filepath.Join(basePath, filename)
}

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

func logToTxt(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "txt")
	
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logLine := fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Sender, entry.Message)
	_, err = file.WriteString(logLine)
	return err
}

func logToCsv(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "csv")
	
	fileExists := true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fileExists = false
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {
		writer.Write([]string{"Timestamp", "Sender", "Message"})
	}

	return writer.Write([]string{entry.Timestamp, entry.Sender, entry.Message})
}

func logToJson(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "json")
	
	var entries []LogEntry
	
	if data, err := os.ReadFile(filename); err == nil {
		json.Unmarshal(data, &entries)
	}
	
	entries = append(entries, entry)
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func logToDocx(basePath string, entry LogEntry) error {
	filename := generateLogFilename(basePath, "txt")
	filename = strings.Replace(filename, ".txt", ".docx", 1)
	
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logLine := fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Sender, entry.Message)
	_, err = file.WriteString(logLine)
	return err
}