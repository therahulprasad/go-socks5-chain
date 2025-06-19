package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrEncryptionPasswordRequired is returned when credentials file exists but no encryption password is provided
var ErrEncryptionPasswordRequired = errors.New("encryption password required to decrypt existing credentials")

const (
	configDir  = ".go-socks5-chain"
	configFile = "upstream_config"
	credsFile  = "upstream_creds.enc"
)

type Config struct {
	Username     string
	Password     string
	UpstreamHost string
	UpstreamPort int
	LocalHost    string
	LocalPort    int
	LogFile      string
}

// getConfigPath is a variable so it can be overridden in tests
var getConfigPath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDir), nil
}

// GetConfigPath returns the current config path function (for testing)
func GetConfigPath() func() (string, error) {
	return getConfigPath
}

// SetConfigPathForTesting allows overriding the config path for tests
func SetConfigPathForTesting(fn func() (string, error)) {
	getConfigPath = fn
}

func LoadOrCreate(username, password, encpass, upstreamHost string, upstreamPort int) (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return nil, err
	}

	cfg := &Config{}

	// Try to load existing config
	configFilePath := filepath.Join(configPath, configFile)
	credsFilePath := filepath.Join(configPath, credsFile)

	// If credentials file exists, handle decryption
	if _, err := os.Stat(credsFilePath); err == nil {
		encData, err := os.ReadFile(credsFilePath)
		if err != nil {
			return nil, err
		}

		// If encpass is not provided but credentials exist, we need to ask for it
		if encpass == "" {
			return nil, ErrEncryptionPasswordRequired
		}

		data, err := decrypt(encData, encpass)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials: %v", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Load host/port config if it exists
	if _, err := os.Stat(configFilePath); err == nil {
		data, err := os.ReadFile(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
		var hostConfig struct {
			UpstreamHost string `json:"upstream_host"`
			UpstreamPort int    `json:"upstream_port"`
		}
		if err := json.Unmarshal(data, &hostConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
		if cfg.UpstreamHost == "" {
			cfg.UpstreamHost = hostConfig.UpstreamHost
		}
		if cfg.UpstreamPort == 0 {
			cfg.UpstreamPort = hostConfig.UpstreamPort
		}
	}

	// Update config with new values if provided
	if upstreamHost != "" {
		cfg.UpstreamHost = upstreamHost
	}
	if upstreamPort != 0 {
		cfg.UpstreamPort = upstreamPort
	}
	if username != "" {
		cfg.Username = username
	}
	if password != "" {
		cfg.Password = password
	}

	// Validate required fields
	if cfg.UpstreamHost == "" || cfg.UpstreamPort == 0 {
		return nil, fmt.Errorf("upstream host and port are required")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Save configs
	hostConfig := struct {
		UpstreamHost string `json:"upstream_host"`
		UpstreamPort int    `json:"upstream_port"`
	}{
		UpstreamHost: cfg.UpstreamHost,
		UpstreamPort: cfg.UpstreamPort,
	}

	data, err := json.Marshal(hostConfig)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configFilePath, data, 0600); err != nil {
		return nil, err
	}

	if encpass != "" {
		credsData, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		encrypted, err := encrypt(credsData, encpass)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(credsFilePath, encrypted, 0600); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func encrypt(data []byte, password string) ([]byte, error) {
	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func decrypt(encData []byte, password string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(encData))
	if err != nil {
		return nil, err
	}

	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ConfigExists checks if configuration files exist
func ConfigExists() bool {
	configPath, err := getConfigPath()
	if err != nil {
		return false
	}
	credsFilePath := filepath.Join(configPath, credsFile)
	_, err = os.Stat(credsFilePath)
	return err == nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	return os.MkdirAll(configPath, 0700)
}

// SaveConfig saves the configuration with encryption
func SaveConfig(cfg *Config, encpass string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configFilePath := filepath.Join(configPath, configFile)
	credsFilePath := filepath.Join(configPath, credsFile)

	// Save host/port config
	hostConfig := struct {
		UpstreamHost string `json:"upstream_host"`
		UpstreamPort int    `json:"upstream_port"`
	}{
		UpstreamHost: cfg.UpstreamHost,
		UpstreamPort: cfg.UpstreamPort,
	}

	data, err := json.Marshal(hostConfig)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFilePath, data, 0600); err != nil {
		return err
	}

	// Save encrypted credentials
	if encpass != "" {
		credsData, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		encrypted, err := encrypt(credsData, encpass)
		if err != nil {
			return err
		}
		if err := os.WriteFile(credsFilePath, encrypted, 0600); err != nil {
			return err
		}
	}

	return nil
}
