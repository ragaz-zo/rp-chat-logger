package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const defaultListenAddr = "127.0.0.1:3000"

// configMu protects concurrent access to the shared AppConfig.
var configMu sync.RWMutex

// ServerConfig holds the webhook URL and username for Discord notifications.
type ServerConfig struct {
	WebhookURL string
	Username   string
}

// AppConfig holds the application configuration including Discord settings,
// file logging options, HTTP forwarding, and server parameters.
type AppConfig struct {
	WebhookURL        string
	DiscordID         string
	UserReplacer      map[string]string
	AutoStart         bool
	Path              string
	EnableDiscord     bool
	EnableLocalSave   bool
	EnableHTTPForward bool
	ForwardURL        string
	ForwardScene      string
	ListenAddr        string
	FileFormat        string
	DebugMode         bool
}

// widgetLogger implements the Logger interface using a Fyne widget.Entry.
type widgetLogger struct {
	logArea   *widget.Entry
	debugMode atomic.Bool
}

// Log writes a timestamped, level-tagged message to the UI log area.
func (l *widgetLogger) Log(level, message string) {
	if l == nil || l.logArea == nil {
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

	logLine := fmt.Sprintf("[%s] %s%s\n", timestamp, levelTag, message)
	fyne.Do(func() {
		newText := l.logArea.Text + logLine
		l.logArea.SetText(newText)
		l.logArea.CursorRow = len(strings.Split(newText, "\n")) - 1
	})
}

// SetDebugMode updates whether debug-level messages are shown.
func (l *widgetLogger) SetDebugMode(enabled bool) {
	l.debugMode.Store(enabled)
}

func main() {
	config, err := loadConfiguration()
	if err != nil {
		log.Printf("Unable to load configuration: %v. Using default values.", err)
		config = &AppConfig{
			ListenAddr: defaultListenAddr,
			FileFormat: "txt",
		}
	}

	if config.ListenAddr == "" {
		config.ListenAddr = defaultListenAddr
	}
	if config.FileFormat == "" {
		config.FileFormat = "txt"
	}

	appInstance := app.New()
	w := appInstance.NewWindow("Discord Notifier")

	var statusLabel *widget.Label
	var startButton, stopButton *widget.Button
	var webhookEntry, usernameEntry, discordIDEntry, pathEntry *widget.Entry
	var addrEntry *widget.Entry
	var fileFormatSelect *widget.Select
	var discordContainer, localSaveContainer, httpForwardContainer *fyne.Container
	var forwardURLEntry, forwardSceneEntry *widget.Entry
	var serverRunning bool

	webhookEntry = widget.NewEntry()
	webhookEntry.PlaceHolder = "Discord Webhook URL"
	webhookEntry.Text = config.WebhookURL

	discordIDEntry = widget.NewEntry()
	discordIDEntry.PlaceHolder = "Discord User ID"
	discordIDEntry.Text = config.DiscordID

	usernameEntry = widget.NewEntry()
	usernameEntry.PlaceHolder = "List of words to replace with Discord Username"
	usernameEntry.Text = strings.Join(mapKeys(config.UserReplacer), ", ")

	pathEntry = widget.NewEntry()
	pathEntry.PlaceHolder = "Path to save log files"
	pathEntry.Text = config.Path

	addrEntry = widget.NewEntry()
	addrEntry.PlaceHolder = "127.0.0.1:3000"
	addrEntry.Text = config.ListenAddr

	fileFormatSelect = widget.NewSelect([]string{"txt", "docx"}, func(value string) {
		configMu.Lock()
		config.FileFormat = value
		configMu.Unlock()
		saveConfiguration(config)
	})
	fileFormatSelect.SetSelected(config.FileFormat)

	discordContainer = container.NewVBox(
		widget.NewLabel("Discord Webhook URL:"),
		webhookEntry,
		widget.NewLabel("Discord User ID:"),
		discordIDEntry,
		widget.NewLabel("Words to replace with Discord Username:"),
		usernameEntry,
	)
	discordContainer.Hide()

	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
			configMu.Lock()
			config.Path = uri.Path()
			configMu.Unlock()
			saveConfiguration(config)
		}, w)
	})

	pathContainer := container.NewBorder(nil, nil, nil, browseButton, pathEntry)

	fileFormatContainer := container.NewHBox(
		widget.NewLabel("File format:"),
		container.NewPadded(fileFormatSelect),
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)

	localSaveContainer = container.NewVBox(
		widget.NewLabel("File path"),
		pathContainer,
		fileFormatContainer,
	)
	localSaveContainer.Hide()

	enableDiscordCheck := widget.NewCheck("Enable Discord notifications", func(checked bool) {
		configMu.Lock()
		config.EnableDiscord = checked
		configMu.Unlock()
		if checked {
			discordContainer.Show()
		} else {
			discordContainer.Hide()
		}
		saveConfiguration(config)
	})
	enableDiscordCheck.SetChecked(config.EnableDiscord)
	if config.EnableDiscord {
		discordContainer.Show()
	}

	enableLocalSaveCheck := widget.NewCheck("Enable file logging", func(checked bool) {
		configMu.Lock()
		config.EnableLocalSave = checked
		configMu.Unlock()
		if checked {
			localSaveContainer.Show()
		} else {
			localSaveContainer.Hide()
		}
		saveConfiguration(config)
	})
	enableLocalSaveCheck.SetChecked(config.EnableLocalSave)
	if config.EnableLocalSave {
		localSaveContainer.Show()
	}

	forwardURLEntry = widget.NewEntry()
	forwardURLEntry.PlaceHolder = "http://127.0.0.1:8080/endpoint"
	forwardURLEntry.Text = config.ForwardURL

	forwardSceneEntry = widget.NewEntry()
	forwardSceneEntry.PlaceHolder = "Scene name"
	forwardSceneEntry.Text = config.ForwardScene

	httpForwardContainer = container.NewVBox(
		widget.NewLabel("Forward URL:"),
		forwardURLEntry,
		widget.NewLabel("Scene:"),
		forwardSceneEntry,
	)
	httpForwardContainer.Hide()

	enableHTTPForwardCheck := widget.NewCheck("Enable HTTP forwarding", func(checked bool) {
		configMu.Lock()
		config.EnableHTTPForward = checked
		configMu.Unlock()
		if checked {
			httpForwardContainer.Show()
		} else {
			httpForwardContainer.Hide()
		}
		saveConfiguration(config)
	})
	enableHTTPForwardCheck.SetChecked(config.EnableHTTPForward)
	if config.EnableHTTPForward {
		httpForwardContainer.Show()
	}

	autoStartCheck := widget.NewCheck("Auto Start Server", func(checked bool) {
		configMu.Lock()
		config.AutoStart = checked
		configMu.Unlock()
		saveConfiguration(config)
	})
	autoStartCheck.SetChecked(config.AutoStart)

	// Create log area and logger
	logTextArea := widget.NewEntry()
	logTextArea.MultiLine = true
	logTextArea.Wrapping = fyne.TextWrapWord
	logTextArea.SetText("Server logs will appear here...\n")
	logTextArea.Disable()
	logTextArea.Resize(fyne.NewSize(580, 150))

	logger := &widgetLogger{logArea: logTextArea}
	logger.SetDebugMode(config.DebugMode)

	debugCheck := widget.NewCheck("Debug mode (show all logs)", func(checked bool) {
		configMu.Lock()
		config.DebugMode = checked
		configMu.Unlock()
		logger.SetDebugMode(checked)
		saveConfiguration(config)
		if checked {
			logger.Log("info", "Debug mode enabled - showing all logs")
		} else {
			logger.Log("info", "Debug mode disabled - showing only warnings and errors")
		}
	})
	debugCheck.SetChecked(config.DebugMode)

	startButton = widget.NewButton("Start Server", func() {
		addr := strings.TrimSpace(addrEntry.Text)
		if addr == "" {
			dialog.ShowError(errors.New("Please enter a listen address (e.g. 127.0.0.1:3000)"), w)
			return
		}

		configMu.Lock()
		config.ListenAddr = addr
		if config.EnableDiscord {
			config.WebhookURL = webhookEntry.Text
			config.DiscordID = discordIDEntry.Text
			config.UserReplacer = parseUsernameEntry(usernameEntry.Text, config.DiscordID)
		}
		if config.EnableLocalSave {
			config.Path = pathEntry.Text
		}
		if config.EnableHTTPForward {
			config.ForwardURL = forwardURLEntry.Text
			config.ForwardScene = forwardSceneEntry.Text
		}
		configMu.Unlock()

		if config.EnableDiscord && config.WebhookURL == "" {
			dialog.ShowError(errors.New("Please input the Discord webhook URL"), w)
			return
		}

		if config.EnableLocalSave && config.Path == "" {
			dialog.ShowError(errors.New("Please input the path for saving log files"), w)
			return
		}

		if config.EnableHTTPForward && config.ForwardURL == "" {
			dialog.ShowError(errors.New("Please input the HTTP forward URL"), w)
			return
		}

		if !config.EnableDiscord && !config.EnableLocalSave && !config.EnableHTTPForward {
			dialog.ShowError(errors.New("Please enable at least one output option (Discord, File logging, or HTTP forwarding)"), w)
			return
		}

		if !serverRunning {
			saveConfiguration(config)
			logger.Log("info", "Starting server...")
			go startServer(config, logger)
			serverRunning = true
			statusLabel.SetText("Server Status: Running")
			logger.Log("info", fmt.Sprintf("Server started on %s", config.ListenAddr))
		} else {
			dialog.ShowInformation("Server Already Running", "The server is already running!", w)
			logger.Log("warning", "Attempted to start already running server")
		}
	})

	stopButton = widget.NewButton("Stop Server", func() {
		if serverRunning {
			logger.Log("info", "Stopping server...")
			if err := serverShutdown(); err != nil {
				logger.Log("error", fmt.Sprintf("Shutdown error: %v", err))
			}
			serverRunning = false
			statusLabel.SetText("Server Status: Stopped")
			logger.Log("info", "Server stopped")
		} else {
			dialog.ShowInformation("Server Not Running", "The server isn't running!", w)
			logger.Log("warning", "Attempted to stop server that wasn't running")
		}
	})

	buttons := container.NewHBox(startButton, stopButton)

	statusLabel = widget.NewLabel("Server Status: Stopped")

	logScroll := container.NewScroll(logTextArea)
	logScroll.SetMinSize(fyne.NewSize(580, 150))
	logContainer := container.NewBorder(widget.NewLabel("Live Server Logs:"), nil, nil, nil, logScroll)

	addrContainer := container.NewVBox(
		widget.NewLabel("Listen Address:"),
		addrEntry,
	)

	separator1 := canvas.NewLine(color.Gray{Y: 128})
	separator1.Resize(fyne.NewSize(500, 1))

	separator2 := canvas.NewLine(color.Gray{Y: 128})
	separator2.Resize(fyne.NewSize(500, 1))

	separator3 := canvas.NewLine(color.Gray{Y: 128})
	separator3.Resize(fyne.NewSize(500, 1))

	separator4 := canvas.NewLine(color.Gray{Y: 128})
	separator4.Resize(fyne.NewSize(500, 1))

	content := container.NewVBox(
		enableDiscordCheck,
		discordContainer,
		separator1,
		enableLocalSaveCheck,
		localSaveContainer,
		separator2,
		enableHTTPForwardCheck,
		httpForwardContainer,
		separator3,
		autoStartCheck,
		debugCheck,
		addrContainer,
		buttons,
		statusLabel,
		separator4,
		logContainer,
	)

	w.SetContent(content)

	if config.AutoStart {
		go startServer(config, logger)
		serverRunning = true
		statusLabel.SetText("Server Status: Running")
	}

	w.Resize(fyne.NewSize(600, 700))
	w.ShowAndRun()
}

// parseUsernameEntry splits a comma-separated list of usernames
// and maps each to the given Discord user ID.
func parseUsernameEntry(entry, discordID string) map[string]string {
	replacer := make(map[string]string)
	entries := strings.Split(entry, ",")

	for _, e := range entries {
		word := strings.TrimSpace(e)
		if word != "" {
			replacer[word] = discordID
		}
	}
	return replacer
}

// mapKeys returns the keys of a string map as a slice.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
