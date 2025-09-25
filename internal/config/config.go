package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// WatchDir represents a directory to watch for changes
type WatchDir struct {
	Path      string   `koanf:"path" yaml:"path"`
	Recursive bool     `koanf:"recursive" yaml:"recursive"`
	Exclude   []string `koanf:"exclude" yaml:"exclude"`
	Include   []string `koanf:"include" yaml:"include"`
	FileMode  string   `koanf:"file_mode" yaml:"file_mode"`
	DirMode   string   `koanf:"dir_mode" yaml:"dir_mode"`
}

// Config represents the application configuration
type Config struct {
	LogLevel     string     `koanf:"log_level" yaml:"log_level"`
	PollInterval int        `koanf:"poll_interval" yaml:"poll_interval"`
	WatchDirs    []WatchDir `koanf:"watch_dirs" yaml:"watch_dirs"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LogLevel:     "info",
		PollInterval: 30,
		WatchDirs:    []WatchDir{},
	}
}

// Load loads configuration from a YAML file
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Load default configuration
	cfg := DefaultConfig()

	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("config file not found: %s", configPath)
		}
		return cfg, fmt.Errorf("error accessing config file: %w", err)
	}

	// Load configuration file
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return cfg, fmt.Errorf("error loading config file: %w", err)
	}

	// Unmarshal into struct
	if err := k.Unmarshal("", cfg); err != nil {
		return cfg, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// validate performs basic configuration validation
func (c *Config) validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be greater than 0")
	}

	for i, watchDir := range c.WatchDirs {
		if watchDir.Path == "" {
			return fmt.Errorf("watch_dirs[%d].path is required", i)
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(watchDir.Path)
		if err != nil {
			return fmt.Errorf("invalid path in watch_dirs[%d]: %w", i, err)
		}
		c.WatchDirs[i].Path = absPath

		// Set default file and directory modes if not specified
		if watchDir.FileMode == "" {
			c.WatchDirs[i].FileMode = "0644"
		}
		if watchDir.DirMode == "" {
			c.WatchDirs[i].DirMode = "0755"
		}
	}

	return nil
}
