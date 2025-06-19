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
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	app            fyne.App
	window         fyne.Window
	config         *config.Config
	encpass        string
	isNewUser      bool
	server         *proxy.Server
	serverMutex    sync.Mutex
	startButton    *widget.Button
	saveButton     *widget.Button
	localHostEntry *widget.Entry
	localPortEntry *widget.Entry
	usernameEntry  *widget.Entry
	passwordEntry  *widget.Entry
	hostEntry      *widget.Entry
	portEntry      *widget.Entry
	logFileEntry   *widget.Entry
}

func NewGUI() *GUI {
	return &GUI{
		app: app.New(),
	}
}

func (g *GUI) Run() {
	g.app.Settings().SetTheme(&myTheme{})
	g.app.SetIcon(resourceIconPng)
	g.window = g.app.NewWindow("Go SOCKS5 Chain Configuration")
	g.window.SetIcon(resourceIconPng)
	g.window.Resize(fyne.NewSize(600, 500))
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
	if g.config != nil {
		originalUsername = g.config.Username
		originalPassword = g.config.Password
		originalHost = g.config.UpstreamHost
		originalPort = strconv.Itoa(g.config.UpstreamPort)
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
	g.localHostEntry.Text = "127.0.0.1"
	g.localHostEntry.PlaceHolder = "127.0.0.1"

	g.localPortEntry = widget.NewEntry()
	g.localPortEntry.Text = "1080"
	g.localPortEntry.PlaceHolder = "1080"

	g.logFileEntry = widget.NewEntry()
	g.logFileEntry.PlaceHolder = "/path/to/logfile (optional)"

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
				g.localHostEntry.Text != "127.0.0.1" ||
				g.localPortEntry.Text != "1080" ||
				g.logFileEntry.Text != ""
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

	// Create form layout with fixed-width labels and aligned entries
	formContent := container.NewVBox()

	// Define fixed width for labels to ensure alignment
	labelWidth := float32(150)

	// Create each form row with proper alignment
	fields := []struct {
		label  string
		widget fyne.CanvasObject
	}{
		{"Upstream Username:", g.usernameEntry},
		{"Upstream Password:", g.passwordEntry},
		{"Upstream Host:", g.hostEntry},
		{"Upstream Port:", g.portEntry},
		{"Local Host:", g.localHostEntry},
		{"Local Port:", g.localPortEntry},
		{"Log File:", g.logFileEntry},
	}

	for _, field := range fields {
		label := widget.NewLabel(field.label)
		// Create a container with fixed width for label
		labelContainer := container.NewWithoutLayout(label)
		label.Resize(fyne.NewSize(labelWidth, label.MinSize().Height))
		label.Move(fyne.NewPos(0, 0))

		row := container.NewBorder(nil, nil, labelContainer, nil, field.widget)
		formContent.Add(row)
	}

	// Create start/stop button
	g.startButton = widget.NewButton("Start", func() {
		g.toggleServer()
	})
	g.startButton.Importance = widget.SuccessImportance

	// Disable start button if no config exists
	if g.config == nil || g.config.UpstreamHost == "" {
		g.startButton.Disable()
	}

	// Add save and start/stop buttons
	formContent.Add(widget.NewSeparator())
	buttonContainer := container.NewHBox(
		g.saveButton,
		g.startButton,
	)
	formContent.Add(container.NewCenter(buttonContainer))

	// Create scrollable content
	scroll := container.NewScroll(formContent)

	var title string
	if g.isNewUser {
		title = "Configure SOCKS5 Proxy Settings"
	} else {
		title = "Edit SOCKS5 Proxy Settings"
	}

	content := container.NewBorder(
		container.NewPadded(widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		nil, nil, nil,
		scroll,
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
	if g.logFileEntry != nil {
		if enabled {
			g.logFileEntry.Enable()
		} else {
			g.logFileEntry.Disable()
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
