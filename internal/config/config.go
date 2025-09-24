package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Default configuration values
const (
	DefaultPort         = 8080
	DefaultLogLevel     = "info"
	DefaultPollInterval = 30 * time.Second
	DefaultTimezone     = "UTC"
)

// YAMLFolder represents a folder configuration in the YAML file
type YAMLFolder struct {
	Path      string `yaml:"path"`
	UID       int    `yaml:"uid"`
	GID       int    `yaml:"gid"`
	Mode      int    `yaml:"mode"`      // Default mode for the folder itself
	FileMode  int    `yaml:"fileMode,omitempty"`  // Mode for files (optional, falls back to Mode)
	DirMode   int    `yaml:"dirMode,omitempty"`   // Mode for subdirectories (optional, falls back to Mode)
}

// YAMLConfig represents the structure of the YAML configuration file
type YAMLConfig struct {
	Port         int          `yaml:"port,omitempty"`
	LogLevel     string       `yaml:"logLevel,omitempty"`
	PollInterval string       `yaml:"pollInterval,omitempty"`
	Timezone     string       `yaml:"timezone,omitempty"`
	Folders      []YAMLFolder `yaml:"folders,omitempty"`
}

// Folder represents a folder configuration with validated values
type Folder struct {
	Path     string
	UID      int
	GID      int
	Mode     os.FileMode
	FileMode os.FileMode // Mode for files within the folder
	DirMode  os.FileMode // Mode for subdirectories within the folder
}

// Validate checks that the folder configuration is valid
func (f Folder) Validate() error {
	// Validate path is not empty
	if f.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Validate path is absolute
	if !filepath.IsAbs(f.Path) {
		return fmt.Errorf("path must be absolute, got %q", f.Path)
	}

	// Validate UID/GID are reasonable (0-65535 is a reasonable range)
	if f.UID < 0 || f.UID > 65535 {
		return fmt.Errorf("uid must be between 0 and 65535, got %d", f.UID)
	}

	if f.GID < 0 || f.GID > 65535 {
		return fmt.Errorf("gid must be between 0 and 65535, got %d", f.GID)
	}

	// Validate mode is within reasonable range (0-0777)
	mode := f.Mode & os.ModePerm
	if mode > 0777 {
		return fmt.Errorf("mode must be between 0 and 0777, got %o", mode)
	}

	// Validate FileMode if specified
	if f.FileMode != 0 {
		fileMode := f.FileMode & os.ModePerm
		if fileMode > 0777 {
			return fmt.Errorf("fileMode must be between 0 and 0777, got %o", fileMode)
		}
	}

	// Validate DirMode if specified
	if f.DirMode != 0 {
		dirMode := f.DirMode & os.ModePerm
		if dirMode > 0777 {
			return fmt.Errorf("dirMode must be between 0 and 0777, got %o", dirMode)
		}
	}

	return nil
}

// Config represents the application configuration
type Config struct {
	Port         int
	LogLevel     string
	PollInterval time.Duration
	Timezone     *time.Location
	Folders      []Folder
}

// String returns a string representation of the config for logging
func (c Config) String() string {
	return fmt.Sprintf("Config{Port:%d, LogLevel:%s, PollInterval:%s, Timezone:%s, Folders:%d}",
		c.Port, c.LogLevel, c.PollInterval, c.Timezone, len(c.Folders))
}

// Load loads the configuration from environment variables or a YAML file
// Environment variables will override values from the config file
func Load() (*Config, error) {
	cfg := &Config{}

	// Try to load from config file first
	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		if err := cfg.loadFromFile(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	} else {
		// Load from environment variables
		if err := cfg.loadFromEnv(); err != nil {
			return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}
	}

	// Apply defaults for any unset values
	cfg.applyDefaults()

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// applyDefaults sets default values for any unset configuration options
func (c *Config) applyDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if c.PollInterval == 0 {
		c.PollInterval = DefaultPollInterval
	}
	if c.Timezone == nil {
		loc, err := time.LoadLocation(DefaultTimezone)
		if err != nil {
			// Fallback to UTC if timezone is invalid
			loc = time.UTC
		}
		c.Timezone = loc
	}
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level %q, must be one of: debug, info, warn, error", c.LogLevel)
	}

	// Validate that we have folders configured
	if len(c.Folders) == 0 {
		return fmt.Errorf("no folders configured")
	}

	// Validate each folder
	for i, folder := range c.Folders {
		if err := folder.Validate(); err != nil {
			return fmt.Errorf("folder %d (%s): %w", i, folder.Path, err)
		}
	}

	return nil
}

