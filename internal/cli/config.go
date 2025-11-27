package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the CLI configuration
type Config struct {
	APIBaseURL         string `json:"apiBaseUrl"`
	AccessToken        string `json:"accessToken,omitempty"`
	RefreshToken       string `json:"refreshToken,omitempty"`
	UserEmail          string `json:"userEmail,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"` // For self-signed certs
}

const (
	configFileName = ".todolist-cli.json"
	configFileMode = 0600 // Read/write for owner only
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, configFileName), nil
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return &Config{
			APIBaseURL:         "https://localhost:8443/api/v1",
			InsecureSkipVerify: false,
		}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(configPath, data, configFileMode); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateTokens updates the access and refresh tokens in config
func UpdateTokens(accessToken, refreshToken string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken

	return SaveConfig(cfg)
}

// UpdateUserEmail updates the user email in config
func UpdateUserEmail(email string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cfg.UserEmail = email

	return SaveConfig(cfg)
}

// ClearTokens removes tokens from config (for logout)
func ClearTokens() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cfg.AccessToken = ""
	cfg.RefreshToken = ""

	return SaveConfig(cfg)
}
