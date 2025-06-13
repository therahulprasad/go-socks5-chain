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
	g.window = g.app.NewWindow("Go SOCKS5 Chain Configuration")
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

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Password:", Widget: passwordEntry},
		},
		OnSubmit: func() {
			g.encpass = passwordEntry.Text
			if err := g.loadConfiguration(); err != nil {
				dialog.ShowError(err, g.window)
				return
			}
			g.showConfigurationEditor()
		},
		OnCancel: func() {
			g.app.Quit()
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Configuration files found. Please enter your password to unlock."),
		form,
	)

	g.window.SetContent(content)
}

func (g *GUI) showFirstTimeSetup() {
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Create access password"
	
	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.PlaceHolder = "Confirm password"

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Access Password:", Widget: passwordEntry},
			{Text: "Confirm Password:", Widget: confirmEntry},
		},
		OnSubmit: func() {
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
		},
		OnCancel: func() {
			g.app.Quit()
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Welcome to Go SOCKS5 Chain!"),
		widget.NewLabel("This appears to be your first time. Please set an access password."),
		form,
	)

	g.window.SetContent(content)
}

func (g *GUI) showConfigurationEditor() {
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

	// Create form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Upstream Username:", Widget: usernameEntry},
			{Text: "Upstream Password:", Widget: passwordEntry},
			{Text: "Upstream Host:", Widget: hostEntry},
			{Text: "Upstream Port:", Widget: portEntry},
			{Text: "Local Host:", Widget: localhostEntry},
			{Text: "Local Port:", Widget: localPortEntry},
			{Text: "Log File:", Widget: logFileEntry},
		},
		OnSubmit: func() {
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

			dialog.ShowInformation("Success", "Configuration saved successfully!", g.window)
		},
		OnCancel: func() {
			g.app.Quit()
		},
	}

	// Create scrollable content
	scroll := container.NewScroll(form)
	
	var title string
	if g.isNewUser {
		title = "Configure SOCKS5 Proxy Settings"
	} else {
		title = "Edit SOCKS5 Proxy Settings"
	}

	content := container.NewBorder(
		widget.NewLabel(title),
		nil, nil, nil,
		scroll,
	)

	g.window.SetContent(content)
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