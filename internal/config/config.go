package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	Path string `yaml:"path"`
	UID  int    `yaml:"uid"`
	GID  int    `yaml:"gid"`
	Mode int    `yaml:"mode"`
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
	Path string
	UID  int
	GID  int
	Mode os.FileMode
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
		if err == nil {
			c.Timezone = loc
		} else {
			c.Timezone = time.UTC
		}
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

	// Apply YAML config values
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

	// Parse folders
	c.Folders = make([]Folder, 0, len(yamlConfig.Folders))
	for _, yamlFolder := range yamlConfig.Folders {
		folder := Folder{
			Path: yamlFolder.Path,
			UID:  yamlFolder.UID,
			GID:  yamlFolder.GID,
			Mode: os.FileMode(yamlFolder.Mode),
		}
		c.Folders = append(c.Folders, folder)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() error {
	// Load basic config values from environment
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

	// Load folders from environment variable
	foldersEnv := os.Getenv("FOLDERS")
	if foldersEnv == "" {
		return fmt.Errorf("FOLDERS environment variable not set")
	}

	// Try to parse as new YAML format first, fallback to legacy format
	if strings.Contains(foldersEnv, "{") {
		// Try to parse as YAML
		var yamlFolders []YAMLFolder
		if err := yaml.Unmarshal([]byte(foldersEnv), &yamlFolders); err == nil {
			c.Folders = make([]Folder, 0, len(yamlFolders))
			for _, yamlFolder := range yamlFolders {
				folder := Folder{
					Path: yamlFolder.Path,
					UID:  yamlFolder.UID,
					GID:  yamlFolder.GID,
					Mode: os.FileMode(yamlFolder.Mode),
				}
				c.Folders = append(c.Folders, folder)
			}
			return nil
		}
	}

	// Fallback to legacy format
	parts := strings.Split(foldersEnv, ",")
	if len(parts) == 0 {
		return fmt.Errorf("no folders specified in FOLDERS environment variable")
	}

	c.Folders = make([]Folder, 0, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		folder, err := parseLegacyFolderConfig(part)
		if err != nil {
			return fmt.Errorf("invalid folder config at position %d (%q): %w", i, part, err)
		}

		c.Folders = append(c.Folders, folder)
	}

	if len(c.Folders) == 0 {
		return fmt.Errorf("no valid folders found in FOLDERS environment variable")
	}

	return nil
}

// parseLegacyFolderConfig parses the legacy folder configuration format
func parseLegacyFolderConfig(config string) (Folder, error) {
	parts := strings.Split(config, ":")
	if len(parts) != 4 {
		return Folder{}, fmt.Errorf("expected format: /path:uid:gid:mode, got %d parts", len(parts))
	}

	path := parts[0]
	if path == "" {
		return Folder{}, fmt.Errorf("path cannot be empty")
	}

	uid, err := strconv.Atoi(parts[1])
	if err != nil {
		return Folder{}, fmt.Errorf("invalid uid %q: %w", parts[1], err)
	}

	gid, err := strconv.Atoi(parts[2])
	if err != nil {
		return Folder{}, fmt.Errorf("invalid gid %q: %w", parts[2], err)
	}

	mode, err := strconv.ParseUint(parts[3], 8, 32)
	if err != nil {
		return Folder{}, fmt.Errorf("invalid mode %q: %w", parts[3], err)
	}

	folder := Folder{
		Path: path,
		UID:  uid,
		GID:  gid,
		Mode: os.FileMode(mode),
	}

	// Validate the folder configuration
	if err := folder.Validate(); err != nil {
		return Folder{}, fmt.Errorf("invalid folder configuration: %w", err)
	}

	return folder, nil
}
