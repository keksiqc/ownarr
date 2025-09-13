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
	Path       string `yaml:"path"`
	UID        int    `yaml:"uid"`
	GID        int    `yaml:"gid"`
	Mode       int    `yaml:"mode,omitempty"`       // Legacy: applies to both files and folders
	FolderMode int    `yaml:"folderMode,omitempty"` // New: specific folder permissions
	FileMode   int    `yaml:"fileMode,omitempty"`   // New: specific file permissions
}

// TrashGuidesConfig represents the Trash Guides folder structure configuration
type TrashGuidesConfig struct {
	Enabled         bool     `yaml:"enabled,omitempty"`
	Type            string   `yaml:"type,omitempty"`            // "usenet" or "torrent"
	RootPath        string   `yaml:"rootPath,omitempty"`        // Base path for structure
	MediaTypes      []string `yaml:"mediaTypes,omitempty"`      // ["movies", "tv", "music", "books"]
	CreateStructure bool     `yaml:"createStructure,omitempty"` // Whether to create directories
	UID             int      `yaml:"uid,omitempty"`
	GID             int      `yaml:"gid,omitempty"`
	FolderMode      int      `yaml:"folderMode,omitempty"`
	FileMode        int      `yaml:"fileMode,omitempty"`
}

// YAMLConfig represents the structure of the YAML configuration file
type YAMLConfig struct {
	Port         int                `yaml:"port,omitempty"`
	LogLevel     string             `yaml:"logLevel,omitempty"`
	PollInterval string             `yaml:"pollInterval,omitempty"`
	Timezone     string             `yaml:"timezone,omitempty"`
	Folders      []YAMLFolder       `yaml:"folders,omitempty"`
	TrashGuides  *TrashGuidesConfig `yaml:"trashGuides,omitempty"`
}

