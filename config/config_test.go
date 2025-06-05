package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		password string
	}{
		{
			name:     "Basic string",
			data:     "Hello World",
			password: "testpassword",
		},
		{
			name:     "JSON data",
			data:     `{"username":"user","password":"pass"}`,
			password: "strongpassword123",
		},
		{
			name:     "Empty string",
			data:     "",
			password: "password",
		},
		{
			name:     "Unicode data",
			data:     "Hello 世界! 🌍",
			password: "unicode_password_测试",
		},
		{
			name:     "Long data",
			data:     string(make([]byte, 10000)), // Large data
			password: "longpassword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := encrypt([]byte(tt.data), tt.password)
			if err != nil {
				t.Fatalf("encrypt() error = %v", err)
			}

			// Decrypt
			decrypted, err := decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("decrypt() error = %v", err)
			}

			if string(decrypted) != tt.data {
				t.Errorf("decrypt() = %q, want %q", string(decrypted), tt.data)
			}
		})
	}
}

func TestEncryptDecryptWrongPassword(t *testing.T) {
	data := "secret data"
	password := "correctpassword"
	wrongPassword := "wrongpassword"

	encrypted, err := encrypt([]byte(data), password)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	_, err = decrypt(encrypted, wrongPassword)
	if err == nil {
		t.Error("decrypt() with wrong password should fail")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Invalid base64",
			data: []byte("invalid base64 data!"),
		},
		{
			name: "Short data",
			data: []byte("c2hvcnQ="), // "short" in base64, but too short for GCM
		},
		{
			name: "Empty data",
			data: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decrypt(tt.data, "password")
			if err == nil {
				t.Error("decrypt() with invalid data should fail")
			}
		})
	}
}

func TestConfigLoadOrCreate(t *testing.T) {
	// Create temporary directory for tests
	tempDir := t.TempDir()
	
	// Override getConfigPath for testing
	originalGetConfigPath := GetConfigPath()
	SetConfigPathForTesting(func() (string, error) {
		return tempDir, nil
	})
	defer func() {
		SetConfigPathForTesting(originalGetConfigPath)
	}()

	t.Run("Create new config", func(t *testing.T) {
		cfg, err := LoadOrCreate("testuser", "testpass", "encpass", "proxy.example.com", 1080)
		if err != nil {
			t.Fatalf("LoadOrCreate() error = %v", err)
		}

		if cfg.Username != "testuser" {
			t.Errorf("Username = %q, want %q", cfg.Username, "testuser")
		}
		if cfg.Password != "testpass" {
			t.Errorf("Password = %q, want %q", cfg.Password, "testpass")
		}
		if cfg.UpstreamHost != "proxy.example.com" {
			t.Errorf("UpstreamHost = %q, want %q", cfg.UpstreamHost, "proxy.example.com")
		}
		if cfg.UpstreamPort != 1080 {
			t.Errorf("UpstreamPort = %d, want %d", cfg.UpstreamPort, 1080)
		}

		// Verify files were created
		configPath := filepath.Join(tempDir, configFile)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		credsPath := filepath.Join(tempDir, credsFile)
		if _, err := os.Stat(credsPath); os.IsNotExist(err) {
			t.Error("Credentials file was not created")
		}
	})

	t.Run("Load existing config", func(t *testing.T) {
		// Load the config created in previous test
		cfg, err := LoadOrCreate("", "", "encpass", "", 0)
		if err != nil {
			t.Fatalf("LoadOrCreate() error = %v", err)
		}

		if cfg.Username != "testuser" {
			t.Errorf("Username = %q, want %q", cfg.Username, "testuser")
		}
		if cfg.Password != "testpass" {
			t.Errorf("Password = %q, want %q", cfg.Password, "testpass")
		}
		if cfg.UpstreamHost != "proxy.example.com" {
			t.Errorf("UpstreamHost = %q, want %q", cfg.UpstreamHost, "proxy.example.com")
		}
		if cfg.UpstreamPort != 1080 {
			t.Errorf("UpstreamPort = %d, want %d", cfg.UpstreamPort, 1080)
		}
	})

	t.Run("Override existing config", func(t *testing.T) {
		cfg, err := LoadOrCreate("newuser", "newpass", "encpass", "newproxy.example.com", 2080)
		if err != nil {
			t.Fatalf("LoadOrCreate() error = %v", err)
		}

		if cfg.Username != "newuser" {
			t.Errorf("Username = %q, want %q", cfg.Username, "newuser")
		}
		if cfg.Password != "newpass" {
			t.Errorf("Password = %q, want %q", cfg.Password, "newpass")
		}
		if cfg.UpstreamHost != "newproxy.example.com" {
			t.Errorf("UpstreamHost = %q, want %q", cfg.UpstreamHost, "newproxy.example.com")
		}
		if cfg.UpstreamPort != 2080 {
			t.Errorf("UpstreamPort = %d, want %d", cfg.UpstreamPort, 2080)
		}
	})
}

