//go:build gui
// +build gui

package gui

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"go-socks5-chain/config"
	"go-socks5-chain/proxy"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	app                 fyne.App
	window              fyne.Window
	config              *config.Config
	encpass             string
	isNewUser           bool
	server              *proxy.Server
	serverMutex         sync.Mutex
	startButton         *widget.Button
	saveButton          *widget.Button
	browseButton        *widget.Button
	clearButton         *widget.Button
	copyButton          *widget.Button
	localHostEntry      *widget.Entry
	localPortEntry      *widget.Entry
	usernameEntry       *widget.Entry
	passwordEntry       *widget.Entry
	hostEntry           *widget.Entry
	portEntry           *widget.Entry
	logFileEntry        *widget.Entry
	logFileLabel        *widget.Label
	updateLogFileButtons func()
}

func NewGUI() *GUI {
	return &GUI{
		app: app.New(),
	}
}

func (g *GUI) Run() {
	g.app.Settings().SetTheme(&macOSTheme{})
	g.app.SetIcon(resourceIconPng)
	g.window = g.app.NewWindow("Go SOCKS5 Chain Configuration")
	g.window.SetIcon(resourceIconPng)
	// Set fixed window size for non-scrollable layout
	g.window.Resize(fyne.NewSize(650, 680))
	g.window.SetFixedSize(true)
	g.window.CenterOnScreen()

	// Check if configuration exists
	configExists := config.ConfigExists()

	if configExists {
		g.isNewUser = false
		g.showPasswordDialog()
	} else {
		g.isNewUser = true
		g.showFirstTimeSetup()
	}

	// Set up cleanup on window close
	g.window.SetOnClosed(func() {
		g.serverMutex.Lock()
		defer g.serverMutex.Unlock()
		if g.server != nil {
			g.server.Stop()
			g.server = nil
		}
	})

	g.window.ShowAndRun()
}

func (g *GUI) showPasswordDialog() {
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Enter encryption password"

	submitFunc := func() {
		g.encpass = passwordEntry.Text
		if err := g.loadConfiguration(); err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		g.showConfigurationEditor()
	}

	// Set up Enter key to submit
	passwordEntry.OnSubmitted = func(string) {
		submitFunc()
	}

	submitButton := widget.NewButton("Submit", submitFunc)
	submitButton.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabel("Configuration files found. Please enter your password to unlock."),
		container.NewBorder(nil, nil, widget.NewLabel("Password:"), nil, passwordEntry),
		container.NewCenter(submitButton),
	)

	g.window.SetContent(content)

	// Focus on the password entry field
	g.window.Canvas().Focus(passwordEntry)
}

func (g *GUI) showFirstTimeSetup() {
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Create access password"

	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.PlaceHolder = "Confirm password"

	submitFunc := func() {
		if passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Password cannot be empty"), g.window)
			return
		}
		if passwordEntry.Text != confirmEntry.Text {
			dialog.ShowError(fmt.Errorf("Passwords do not match"), g.window)
			return
		}
		g.encpass = passwordEntry.Text
		g.config = &config.Config{}
		g.showConfigurationEditor()
	}

	// Set up Enter key to submit on both fields
	passwordEntry.OnSubmitted = func(string) {
		submitFunc()
	}
	confirmEntry.OnSubmitted = func(string) {
		submitFunc()
	}

	submitButton := widget.NewButton("Submit", submitFunc)
	submitButton.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabel("Welcome to Go SOCKS5 Chain!"),
		widget.NewLabel("This appears to be your first time. Please set an access password."),
		container.NewBorder(nil, nil, widget.NewLabel("Access Password:"), nil, passwordEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Confirm Password:"), nil, confirmEntry),
		container.NewCenter(submitButton),
	)

	g.window.SetContent(content)

	// Focus on the password entry field
	g.window.Canvas().Focus(passwordEntry)
}