// Folder represents a folder configuration with validated values
type Folder struct {
	Path       string
	UID        int
	GID        int
	Mode       os.FileMode // Legacy: used when FolderMode/FileMode not specified
	FolderMode os.FileMode // Specific permissions for folders
	FileMode   os.FileMode // Specific permissions for files
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

	// Validate modes are within reasonable range (0-0777)
	if f.Mode != 0 {
		mode := f.Mode & os.ModePerm
		if mode > 0777 {
			return fmt.Errorf("mode must be between 0 and 0777, got %o", mode)
		}
	}

	if f.FolderMode != 0 {
		mode := f.FolderMode & os.ModePerm
		if mode > 0777 {
			return fmt.Errorf("folderMode must be between 0 and 0777, got %o", mode)
		}
	}

	if f.FileMode != 0 {
		mode := f.FileMode & os.ModePerm
		if mode > 0777 {
			return fmt.Errorf("fileMode must be between 0 and 0777, got %o", mode)
		}
	}

	// At least one mode must be specified
	if f.Mode == 0 && f.FolderMode == 0 && f.FileMode == 0 {
		return fmt.Errorf("at least one of mode, folderMode, or fileMode must be specified")
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
	TrashGuides  *TrashGuidesConfig
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

	// Parse TrashGuides configuration
	if yamlConfig.TrashGuides != nil {
		c.TrashGuides = yamlConfig.TrashGuides

		// Generate Trash Guides folder structure if enabled
		if c.TrashGuides.Enabled && c.TrashGuides.CreateStructure {
			trashFolders, err := c.generateTrashGuidesStructure()
			if err != nil {
				return fmt.Errorf("failed to generate Trash Guides structure: %w", err)
			}
			c.Folders = append(c.Folders, trashFolders...)
		}
	}

	// Parse folders
	c.Folders = make([]Folder, 0, len(yamlConfig.Folders))
	for _, yamlFolder := range yamlConfig.Folders {
		folder := Folder{
			Path: yamlFolder.Path,
			UID:  yamlFolder.UID,
			GID:  yamlFolder.GID,
		}

		// Handle backward compatibility with Mode field
		if yamlFolder.Mode != 0 {
			folder.Mode = os.FileMode(yamlFolder.Mode)
		}

		// Use specific folder/file modes if provided
		if yamlFolder.FolderMode != 0 {
			folder.FolderMode = os.FileMode(yamlFolder.FolderMode)
		}
		if yamlFolder.FileMode != 0 {
			folder.FileMode = os.FileMode(yamlFolder.FileMode)
		}

		// If only legacy Mode is specified, use it for both files and folders
		if folder.Mode != 0 && folder.FolderMode == 0 && folder.FileMode == 0 {
			folder.FolderMode = folder.Mode
			folder.FileMode = folder.Mode
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

// generateTrashGuidesStructure generates the recommended Trash Guides folder structure
func (c *Config) generateTrashGuidesStructure() ([]Folder, error) {
	if c.TrashGuides == nil || !c.TrashGuides.Enabled {
		return nil, nil
	}

	// Validate TrashGuides configuration
	if c.TrashGuides.RootPath == "" {
		return nil, fmt.Errorf("trashGuides.rootPath is required")
	}

	if c.TrashGuides.Type != "usenet" && c.TrashGuides.Type != "torrent" {
		return nil, fmt.Errorf("trashGuides.type must be 'usenet' or 'torrent', got %q", c.TrashGuides.Type)
	}

	// Default media types
	mediaTypes := c.TrashGuides.MediaTypes
	if len(mediaTypes) == 0 {
		mediaTypes = []string{"movies", "tv", "music", "books"}
	}

	// Default permissions from TrashGuides config or use defaults
	folderMode := os.FileMode(0755)
	fileMode := os.FileMode(0644)
	uid := 1000
	gid := 1000

	if c.TrashGuides.FolderMode != 0 {
		folderMode = os.FileMode(c.TrashGuides.FolderMode)
	}
	if c.TrashGuides.FileMode != 0 {
		fileMode = os.FileMode(c.TrashGuides.FileMode)
	}
	if c.TrashGuides.UID != 0 {
		uid = c.TrashGuides.UID
	}
	if c.TrashGuides.GID != 0 {
		gid = c.TrashGuides.GID
	}

	var folders []Folder

	// Create base directories
	folders = append(folders, Folder{
		Path:       filepath.Join(c.TrashGuides.RootPath, "media"),
		UID:        uid,
		GID:        gid,
		FolderMode: folderMode,
		FileMode:   fileMode,
	})

	folders = append(folders, Folder{
		Path:       filepath.Join(c.TrashGuides.RootPath, "torrents"),
		UID:        uid,
		GID:        gid,
		FolderMode: folderMode,
		FileMode:   fileMode,
	})

	// For media folders
	for _, mediaType := range mediaTypes {
		folders = append(folders, Folder{
			Path:       filepath.Join(c.TrashGuides.RootPath, "media", mediaType),
			UID:        uid,
			GID:        gid,
			FolderMode: folderMode,
			FileMode:   fileMode,
		})

		folders = append(folders, Folder{
			Path:       filepath.Join(c.TrashGuides.RootPath, "torrents", mediaType),
			UID:        uid,
			GID:        gid,
			FolderMode: folderMode,
			FileMode:   fileMode,
		})
	}

	// Add usenet-specific directories if type is usenet
	if c.TrashGuides.Type == "usenet" {
		folders = append(folders, Folder{
			Path:       filepath.Join(c.TrashGuides.RootPath, "usenet"),
			UID:        uid,
			GID:        gid,
			FolderMode: folderMode,
			FileMode:   fileMode,
		})

		folders = append(folders, Folder{
			Path:       filepath.Join(c.TrashGuides.RootPath, "usenet", "complete"),
			UID:        uid,
			GID:        gid,
			FolderMode: folderMode,
			FileMode:   fileMode,
		})

		folders = append(folders, Folder{
			Path:       filepath.Join(c.TrashGuides.RootPath, "usenet", "incomplete"),
			UID:        uid,
			GID:        gid,
			FolderMode: folderMode,
			FileMode:   fileMode,
		})

		for _, mediaType := range mediaTypes {
			folders = append(folders, Folder{
				Path:       filepath.Join(c.TrashGuides.RootPath, "usenet", "complete", mediaType),
				UID:        uid,
				GID:        gid,
				FolderMode: folderMode,
				FileMode:   fileMode,
			})
		}
	}

	// Create directories if they don't exist
	for _, folder := range folders {
		if err := os.MkdirAll(folder.Path, folderMode); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", folder.Path, err)
		}
	}

	return folders, nil
}
