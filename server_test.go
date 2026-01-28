package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfig() *AppConfig {
	return &AppConfig{
		ListenAddr:      "127.0.0.1:3000",
		FileFormat:      "txt",
		EnableDiscord:   false,
		EnableLocalSave: false,
	}
}

func TestCreateHandler_NoMessage(t *testing.T) {
	config := setupTestConfig()

	req, err := http.NewRequest("GET", "/message", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Errorf("Handler returned invalid JSON: %v", err)
	}

	if _, ok := response["ManifestFileVersion"]; !ok {
		t.Error("Response missing ManifestFileVersion field")
	}
}

func TestCreateHandler_MessageParams(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "with sender and message",
			url:            "/message?sender=TestUser&message=Hello+World",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing sender",
			url:            "/message?message=Hello+World",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing message",
			url:            "/message?sender=TestUser",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "special characters",
			url:            "/message?sender=Test%20User&message=Hello%21%20%3CWorld%3E",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := setupTestConfig()

			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			handlerFunc := createHandler(config, nil)
			handlerFunc.ServeHTTP(recorder, req)

			if status := recorder.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Errorf("Handler returned invalid JSON: %v", err)
			}
		})
	}
}

func TestCreateHandler_WithLocalSave(t *testing.T) {
	config := setupTestConfig()

	tmpDir, err := os.MkdirTemp("", "rp-chat-logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config.EnableLocalSave = true
	config.Path = tmpDir
	config.FileFormat = "txt"

	req, err := http.NewRequest("GET", "/message?sender=TestUser&message=Hello+World", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Error("Expected log file to be created, but no files found")
	}
}

func TestCreateHandler_LocalSaveInvalidPath(t *testing.T) {
	config := setupTestConfig()

	config.EnableLocalSave = true
	config.Path = "/nonexistent/invalid/path/that/should/not/exist"
	config.FileFormat = "txt"

	req, err := http.NewRequest("GET", "/message?sender=TestUser&message=Hello+World", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	// File logging failure doesn't cause HTTP error
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestCreateHandler_ResponseFields(t *testing.T) {
	config := setupTestConfig()

	req, err := http.NewRequest("GET", "/message", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	var response map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Handler returned invalid JSON: %v", err)
	}

	expectedFields := []string{
		"ManifestFileVersion",
		"bIsFileData",
		"AppID",
		"AppNameString",
		"BuildVersionString",
		"LaunchExeString",
		"LaunchCommand",
		"PrereqIds",
		"PrereqName",
		"PrereqPath",
		"PrereqArgs",
		"FileManifestList",
		"ChunkHashList",
		"ChunkShaList",
		"DataGroupList",
		"ChunkFilesizeList",
		"CustomFields",
	}

	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing expected field: %s", field)
		}
	}
}

func TestCreateHandler_ContentType(t *testing.T) {
	config := setupTestConfig()

	req, err := http.NewRequest("GET", "/message", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestCreateHandler_DocxFormat(t *testing.T) {
	config := setupTestConfig()

	tmpDir, err := os.MkdirTemp("", "rp-chat-logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config.EnableLocalSave = true
	config.Path = tmpDir
	config.FileFormat = "docx"

	req, err := http.NewRequest("GET", "/message?sender=TestUser&message=Hello+World", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handlerFunc := createHandler(config, nil)

	handlerFunc.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	foundDocx := false
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".docx" {
			foundDocx = true
			break
		}
	}

	if !foundDocx {
		t.Error("Expected .docx file to be created")
	}
}
