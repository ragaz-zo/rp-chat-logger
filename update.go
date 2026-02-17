package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	githubOwner = "ragaz-zo"
	githubRepo  = "rp-chat-logger"
)

// GitHubRelease represents a GitHub release from the API.
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	PublishedAt string        `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a downloadable asset from a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateInfo holds information about an available update.
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	DownloadURL    string
	AssetName      string
	LastChecked    time.Time
}

// Updater handles checking for and applying updates.
type Updater struct {
	info   UpdateInfo
	mu     sync.RWMutex
	logger *SSELogger
}

// NewUpdater creates a new Updater instance.
func NewUpdater(logger *SSELogger) *Updater {
	return &Updater{
		logger: logger,
		info: UpdateInfo{
			CurrentVersion: Version,
		},
	}
}

// GetInfo returns the current update information.
func (u *Updater) GetInfo() UpdateInfo {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.info
}

// CheckForUpdate queries GitHub for the latest release and updates the info.
func (u *Updater) CheckForUpdate() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.logger != nil {
		u.logger.Log("info", "Checking for updates...")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "rp-chat-logger/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No releases yet
		u.info.Available = false
		u.info.LastChecked = time.Now()
		if u.logger != nil {
			u.logger.Log("info", "No releases found")
		}
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("decoding release: %w", err)
	}

	// Skip prereleases and drafts
	if release.Prerelease || release.Draft {
		u.info.Available = false
		u.info.LastChecked = time.Now()
		return nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	u.info.LatestVersion = latestVersion
	u.info.ReleaseURL = release.HTMLURL
	u.info.LastChecked = time.Now()

	// Compare versions
	if Version == "dev" || isNewerVersion(latestVersion, Version) {
		// Find the appropriate asset for this platform
		assetName := getAssetName()
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				u.info.Available = true
				u.info.DownloadURL = asset.BrowserDownloadURL
				u.info.AssetName = asset.Name
				if u.logger != nil {
					u.logger.Log("info", fmt.Sprintf("Update available: %s -> %s", Version, latestVersion))
				}
				return nil
			}
		}
		// Asset not found for this platform
		if u.logger != nil {
			u.logger.Log("info", fmt.Sprintf("Update %s available but no binary for %s/%s", latestVersion, runtime.GOOS, runtime.GOARCH))
		}
	} else {
		u.info.Available = false
		if u.logger != nil {
			u.logger.Log("info", fmt.Sprintf("Already on latest version (%s)", Version))
		}
	}

	return nil
}

// getAssetName returns the expected asset name for the current platform.
func getAssetName() string {
	if runtime.GOOS == "windows" {
		return "rp-chat-logger.exe"
	}
	return "rp-chat-logger"
}

// isNewerVersion returns true if latest is newer than current.
// Uses simple string comparison; assumes semver format (e.g., "1.2.3").
func isNewerVersion(latest, current string) bool {
	// Strip v prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		if latestParts[i] > currentParts[i] {
			return true
		} else if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return len(latestParts) > len(currentParts)
}

// PerformUpdate downloads and applies the update, then restarts the application.
func (u *Updater) PerformUpdate() error {
	u.mu.RLock()
	info := u.info
	u.mu.RUnlock()

	if !info.Available || info.DownloadURL == "" {
		return fmt.Errorf("no update available")
	}

	if u.logger != nil {
		u.logger.Log("info", fmt.Sprintf("Downloading update from %s...", info.DownloadURL))
	}

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Download the new binary to a temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(info.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file in same directory (for atomic rename)
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "rp-chat-logger-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("writing update: %w", err)
	}

	// Make executable (Unix only)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("setting permissions: %w", err)
		}
	}

	if u.logger != nil {
		u.logger.Log("info", "Download complete, applying update...")
	}

	// Apply the update
	if err := applyUpdate(execPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("applying update: %w", err)
	}

	if u.logger != nil {
		u.logger.Log("info", "Update applied, restarting...")
	}

	// Restart the application
	return restartApplication(execPath)
}

// applyUpdate replaces the current executable with the new one.
func applyUpdate(currentPath, newPath string) error {
	if runtime.GOOS == "windows" {
		// Windows: rename current exe to .old, then rename new to current
		oldPath := currentPath + ".old"

		// Remove any existing .old file
		os.Remove(oldPath)

		// Rename current to .old (Windows allows renaming running exe)
		if err := os.Rename(currentPath, oldPath); err != nil {
			return fmt.Errorf("renaming current executable: %w", err)
		}

		// Rename new to current
		if err := os.Rename(newPath, currentPath); err != nil {
			// Try to restore old
			os.Rename(oldPath, currentPath)
			return fmt.Errorf("renaming new executable: %w", err)
		}

		return nil
	}

	// Unix: just replace the file
	return os.Rename(newPath, currentPath)
}

// restartApplication restarts the application by spawning a new process.
func restartApplication(execPath string) error {
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting new process: %w", err)
	}

	// Exit the current process
	os.Exit(0)
	return nil
}

// CleanupOldBinary removes the .old backup file on startup (Windows).
func CleanupOldBinary() {
	if runtime.GOOS != "windows" {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	oldPath := execPath + ".old"

	// Try to remove, ignore errors (file might not exist)
	os.Remove(oldPath)
}
