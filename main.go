package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	hostname = "127.0.0.1"
	port     = 3000
)

var globalLogArea *widget.Entry

type ServerConfig struct {
	WebhookURL string
	Username   string
}

/*
func main() {
	config, err := loadConfiguration()
	if err != nil {
		log.Printf("Unable to load configuration: %v. Using default values.", err)
		config = &AppConfig{
			Port:       3000,
			FileFormat: "txt",
		}
	}

	appInstance := app.New()
	w := appInstance.NewWindow("Discord Notifier")

	var statusLabel *widget.Label
	var startButton, stopButton *widget.Button
	var webhookEntry, usernameEntry *widget.Entry
	var serverRunning bool

	// Server Configuration
	serverConfig := &ServerConfig{}

	// Set up the UI elements
	webhookEntry = widget.NewEntry()
	webhookEntry.PlaceHolder = "Discord Webhook URL"
	webhookEntry.Text = config.WebhookURL

	usernameEntry = widget.NewEntry()
	usernameEntry.PlaceHolder = "Username (for pinging)"
	usernameEntry.Text = config.Username

	startButton = widget.NewButton("Start Server", func() {
		serverConfig.WebhookURL = webhookEntry.Text
		serverConfig.Username = usernameEntry.Text

		if serverConfig.WebhookURL == "" {
			dialog.ShowError(errors.New("Please input the Discord webhook URL"), w)
			return
		}

		if !serverRunning {
			// Save configuration
			config.WebhookURL = serverConfig.WebhookURL
			config.Username = serverConfig.Username
			saveConfiguration(config)

			go startServer(serverConfig)
			serverRunning = true
			statusLabel.SetText("Server Status: Running")
		} else {
			dialog.ShowInformation("Server Already Running", "The server is already running!", w)
		}
	})

	stopButton = widget.NewButton("Stop Server", func() {
		if serverRunning {
			serverShutdown()
			serverRunning = false
			statusLabel.SetText("Server Status: Stopped")
		} else {
			dialog.ShowInformation("Server Not Running", "The server isn't running!", w)
		}
	})

	statusLabel = widget.NewLabel("Server Status: Stopped")

	content := container.NewVBox(
		widget.NewLabel("Enter the Discord Webhook URL:"),
		webhookEntry,
		widget.NewLabel("Enter the username for pinging:"),
		usernameEntry,
		container.NewHBox(startButton, stopButton),
		statusLabel,
	)

	w.SetContent(content)
	w.Resize(fyne.Size{Width: 600, Height: 300})
	w.CenterOnScreen()
	w.ShowAndRun()
}
*/