func (g *GUI) showConfigurationEditor() {
	// Store original values for change detection
	originalUsername := ""
	originalPassword := ""
	originalHost := ""
	originalPort := ""
	originalLocalHost := "127.0.0.1"
	originalLocalPort := "1080"
	originalLogFile := ""
	if g.config != nil {
		originalUsername = g.config.Username
		originalPassword = g.config.Password
		originalHost = g.config.UpstreamHost
		originalPort = strconv.Itoa(g.config.UpstreamPort)
		if g.config.LocalHost != "" {
			originalLocalHost = g.config.LocalHost
		}
		if g.config.LocalPort > 0 {
			originalLocalPort = strconv.Itoa(g.config.LocalPort)
		}
		originalLogFile = g.config.LogFile
	}

	// Create form fields
	g.usernameEntry = widget.NewEntry()
	g.usernameEntry.PlaceHolder = "SOCKS5 Username"
	if g.config != nil {
		g.usernameEntry.Text = g.config.Username
	}

	g.passwordEntry = widget.NewPasswordEntry()
	g.passwordEntry.PlaceHolder = "SOCKS5 Password"
	if g.config != nil {
		g.passwordEntry.Text = g.config.Password
	}

	g.hostEntry = widget.NewEntry()
	g.hostEntry.PlaceHolder = "proxy.example.com"
	if g.config != nil {
		g.hostEntry.Text = g.config.UpstreamHost
	}

	g.portEntry = widget.NewEntry()
	g.portEntry.PlaceHolder = "1080"
	if g.config != nil && g.config.UpstreamPort > 0 {
		g.portEntry.Text = strconv.Itoa(g.config.UpstreamPort)
	}

	g.localHostEntry = widget.NewEntry()
	g.localHostEntry.PlaceHolder = "127.0.0.1"
	if g.config != nil && g.config.LocalHost != "" {
		g.localHostEntry.Text = g.config.LocalHost
	} else {
		g.localHostEntry.Text = "127.0.0.1"
	}

	g.localPortEntry = widget.NewEntry()
	g.localPortEntry.PlaceHolder = "1080"
	if g.config != nil && g.config.LocalPort > 0 {
		g.localPortEntry.Text = strconv.Itoa(g.config.LocalPort)
	} else {
		g.localPortEntry.Text = "1080"
	}

	// Use a label with entry-like styling to avoid scrollbars
	logFileText := "No log file selected"
	if g.config != nil && g.config.LogFile != "" {
		logFileText = g.config.LogFile
	}
	
	// Create a label that looks like a disabled entry
	g.logFileLabel = widget.NewLabel(logFileText)
	// Labels truncate by default when they exceed container width
	
	// Keep the entry for compatibility but make it hidden
	g.logFileEntry = widget.NewEntry()
	g.logFileEntry.Text = logFileText
	g.logFileEntry.Disable()
	g.logFileEntry.Hide() // Hide the actual entry

	// Determine button text based on whether it's new or existing config
	buttonText := "Save"
	if g.isNewUser == false {
		buttonText = "Update"
	}

	// Create save/update function
	saveFunc := func() {
		// Validate input
		if g.usernameEntry.Text == "" || g.passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Username and password are required"), g.window)
			return
		}
		if g.hostEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Upstream host is required"), g.window)
			return
		}

		port, err := strconv.Atoi(g.portEntry.Text)
		if err != nil || port <= 0 || port > 65535 {
			dialog.ShowError(fmt.Errorf("Invalid upstream port"), g.window)
			return
		}

		localPort, err := strconv.Atoi(g.localPortEntry.Text)
		if err != nil || localPort <= 0 || localPort > 65535 {
			dialog.ShowError(fmt.Errorf("Invalid local port"), g.window)
			return
		}

		// Update configuration
		g.config = &config.Config{
			Username:     g.usernameEntry.Text,
			Password:     g.passwordEntry.Text,
			UpstreamHost: g.hostEntry.Text,
			UpstreamPort: port,
			LocalHost:    g.localHostEntry.Text,
			LocalPort:    localPort,
			LogFile:      g.logFileEntry.Text,
		}

		// Save configuration
		if err := g.saveConfiguration(); err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		successMsg := "Configuration saved successfully!"
		if buttonText == "Update" {
			successMsg = "Configuration updated successfully!"
		}
		dialog.ShowInformation("Success", successMsg, g.window)

		// Update original values after successful save
		originalUsername = g.usernameEntry.Text
		originalPassword = g.passwordEntry.Text
		originalHost = g.hostEntry.Text
		originalPort = g.portEntry.Text
		originalLocalHost = g.localHostEntry.Text
		originalLocalPort = g.localPortEntry.Text
		originalLogFile = g.logFileEntry.Text

		// Enable start button after successful save
		if g.startButton != nil {
			g.startButton.Enable()
		}
	}

	// Create save button (disabled by default)
	g.saveButton = widget.NewButton(buttonText, saveFunc)
	g.saveButton.Importance = widget.HighImportance
	g.saveButton.Disable()

	// Function to check for changes
	checkChanges := func() {
		hasChanges := false

		// Check if any field has changed from original
		if g.isNewUser {
			// For new users, check if required fields have values
			hasChanges = g.usernameEntry.Text != "" && g.passwordEntry.Text != "" &&
				g.hostEntry.Text != "" && g.portEntry.Text != ""
		} else {
			// For existing users, check if any field differs from original
			hasChanges = g.usernameEntry.Text != originalUsername ||
				g.passwordEntry.Text != originalPassword ||
				g.hostEntry.Text != originalHost ||
				g.portEntry.Text != originalPort ||
				g.localHostEntry.Text != originalLocalHost ||
				g.localPortEntry.Text != originalLocalPort ||
				g.logFileEntry.Text != originalLogFile
		}

		// Only enable save button if there are changes AND server is not running
		g.serverMutex.Lock()
		serverRunning := g.server != nil
		g.serverMutex.Unlock()

		if hasChanges && !serverRunning {
			g.saveButton.Enable()
		} else {
			g.saveButton.Disable()
		}
	}

	// Add change listeners to all fields
	g.usernameEntry.OnChanged = func(string) { checkChanges() }
	g.passwordEntry.OnChanged = func(string) { checkChanges() }
	g.hostEntry.OnChanged = func(string) { checkChanges() }
	g.portEntry.OnChanged = func(string) { checkChanges() }
	g.localHostEntry.OnChanged = func(string) { checkChanges() }
	g.localPortEntry.OnChanged = func(string) { checkChanges() }
	g.logFileEntry.OnChanged = func(string) { checkChanges() }

	// Create modern form layout with cards and better spacing
	formContent := container.NewVBox()

	// Upstream Proxy Settings Card
	upstreamCard := widget.NewCard("", "Upstream Proxy Settings", container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel("Username:"), g.usernameEntry,
			widget.NewLabel("Password:"), g.passwordEntry,
			widget.NewLabel("Host:"), g.hostEntry,
			widget.NewLabel("Port:"), g.portEntry,
		),
	))
	formContent.Add(upstreamCard)

	// Local Server Settings Card with some spacing
	localCard := widget.NewCard("", "Local Server Settings", container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel("Bind Host:"), g.localHostEntry,
			widget.NewLabel("Bind Port:"), g.localPortEntry,
		),
	))
	formContent.Add(localCard)

	// Create copy button (initially hidden)
	g.copyButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if g.logFileEntry.Text != "" {
			g.window.Clipboard().SetContent(g.logFileEntry.Text)
			// Show brief notification
			dialog.ShowInformation("Copied", "Log file path copied to clipboard", g.window)
		}
	})
	g.copyButton.Importance = widget.LowImportance
	
	// Create browse button for log file selection
	g.browseButton = widget.NewButton("Browse", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			writer.Close()
			path := writer.URI().Path()
			g.logFileEntry.SetText(path)
			g.logFileLabel.SetText(path)
			g.updateLogFileButtons()
		}, g.window)
	})
	g.browseButton.Importance = widget.LowImportance
	
	// Create clear button to remove log file selection
	g.clearButton = widget.NewButton("Clear", func() {
		g.logFileEntry.SetText("")
		g.logFileLabel.SetText("No log file selected")
		g.updateLogFileButtons()
	})
	g.clearButton.Importance = widget.LowImportance
	
	// Create file button container - will be updated dynamically
	fileButtonContainer := container.NewHBox()
	g.updateLogFileButtons = func() {
		if g.logFileEntry.Text != "" {
			fileButtonContainer.Objects = []fyne.CanvasObject{g.copyButton, g.browseButton, g.clearButton}
		} else {
			fileButtonContainer.Objects = []fyne.CanvasObject{g.browseButton}
		}
		fileButtonContainer.Refresh()
	}
	
	// Set initial button state
	g.updateLogFileButtons()
	
	// Create a fixed-size container for the log file label
	logFileDisplay := container.NewWithoutLayout(g.logFileLabel)
	// Set fixed size for the label container to prevent layout changes
	logFileDisplay.Resize(fyne.NewSize(350, 24)) // Fixed width and height
	g.logFileLabel.Resize(fyne.NewSize(350, 24))
	g.logFileLabel.Move(fyne.NewPos(0, 0))
	
	// Create log file input with browse button - use label to avoid scrollbars
	logFileContainer := container.NewBorder(nil, nil, nil, fileButtonContainer, logFileDisplay)
	
	// Optional Settings Card
	optionalCard := widget.NewCard("", "Optional Settings", container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel("Log File:"), logFileContainer,
		),
	))
	formContent.Add(optionalCard)

	// Create start/stop button
	g.startButton = widget.NewButton("Start", func() {
		g.toggleServer()
	})
	g.startButton.Importance = widget.SuccessImportance

	// Disable start button if no config exists
	if g.config == nil || g.config.UpstreamHost == "" {
		g.startButton.Disable()
	}

	// Add save and start/stop buttons with better styling
	// Create button container with proper spacing
	buttonContainer := container.NewHBox(
		g.saveButton,
		widget.NewLabel(""), // Add spacing between buttons
		g.startButton,
	)
	
	// Add the buttons in a padded container
	buttonSection := container.NewPadded(
		container.NewCenter(buttonContainer),
	)
	formContent.Add(buttonSection)

	var title string
	if g.isNewUser {
		title = "Configure SOCKS5 Proxy Settings"
	} else {
		title = "Edit SOCKS5 Proxy Settings"
	}

	// Create a modern header with better typography
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerContainer := container.NewVBox(
		container.NewPadded(titleLabel),
		widget.NewSeparator(),
	)

	// Use border layout without scroll for fixed height
	content := container.NewBorder(
		headerContainer,
		nil, nil, nil,
		container.NewPadded(formContent),
	)

	g.window.SetContent(content)

	// Focus on the username field
	g.window.Canvas().Focus(g.usernameEntry)
}