// loadFromFile loads configuration from a YAML file
// Environment variables will override values loaded from the file
func (c *Config) loadFromFile(filename string) error {
	// Read the YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading config file %q: %w", filename, err)
	}

	// Parse the YAML data
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("parsing config file %q: %w", filename, err)
	}

	// Apply YAML config values (will be potentially overridden by env vars)
	if yamlConfig.Port != 0 {
		c.Port = yamlConfig.Port
	}

	if yamlConfig.LogLevel != "" {
		c.LogLevel = yamlConfig.LogLevel
	}

	if yamlConfig.PollInterval != "" {
		duration, err := time.ParseDuration(yamlConfig.PollInterval)
		if err != nil {
			return fmt.Errorf("invalid pollInterval in config file: %w", err)
	}
		c.PollInterval = duration
	}

	if yamlConfig.Timezone != "" {
		loc, err := time.LoadLocation(yamlConfig.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone in config file: %w", err)
	}
		c.Timezone = loc
	}

	// Parse folders using common function
	c.Folders, err = parseFolders(yamlConfig.Folders)
	if err != nil {
		return fmt.Errorf("parsing folders from config file: %w", err)
	}

	// Load basic config from environment to allow env vars to override file config
	c.loadBasicConfigFromEnv()

	return nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() error {
	// Load basic config values from environment
	c.loadBasicConfigFromEnv()

	// Load folders from environment variable as YAML format only
	foldersEnv := os.Getenv("FOLDERS")
	if foldersEnv == "" {
		return fmt.Errorf("FOLDERS environment variable not set")
	}

	// Parse as YAML format only (removing legacy support)
	var yamlFolders []YAMLFolder
	if err := yaml.Unmarshal([]byte(foldersEnv), &yamlFolders); err != nil {
		return fmt.Errorf("failed to parse FOLDERS environment variable as YAML: %w", err)
	}

	var err error
	c.Folders, err = parseFolders(yamlFolders)
	if err != nil {
	return fmt.Errorf("parsing folders from environment: %w", err)
	}

	return nil
}

// parseFolders converts YAMLFolder structs to Folder structs with proper mode fallbacks
func parseFolders(yamlFolders []YAMLFolder) ([]Folder, error) {
	folders := make([]Folder, 0, len(yamlFolders))
	for _, yamlFolder := range yamlFolders {
		// If FileMode or DirMode are not set, use Mode as fallback
	fileMode := os.FileMode(yamlFolder.FileMode)
		if fileMode == 0 {
			fileMode = os.FileMode(yamlFolder.Mode)
	}
		
		dirMode := os.FileMode(yamlFolder.DirMode)
		if dirMode == 0 {
			dirMode = os.FileMode(yamlFolder.Mode)
		}
		
		folder := Folder{
			Path:     yamlFolder.Path,
			UID:      yamlFolder.UID,
			GID:      yamlFolder.GID,
			Mode:     os.FileMode(yamlFolder.Mode),
			FileMode: fileMode,
			DirMode:  dirMode,
		}
		folders = append(folders, folder)
	}
	return folders, nil
}

// loadBasicConfigFromEnv loads basic configuration values from environment variables
func (c *Config) loadBasicConfigFromEnv() {
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			c.Port = port
		}
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}

	if pollIntervalStr := os.Getenv("POLL_INTERVAL"); pollIntervalStr != "" {
		if duration, err := time.ParseDuration(pollIntervalStr); err == nil {
			c.PollInterval = duration
		}
	}

	if tz := os.Getenv("TZ"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			c.Timezone = loc
		}
	}
}
