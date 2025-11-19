// Package config handles application configuration using Viper.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Storage  StorageConfig  `mapstructure:"storage"`
	Defaults DefaultsConfig `mapstructure:"defaults"`
	Display  DisplayConfig  `mapstructure:"display"`
}

// StorageConfig holds storage-related configuration.
type StorageConfig struct {
	Path string `mapstructure:"path"`
}

// DefaultsConfig holds default values for new tasks.
type DefaultsConfig struct {
	Priority string `mapstructure:"priority"`
}

// DisplayConfig holds display-related configuration.
type DisplayConfig struct {
	Colors     bool   `mapstructure:"colors"`
	DateFormat string `mapstructure:"date_format"`
}

// Load reads configuration from file and environment.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configDir := filepath.Join(home, ".task-cli")
		v.AddConfigPath(configDir)
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Environment variables
	v.SetEnvPrefix("TASK")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file not found is OK, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Expand home directory in storage path
	if cfg.Storage.Path != "" && cfg.Storage.Path[0] == '~' {
		home, _ := os.UserHomeDir()
		cfg.Storage.Path = filepath.Join(home, cfg.Storage.Path[1:])
	}

	return &cfg, nil
}

// setDefaults configures default values.
func setDefaults(v *viper.Viper) {
	home, _ := os.UserHomeDir()

	v.SetDefault("storage.path", filepath.Join(home, ".task-cli", "tasks.json"))
	v.SetDefault("defaults.priority", "medium")
	v.SetDefault("display.colors", true)
	v.SetDefault("display.date_format", "2006-01-02")
}

// Save writes the current configuration to file.
func Save(cfg *Config, path string) error {
	v := viper.New()

	v.Set("storage.path", cfg.Storage.Path)
	v.Set("defaults.priority", cfg.Defaults.Priority)
	v.Set("display.colors", cfg.Display.Colors)
	v.Set("display.date_format", cfg.Display.DateFormat)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return v.WriteConfigAs(path)
}