/*
func main() {
	log.Println("Application started.")

	config, err := loadConfiguration()
	if err != nil {
		log.Printf("Unable to load configuration: %v. Using default values.", err)
		config = &AppConfig{
			Port:       3000,
			FileFormat: "txt",
		}
	} else {
		log.Println("Configuration loaded successfully.")
	}

	appInstance := app.New()
	w := appInstance.NewWindow("Discord Notifier")
	log.Println("Main application window created.")

	var statusLabel *widget.Label
	var startButton, stopButton *widget.Button
	var webhookEntry, usernameEntry *widget.Entry
	var serverRunning bool

	// Server Configuration
	serverConfig := &ServerConfig{}
	log.Println("Server configuration initialized.")

	// Set up the UI elements
	webhookEntry = widget.NewEntry()
	webhookEntry.PlaceHolder = "Discord Webhook URL"
	webhookEntry.Text = config.WebhookURL
	log.Println("Webhook entry field created.")

	usernameEntry = widget.NewEntry()
	usernameEntry.PlaceHolder = "Username (for pinging)"
	usernameEntry.Text = config.Username
	log.Println("Username entry field created.")

	// Checkbox for auto start
	autoStartCheck := widget.NewCheck("Auto Start Server", func(checked bool) {
		config.AutoStart = checked
		saveConfiguration(config)
		if checked {
			log.Println("Auto Start Server option enabled.")
		} else {
			log.Println("Auto Start Server option disabled.")
		}
	})
	autoStartCheck.SetChecked(config.AutoStart)
	log.Println("Auto start checkbox added to UI.")

	// Buttons and labels (same as your existing code)
	startButton = widget.NewButton("Start Server", func() {

		log.Println("Start Server button clicked.")
		serverConfig.WebhookURL = webhookEntry.Text
		serverConfig.Username = usernameEntry.Text

		if serverConfig.WebhookURL == "" {
			dialog.ShowError(errors.New("Please input the Discord webhook URL"), w)
			log.Println("Error: Discord webhook URL is empty.")
			return
		}

		if !serverRunning {
			log.Println("Starting server...")
			// Save configuration
			config.WebhookURL = serverConfig.WebhookURL
			config.Username = serverConfig.Username
			saveConfiguration(config)
			log.Println("Configuration saved.")

			go startServer(serverConfig)
			serverRunning = true
			statusLabel.SetText("Server Status: Running")
			log.Println("Server started successfully.")
		} else {
			dialog.ShowInformation("Server Already Running", "The server is already running!", w)
			log.Println("Server start attempted but already running.")
		}
	})

	stopButton = widget.NewButton("Stop Server", func() {
		log.Println("Stop Server button clicked.")
		if serverRunning {
			log.Println("Stopping server...")
			serverShutdown()
			serverRunning = false
			statusLabel.SetText("Server Status: Stopped")
			log.Println("Server stopped successfully.")
		} else {
			dialog.ShowInformation("Server Not Running", "The server isn't running!", w)
			log.Println("Stop server attempted but server was not running.")
		}
	})

	statusLabel = widget.NewLabel("Server Status: Stopped")
	log.Println("Status label initialized.")

	content := container.NewVBox(
		webhookEntry,
		usernameEntry,
		autoStartCheck, // Add the checkbox to the layout
		startButton,
		stopButton,
		statusLabel,
	)
	log.Println("UI layout set up.")

	w.SetContent(content)

	// Automatically start server if AutoStart is enabled
	if config.AutoStart {
		log.Println("Auto start is enabled; starting server automatically.")
		go startServer(serverConfig)
		serverRunning = true
		statusLabel.SetText("Server Status: Running")
		log.Println("Server started automatically.")
	}

	w.ShowAndRun()
	log.Println("Application window is now running.")
}
*/

type AppConfig struct {
	WebhookURL      string
	DiscordID       string
	UserReplacer    map[string]string
	AutoStart       bool
	Path            string
	EnableDiscord   bool
	EnableLocalSave bool
	Port            int
	FileFormat      string
	DebugMode       bool
}

