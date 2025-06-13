package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go-socks5-chain/config"
	"go-socks5-chain/gui"
	"go-socks5-chain/proxy"

	"golang.org/x/term"
)

const Version = "0.1"

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add a newline after the password input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func setupInteractiveConfig() (string, string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter upstream username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", err
	}
	username = strings.TrimSpace(username)

	password, err := readPassword("Enter upstream password: ")
	if err != nil {
		return "", "", "", err
	}

	encpass, err := readPassword("Enter encryption password to protect credentials: ")
	if err != nil {
		return "", "", "", err
	}

	return username, password, encpass, nil
}

func main() {
	// Command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	username := flag.String("username", os.Getenv("UPSTREAM_USERNAME"), "Upstream SOCKS5 username")
	password := flag.String("password", os.Getenv("UPSTREAM_PASSWORD"), "Upstream SOCKS5 password")
	encpass := flag.String("encpass", os.Getenv("SOCKS5CHAIN_PASSWORD"), "Password to encrypt/decrypt stored credentials")
	upstreamHost := flag.String("upstream-host", "", "Upstream SOCKS5 proxy hostname")
	upstreamPort := flag.Int("upstream-port", 0, "Upstream SOCKS5 proxy port")
	localHost := flag.String("local-host", "127.0.0.1", "Local host to bind")
	localPort := flag.Int("local-port", 1080, "Local port to bind")
	logFile := flag.String("log-file", "", "Log file location")
	consoleLog := flag.Bool("console-log", false, "Enable console logging")
	configureMode := flag.Bool("configure", false, "Interactive mode to configure credentials")
	guiMode := flag.Bool("gui", false, "Launch graphical user interface for configuration")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("go-socks5-chain version %s\n", Version)
		os.Exit(0)
	}

	// Setup logging
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("Error opening log file:", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	if *consoleLog {
		log.SetOutput(os.Stdout)
	}

	// Handle GUI mode if requested
	if *guiMode {
		g := gui.NewGUI()
		g.Run()
		return
	}

	// Handle interactive configuration if requested
	if *configureMode {
		var err error
		*username, *password, *encpass, err = setupInteractiveConfig()
		if err != nil {
			log.Fatal("Error during configuration:", err)
		}
	}

	// Load or create configuration
	var cfg *config.Config
	cfg, err := config.LoadOrCreate(*username, *password, *encpass, *upstreamHost, *upstreamPort)
	if err == config.ErrEncryptionPasswordRequired {
		// Prompt for encryption password
		pwd, promptErr := readPassword("Enter encryption password to decrypt credentials: ")
		if promptErr != nil {
			log.Fatal("Failed to read encryption password:", promptErr)
		}
		// Try loading again with the provided password
		cfg, err = config.LoadOrCreate(*username, *password, pwd, *upstreamHost, *upstreamPort)
	}
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}

	// Create and start proxy server
	server := proxy.NewServer(cfg)
	localAddr := fmt.Sprintf("%s:%d", *localHost, *localPort)

	// Create error channel for server errors
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(localAddr); err != nil {
			errChan <- err
		}
	}()

	log.Printf("SOCKS5 proxy server listening on %s", localAddr)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error or shutdown signal
	select {
	case err := <-errChan:
		log.Fatal("Server error:", err)
	case sig := <-sigChan:
		log.Printf("Received signal %v, initiating shutdown...", sig)
		server.Stop()
		log.Println("Server shutdown complete")
	}
}