func (g *GUI) loadConfiguration() error {
	cfg, err := config.LoadOrCreate("", "", g.encpass, "", 0)
	if err != nil {
		return fmt.Errorf("Failed to load configuration: %v", err)
	}
	g.config = cfg
	return nil
}

func (g *GUI) saveConfiguration() error {
	// Ensure configuration directory exists
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("Failed to create config directory: %v", err)
	}

	// Save the configuration
	if err := config.SaveConfig(g.config, g.encpass); err != nil {
		return fmt.Errorf("Failed to save configuration: %v", err)
	}

	return nil
}

func (g *GUI) setFormFieldsEnabled(enabled bool) {
	if g.usernameEntry != nil {
		if enabled {
			g.usernameEntry.Enable()
		} else {
			g.usernameEntry.Disable()
		}
	}
	if g.passwordEntry != nil {
		if enabled {
			g.passwordEntry.Enable()
		} else {
			g.passwordEntry.Disable()
		}
	}
	if g.hostEntry != nil {
		if enabled {
			g.hostEntry.Enable()
		} else {
			g.hostEntry.Disable()
		}
	}
	if g.portEntry != nil {
		if enabled {
			g.portEntry.Enable()
		} else {
			g.portEntry.Disable()
		}
	}
	if g.localHostEntry != nil {
		if enabled {
			g.localHostEntry.Enable()
		} else {
			g.localHostEntry.Disable()
		}
	}
	if g.localPortEntry != nil {
		if enabled {
			g.localPortEntry.Enable()
		} else {
			g.localPortEntry.Disable()
		}
	}
	// Log file entry remains disabled as it's read-only
	if g.browseButton != nil {
		if enabled {
			g.browseButton.Enable()
		} else {
			g.browseButton.Disable()
		}
	}
	if g.clearButton != nil {
		if enabled {
			g.clearButton.Enable()
		} else {
			g.clearButton.Disable()
		}
	}
	if g.copyButton != nil {
		if enabled {
			g.copyButton.Enable()
		} else {
			g.copyButton.Disable()
		}
	}
}