func TestConfigLoadOrCreateMissingEncryptionPassword(t *testing.T) {
	tempDir := t.TempDir()
	
	originalGetConfigPath := GetConfigPath()
	SetConfigPathForTesting(func() (string, error) {
		return tempDir, nil
	})
	defer func() {
		SetConfigPathForTesting(originalGetConfigPath)
	}()

	// First create a config with encrypted credentials
	_, err := LoadOrCreate("testuser", "testpass", "encpass", "proxy.example.com", 1080)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}

	// Now try to load without encryption password
	_, err = LoadOrCreate("", "", "", "", 0)
	if err != ErrEncryptionPasswordRequired {
		t.Errorf("Expected ErrEncryptionPasswordRequired, got %v", err)
	}
}

func TestConfigLoadOrCreateWrongEncryptionPassword(t *testing.T) {
	tempDir := t.TempDir()
	
	originalGetConfigPath := GetConfigPath()
	SetConfigPathForTesting(func() (string, error) {
		return tempDir, nil
	})
	defer func() {
		SetConfigPathForTesting(originalGetConfigPath)
	}()

	// First create a config with encrypted credentials
	_, err := LoadOrCreate("testuser", "testpass", "correctpass", "proxy.example.com", 1080)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}

	// Now try to load with wrong encryption password
	_, err = LoadOrCreate("", "", "wrongpass", "", 0)
	if err == nil {
		t.Error("Expected decryption error with wrong password")
	}
}

func TestConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	originalGetConfigPath := GetConfigPath()
	SetConfigPathForTesting(func() (string, error) {
		return tempDir, nil
	})
	defer func() {
		SetConfigPathForTesting(originalGetConfigPath)
	}()

	tests := []struct {
		name         string
		username     string
		password     string
		upstreamHost string
		upstreamPort int
		wantError    bool
	}{
		{
			name:         "Valid config",
			username:     "user",
			password:     "pass",
			upstreamHost: "proxy.example.com",
			upstreamPort: 1080,
			wantError:    false,
		},
		{
			name:         "Missing username",
			username:     "",
			password:     "pass",
			upstreamHost: "proxy.example.com",
			upstreamPort: 1080,
			wantError:    true,
		},
		{
			name:         "Missing password",
			username:     "user",
			password:     "",
			upstreamHost: "proxy.example.com",
			upstreamPort: 1080,
			wantError:    true,
		},
		{
			name:         "Missing host",
			username:     "user",
			password:     "pass",
			upstreamHost: "",
			upstreamPort: 1080,
			wantError:    true,
		},
		{
			name:         "Missing port",
			username:     "user",
			password:     "pass",
			upstreamHost: "proxy.example.com",
			upstreamPort: 0,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing config files
			os.RemoveAll(filepath.Join(tempDir, configFile))
			os.RemoveAll(filepath.Join(tempDir, credsFile))

			_, err := LoadOrCreate(tt.username, tt.password, "encpass", tt.upstreamHost, tt.upstreamPort)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadOrCreate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigPartialLoad(t *testing.T) {
	tempDir := t.TempDir()
	
	originalGetConfigPath := GetConfigPath()
	SetConfigPathForTesting(func() (string, error) {
		return tempDir, nil
	})
	defer func() {
		SetConfigPathForTesting(originalGetConfigPath)
	}()

	// Create partial config files manually
	configPath := filepath.Join(tempDir, configFile)
	hostConfig := struct {
		UpstreamHost string `json:"upstream_host"`
		UpstreamPort int    `json:"upstream_port"`
	}{
		UpstreamHost: "existing.proxy.com",
		UpstreamPort: 3080,
	}

	data, err := json.Marshal(hostConfig)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := os.MkdirAll(tempDir, 0700); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	// Load config with new credentials but existing host config
	cfg, err := LoadOrCreate("newuser", "newpass", "encpass", "", 0)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}

	// Should use existing host config
	if cfg.UpstreamHost != "existing.proxy.com" {
		t.Errorf("UpstreamHost = %q, want %q", cfg.UpstreamHost, "existing.proxy.com")
	}
	if cfg.UpstreamPort != 3080 {
		t.Errorf("UpstreamPort = %d, want %d", cfg.UpstreamPort, 3080)
	}
	// Should use new credentials
	if cfg.Username != "newuser" {
		t.Errorf("Username = %q, want %q", cfg.Username, "newuser")
	}
	if cfg.Password != "newpass" {
		t.Errorf("Password = %q, want %q", cfg.Password, "newpass")
	}
}