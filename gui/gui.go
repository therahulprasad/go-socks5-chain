//go:build gui
// +build gui

package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"go-socks5-chain/config"
)

type GUI struct {
	app       fyne.App
	window    fyne.Window
	config    *config.Config
	encpass   string
	isNewUser bool
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
	usernameEntry := widget.NewEntry()
	usernameEntry.PlaceHolder = "SOCKS5 Username"
	if g.config != nil {
		usernameEntry.Text = g.config.Username
	}

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "SOCKS5 Password"
	if g.config != nil {
		passwordEntry.Text = g.config.Password
	}

	hostEntry := widget.NewEntry()
	hostEntry.PlaceHolder = "proxy.example.com"
	if g.config != nil {
		hostEntry.Text = g.config.UpstreamHost
	}

	portEntry := widget.NewEntry()
	portEntry.PlaceHolder = "1080"
	if g.config != nil && g.config.UpstreamPort > 0 {
		portEntry.Text = strconv.Itoa(g.config.UpstreamPort)
	}

	localhostEntry := widget.NewEntry()
	localhostEntry.Text = "127.0.0.1"
	localhostEntry.PlaceHolder = "127.0.0.1"

	localPortEntry := widget.NewEntry()
	localPortEntry.Text = "1080"
	localPortEntry.PlaceHolder = "1080"

	logFileEntry := widget.NewEntry()
	logFileEntry.PlaceHolder = "/path/to/logfile (optional)"

	// Determine button text based on whether it's new or existing config
	buttonText := "Save"
	if g.isNewUser == false {
		buttonText = "Update"
	}

	// Create save/update function
	saveFunc := func() {
		// Validate input
		if usernameEntry.Text == "" || passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Username and password are required"), g.window)
			return
		}
		if hostEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Upstream host is required"), g.window)
			return
		}
		
		port, err := strconv.Atoi(portEntry.Text)
		if err != nil || port <= 0 || port > 65535 {
			dialog.ShowError(fmt.Errorf("Invalid upstream port"), g.window)
			return
		}

		localPort, err := strconv.Atoi(localPortEntry.Text)
		if err != nil || localPort <= 0 || localPort > 65535 {
			dialog.ShowError(fmt.Errorf("Invalid local port"), g.window)
			return
		}

		// Update configuration
		g.config = &config.Config{
			Username:     usernameEntry.Text,
			Password:     passwordEntry.Text,
			UpstreamHost: hostEntry.Text,
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
		originalUsername = usernameEntry.Text
		originalPassword = passwordEntry.Text
		originalHost = hostEntry.Text
		originalPort = portEntry.Text
	}

	// Create save button (disabled by default)
	saveButton := widget.NewButton(buttonText, saveFunc)
	saveButton.Importance = widget.HighImportance
	saveButton.Disable()

	// Function to check for changes
	checkChanges := func() {
		hasChanges := false
		
		// Check if any field has changed from original
		if g.isNewUser {
			// For new users, check if required fields have values
			hasChanges = usernameEntry.Text != "" && passwordEntry.Text != "" && 
				hostEntry.Text != "" && portEntry.Text != ""
		} else {
			// For existing users, check if any field differs from original
			hasChanges = usernameEntry.Text != originalUsername ||
				passwordEntry.Text != originalPassword ||
				hostEntry.Text != originalHost ||
				portEntry.Text != originalPort ||
				localhostEntry.Text != "127.0.0.1" ||
				localPortEntry.Text != "1080" ||
				logFileEntry.Text != ""
		}

		if hasChanges {
			saveButton.Enable()
		} else {
			saveButton.Disable()
		}
	}

	// Add change listeners to all fields
	usernameEntry.OnChanged = func(string) { checkChanges() }
	passwordEntry.OnChanged = func(string) { checkChanges() }
	hostEntry.OnChanged = func(string) { checkChanges() }
	portEntry.OnChanged = func(string) { checkChanges() }
	localhostEntry.OnChanged = func(string) { checkChanges() }
	localPortEntry.OnChanged = func(string) { checkChanges() }
	logFileEntry.OnChanged = func(string) { checkChanges() }

	// Create form layout with fixed-width labels and aligned entries
	formContent := container.NewVBox()
	
	// Define fixed width for labels to ensure alignment
	labelWidth := float32(150)
	
	// Create each form row with proper alignment
	fields := []struct {
		label  string
		widget fyne.CanvasObject
	}{
		{"Upstream Username:", usernameEntry},
		{"Upstream Password:", passwordEntry},
		{"Upstream Host:", hostEntry},
		{"Upstream Port:", portEntry},
		{"Local Host:", localhostEntry},
		{"Local Port:", localPortEntry},
		{"Log File:", logFileEntry},
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

	// Add save button (no cancel button)
	formContent.Add(widget.NewSeparator())
	formContent.Add(container.NewCenter(saveButton))

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
	g.window.Canvas().Focus(usernameEntry)
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