func (g *GUI) toggleServer() {
	g.serverMutex.Lock()
	defer g.serverMutex.Unlock()

	if g.server == nil {
		// Start server
		g.startServer()
	} else {
		// Stop server
		g.stopServer()
	}
}

func (g *GUI) startServer() {
	// Load configuration
	cfg, err := config.LoadOrCreate("", "", g.encpass, "", 0)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load configuration: %v", err), g.window)
		return
	}

	// Get local host and port from form fields or use defaults
	localHost := "127.0.0.1"
	localPort := 1080

	// Use the stored entry values if available
	if g.localHostEntry != nil && g.localHostEntry.Text != "" {
		localHost = g.localHostEntry.Text
	}
	if g.localPortEntry != nil && g.localPortEntry.Text != "" {
		port, err := strconv.Atoi(g.localPortEntry.Text)
		if err == nil && port > 0 && port <= 65535 {
			localPort = port
		}
	}

	// Create and start server
	g.server = proxy.NewServer(cfg)

	// Try to start the server first to check for immediate errors (like port in use)
	localAddr := fmt.Sprintf("%s:%d", localHost, localPort)

	// Create a channel to communicate startup result
	startupResult := make(chan error, 1)

	go func() {
		err := g.server.Start(localAddr)
		startupResult <- err
	}()

	// Wait a short time for startup to complete or fail
	go func() {
		select {
		case err := <-startupResult:
			if err != nil {
				g.serverMutex.Lock()
				g.server = nil
				g.serverMutex.Unlock()

				// Update button state back to "Start" in UI thread
				g.startButton.SetText("Start")
				g.startButton.Importance = widget.SuccessImportance
				g.startButton.Refresh()

				// Re-enable form fields since server failed to start
				g.setFormFieldsEnabled(true)

				// Show detailed error message
				errorMsg := fmt.Sprintf("Failed to start SOCKS5 proxy server:\n\n%v\n\nPlease ensure port %d is not already in use.", err, localPort)
				dialog.ShowError(fmt.Errorf("%s", errorMsg), g.window)
			}
		case <-time.After(100 * time.Millisecond):
			// Server started successfully (no immediate error)
			// No popup - silent success
		}
	}()

	// Update button immediately
	g.startButton.SetText("Stop")
	g.startButton.Importance = widget.DangerImportance

	// Disable save button and form fields while server is running
	if g.saveButton != nil {
		g.saveButton.Disable()
	}
	g.setFormFieldsEnabled(false)
}

func (g *GUI) stopServer() {
	// Disable button to prevent double-tap and change text immediately
	g.startButton.Disable()
	// g.startButton.SetText("Stopping...")

	// Stop server in background since it can block
	go func() {
		if g.server != nil {
			g.server.Stop()
			g.serverMutex.Lock()
			g.server = nil
			g.serverMutex.Unlock()

			fyne.DoAndWait(func() {
				g.startButton.SetText("Start")
				g.startButton.Importance = widget.SuccessImportance
				g.startButton.Enable()
				// Re-enable form fields when server stops
				g.setFormFieldsEnabled(true)
			})
		}
	}()

	// Just change the state - Fyne should handle the refresh
	// time.AfterFunc(1*time.Second, func() {
	// 	g.serverMutex.Lock()
	// 	defer g.serverMutex.Unlock()
	// 	if g.server == nil {
	// 		g.startButton.SetText("Start")
	// 		g.startButton.Importance = widget.SuccessImportance
	// 		g.startButton.Enable()
	// 	}
	// })
}