func main() {
	config, err := loadConfiguration()
	if err != nil {
		log.Printf("Unable to load configuration: %v. Using default values.", err)
		config = &AppConfig{
			Port:       3000,
			FileFormat: "txt",
		}
	}

	if config.Port == 0 {
		config.Port = 3000
	}
	if config.FileFormat == "" {
		config.FileFormat = "txt"
	}
	
	globalConfig = config

	appInstance := app.New()
	w := appInstance.NewWindow("Discord Notifier")

	var statusLabel *widget.Label
	var startButton, stopButton *widget.Button
	var webhookEntry, usernameEntry, discordIDEntry, pathEntry *widget.Entry
	var portEntry *widget.Entry
	var fileFormatSelect *widget.Select
	var discordContainer, localSaveContainer *fyne.Container
	var logTextArea *widget.Entry
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

	portEntry = widget.NewEntry()
	portEntry.PlaceHolder = "Port number"
	portEntry.Text = fmt.Sprintf("%d", config.Port)
	portEntry.Resize(fyne.NewSize(400, portEntry.Size().Height))

	fileFormatSelect = widget.NewSelect([]string{"txt", "docx"}, func(value string) {
		config.FileFormat = value
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
			config.Path = uri.Path()
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
		config.EnableDiscord = checked
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
		config.EnableLocalSave = checked
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

	autoStartCheck := widget.NewCheck("Auto Start Server", func(checked bool) {
		config.AutoStart = checked
		saveConfiguration(config)
	})
	autoStartCheck.SetChecked(config.AutoStart)

	debugCheck := widget.NewCheck("Debug mode (show all logs)", func(checked bool) {
		config.DebugMode = checked
		saveConfiguration(config)
		if checked {
			appendToLiveLogWithLevel(logTextArea, "info", "Debug mode enabled - showing all logs")
		} else {
			appendToLiveLogWithLevel(logTextArea, "info", "Debug mode disabled - showing only warnings and errors")
		}
	})
	debugCheck.SetChecked(config.DebugMode)

	startButton = widget.NewButton("Start Server", func() {
		portNum, err := strconv.Atoi(portEntry.Text)
		if err != nil || portNum < 1024 || portNum > 65535 {
			dialog.ShowError(errors.New("Please enter a valid port number (1024-65535)"), w)
			return
		}
		config.Port = portNum

		if config.EnableDiscord {
			config.WebhookURL = webhookEntry.Text
			config.DiscordID = discordIDEntry.Text
			config.UserReplacer = parseUsernameEntry(usernameEntry.Text, config.DiscordID)

			if config.WebhookURL == "" {
				dialog.ShowError(errors.New("Please input the Discord webhook URL"), w)
				return
			}
		}

		if config.EnableLocalSave {
			config.Path = pathEntry.Text
			if config.Path == "" {
				dialog.ShowError(errors.New("Please input the path for saving log files"), w)
				return
			}
		}

		if !config.EnableDiscord && !config.EnableLocalSave {
			dialog.ShowError(errors.New("Please enable at least one output option (Discord or File logging)"), w)
			return
		}

		if !serverRunning {
			saveConfiguration(config)
			appendToLiveLogWithLevel(logTextArea, "info", "Starting server...")
			go startServer(config)
			serverRunning = true
			statusLabel.SetText("Server Status: Running")
			appendToLiveLogWithLevel(logTextArea, "info", fmt.Sprintf("Server started on port %d", config.Port))
		} else {
			dialog.ShowInformation("Server Already Running", "The server is already running!", w)
			appendToLiveLogWithLevel(logTextArea, "warning", "Attempted to start already running server")
		}
	})

	stopButton = widget.NewButton("Stop Server", func() {
		if serverRunning {
			appendToLiveLogWithLevel(logTextArea, "info", "Stopping server...")
			serverShutdown()
			serverRunning = false
			statusLabel.SetText("Server Status: Stopped")
			appendToLiveLogWithLevel(logTextArea, "info", "Server stopped")
		} else {
			dialog.ShowInformation("Server Not Running", "The server isn't running!", w)
			appendToLiveLogWithLevel(logTextArea, "warning", "Attempted to stop server that wasn't running")
		}
	})

	buttons := container.NewHBox(startButton, stopButton)

	statusLabel = widget.NewLabel("Server Status: Stopped")

	logTextArea = widget.NewEntry()
	logTextArea.MultiLine = true
	logTextArea.Wrapping = fyne.TextWrapWord
	logTextArea.SetText("Server logs will appear here...\n")
	logTextArea.Disable()
	logTextArea.Resize(fyne.NewSize(580, 150))
	globalLogArea = logTextArea
	
	logScroll := container.NewScroll(logTextArea)
	logScroll.SetMinSize(fyne.NewSize(580, 150))
	logContainer := container.NewBorder(widget.NewLabel("Live Server Logs:"), nil, nil, nil, logScroll)

	portContainer := container.NewHBox(
		widget.NewLabel("Server Port:"),
		container.NewWithoutLayout(portEntry),
	)
	portEntry.Resize(fyne.NewSize(60, 36))
	portEntry.Move(fyne.NewPos(0, 0))

	separator1 := canvas.NewLine(color.Gray{Y: 128})
	separator1.Resize(fyne.NewSize(500, 1))
	
	separator2 := canvas.NewLine(color.Gray{Y: 128})
	separator2.Resize(fyne.NewSize(500, 1))

	separator3 := canvas.NewLine(color.Gray{Y: 128})
	separator3.Resize(fyne.NewSize(500, 1))

	content := container.NewVBox(
		enableDiscordCheck,
		discordContainer,
		separator1,
		enableLocalSaveCheck,
		localSaveContainer,
		separator2,
		autoStartCheck,
		debugCheck,
		portContainer,
		buttons,
		statusLabel,
		separator3,
		logContainer,
	)

	w.SetContent(content)

	if config.AutoStart {
		go startServer(config)
		serverRunning = true
		statusLabel.SetText("Server Status: Running")
	}

	w.Resize(fyne.NewSize(600, 700))
	w.ShowAndRun()
}

func appendToLiveLog(logArea *widget.Entry, message string) {
	appendToLiveLogWithLevel(logArea, "info", message)
}

func appendToLiveLogWithLevel(logArea *widget.Entry, level, message string) {
	if !globalConfig.DebugMode && level == "debug" {
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
	currentText := logArea.Text
	newText := currentText + logLine
	logArea.SetText(newText)
	logArea.CursorRow = len(strings.Split(newText, "\n")) - 1
}

var globalConfig *AppConfig

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

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
