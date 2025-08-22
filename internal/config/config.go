package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Folder struct {
	Path string
	UID  int
	GID  int
	Mode os.FileMode
}

type Config struct {
	Port         int
	LogLevel     string
	PollInterval time.Duration
	Timezone     *time.Location
	Folders      []Folder
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:         getEnvInt("PORT", 8080),
		LogLevel:     getEnvString("LOG_LEVEL", "info"),
		PollInterval: getEnvDuration("POLL_INTERVAL", 30*time.Second),
		Timezone:     getEnvTimezone("TZ", time.UTC),
	}

	// Load folders from environment or config file
	if err := cfg.loadFolders(); err != nil {
		return nil, fmt.Errorf("loading folders: %w", err)
	}

	if len(cfg.Folders) == 0 {
		return nil, fmt.Errorf("no folders configured")
	}

	return cfg, nil
}

func (c *Config) loadFolders() error {
	// Try config file first
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		if err := c.loadFromFile(configFile); err == nil {
			return nil
		}
	}

	// Fall back to environment variable
	return c.loadFromEnv()
}

func (c *Config) loadFromFile(filename string) error {
	// For now, skip config file loading - can be added later with YAML support
	return fmt.Errorf("config file loading not implemented")
}

func (c *Config) loadFromEnv() error {
	foldersEnv := os.Getenv("FOLDERS")
	if foldersEnv == "" {
		return fmt.Errorf("FOLDERS environment variable not set")
	}

	parts := strings.Split(foldersEnv, ",")
	c.Folders = make([]Folder, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		folder, err := parseFolderConfig(part)
		if err != nil {
			return fmt.Errorf("invalid folder config %q: %w", part, err)
		}

		c.Folders = append(c.Folders, folder)
	}

	return nil
}

func parseFolderConfig(config string) (Folder, error) {
	parts := strings.Split(config, ":")
	if len(parts) != 4 {
		return Folder{}, fmt.Errorf("expected format: /path:uid:gid:mode")
	}

	uid, err := strconv.Atoi(parts[1])
	if err != nil {
		return Folder{}, fmt.Errorf("invalid uid: %w", err)
	}

	gid, err := strconv.Atoi(parts[2])
	if err != nil {
		return Folder{}, fmt.Errorf("invalid gid: %w", err)
	}

	mode, err := strconv.ParseUint(parts[3], 8, 32)
	if err != nil {
		return Folder{}, fmt.Errorf("invalid mode: %w", err)
	}

	return Folder{
		Path: parts[0],
		UID:  uid,
		GID:  gid,
		Mode: os.FileMode(mode),
	}, nil
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvTimezone(key string, defaultValue *time.Location) *time.Location {
	if value := os.Getenv(key); value != "" {
		if loc, err := time.LoadLocation(value); err == nil {
			return loc
		}
	}
	return defaultValue
}
