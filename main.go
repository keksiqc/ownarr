package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FolderConfig holds the configuration for a watched folder.
type FolderConfig struct {
	Path string
	UID  int
	GID  int
	Mode os.FileMode
}

// Config holds the application configuration
type Config struct {
	Folders      []FolderConfig
	LogLevel     slog.Leveler
	Timezone     *time.Location
	PollInterval time.Duration
}

// parseConfig parses a single folder config string in the format: /path:uid:gid:mode
func parseConfig(configStr string) (FolderConfig, error) {
	parts := strings.Split(configStr, ":")
	if len(parts) != 4 {
		return FolderConfig{}, fmt.Errorf("invalid config format: %s", configStr)
	}

	uid, err := strconv.Atoi(parts[1])
	if err != nil {
		return FolderConfig{}, fmt.Errorf("invalid uid: %w", err)
	}
	gid, err := strconv.Atoi(parts[2])
	if err != nil {
		return FolderConfig{}, fmt.Errorf("invalid gid: %w", err)
	}
	modeInt, err := strconv.ParseInt(parts[3], 8, 32)
	if err != nil {
		return FolderConfig{}, fmt.Errorf("invalid mode: %w", err)
	}

	return FolderConfig{
		Path: parts[0],
		UID:  uid,
		GID:  gid,
		Mode: os.FileMode(modeInt),
	}, nil
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	cfg := &Config{
		PollInterval: 30 * time.Second,
	}

	// Log level
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		cfg.LogLevel = slog.LevelDebug
	} else {
		cfg.LogLevel = slog.LevelInfo
	}

	// Timezone
	if tz := os.Getenv("TZ"); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			slog.Warn("invalid TZ, falling back to UTC", "tz", tz, "error", err)
			cfg.Timezone = time.UTC
		} else {
			cfg.Timezone = loc
		}
	} else {
		cfg.Timezone = time.UTC
	}

	// Poll interval
	if pollInterval := os.Getenv("POLL_INTERVAL"); pollInterval != "" {
		if interval, err := time.ParseDuration(pollInterval); err == nil {
			cfg.PollInterval = interval
		}
	}

	// Watch folders
	env := os.Getenv("WATCH_FOLDERS")
	if env == "" {
		return nil, fmt.Errorf("WATCH_FOLDERS environment variable not set")
	}

	configStrs := strings.Split(env, ",")
	cfg.Folders = make([]FolderConfig, 0, len(configStrs))

	for _, cfgStr := range configStrs {
		cfgStr = strings.TrimSpace(cfgStr)
		if cfgStr == "" {
			continue
		}

		folderCfg, err := parseConfig(cfgStr)
		if err != nil {
			return nil, fmt.Errorf("invalid config %q: %w", cfgStr, err)
		}

		cfg.Folders = append(cfg.Folders, folderCfg)
	}

	if len(cfg.Folders) == 0 {
		return nil, fmt.Errorf("no folder configs provided in WATCH_FOLDERS")
	}

	return cfg, nil
}

// enforceTree walks the entire folder tree and ensures ownership and permissions.
// Returns counts of fixed, skipped, and failed files.
func enforceTree(cfg FolderConfig) (fixed, skipped, failed int) {
	filepath.Walk(cfg.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("error accessing path", "path", path, "error", err)
			failed++
			return nil
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		changed := false

		// Ownership
		if int(stat.Uid) != cfg.UID || int(stat.Gid) != cfg.GID {
			if err := os.Chown(path, cfg.UID, cfg.GID); err != nil {
				slog.Warn("failed to chown", "path", path, "error", err)
				failed++
			} else {
				changed = true
			}
		}

		// Permissions
		if info.Mode().Perm() != cfg.Mode.Perm() {
			if err := os.Chmod(path, cfg.Mode); err != nil {
				slog.Warn("failed to chmod", "path", path, "error", err)
				failed++
			} else {
				changed = true
			}
		}

		if changed {
			fixed++
		} else {
			skipped++
		}
		return nil
	})
	return
}

// enforceFile ensures ownership and permissions for a single file or folder.
func enforceFile(cfg FolderConfig, path string) {
	// Just check if the file exists
	if _, err := os.Lstat(path); err != nil {
		slog.Warn("failed to stat file", "path", path, "error", err)
		return
	}

	// Change ownership
	if err := os.Chown(path, cfg.UID, cfg.GID); err != nil {
		slog.Warn("failed to chown", "path", path, "error", err)
	} else {
		slog.Debug("ownership set", "path", path, "uid", cfg.UID, "gid", cfg.GID)
	}

	// Change permissions
	if err := os.Chmod(path, cfg.Mode); err != nil {
		slog.Warn("failed to chmod", "path", path, "error", err)
	} else {
		slog.Debug("permissions set", "path", path, "mode", fmt.Sprintf("%o", cfg.Mode))
	}
}

// watchFolder sets up a watcher for a folder and enforces permissions on changes.
func watchFolder(cfg FolderConfig, pollInterval time.Duration) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("failed to create watcher", "error", err)
		return
	}
	defer watcher.Close()

	// Add initial folder recursively
	filepath.Walk(cfg.Path, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			if err := watcher.Add(path); err != nil {
				slog.Warn("failed to add path to watcher", "path", path, "error", err)
			}
		}
		return nil
	})

	slog.Info("started watching folder",
		"path", cfg.Path,
		"uid", cfg.UID,
		"gid", cfg.GID,
		"mode", fmt.Sprintf("%o", cfg.Mode),
	)

	// Initial enforcement
	fixed, skipped, failed := enforceTree(cfg)
	slog.Info("initial enforcement complete",
		"path", cfg.Path,
		"fixed", fixed,
		"skipped", skipped,
		"failed", failed,
	)

	// Polling fallback
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		slog.Info("started periodic polling",
			"path", cfg.Path,
			"interval", pollInterval.String())

		for range ticker.C {
			slog.Debug("running periodic enforcement", "path", cfg.Path)
			fixed, skipped, failed := enforceTree(cfg)

			// Only log if there were changes or errors
			if fixed > 0 || failed > 0 {
				slog.Info("periodic enforcement complete",
					"path", cfg.Path,
					"fixed", fixed,
					"skipped", skipped,
					"failed", failed,
				)
			} else {
				slog.Debug("periodic enforcement complete, no changes needed",
					"path", cfg.Path,
					"skipped", skipped,
				)
			}
		}
	}()

	// Event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Add new directories to watcher
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := watcher.Add(event.Name); err != nil {
						slog.Warn("failed to add new directory to watcher", "path", event.Name, "error", err)
					} else {
						slog.Info("added new directory to watcher", "path", event.Name)
					}
				}
			}

			// Ignore CHMOD spam
			if event.Op&fsnotify.Chmod != 0 {
				continue
			}

			slog.Debug("event detected", "path", event.Name, "op", event.Op.String())

			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				enforceFile(cfg, event.Name)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		}
	}
}

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Set up logging
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Set timezone
	time.Local = cfg.Timezone

	slog.Info("Ownarr starting up",
		"tz", time.Now().Location().String(),
		"debug", cfg.LogLevel == slog.LevelDebug,
	)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watching folders
	for _, folderCfg := range cfg.Folders {
		go watchFolder(folderCfg, cfg.PollInterval)
	}

	// Wait for termination signal
	<-sigChan
	slog.Info("Received termination signal, shutting down...")
}
