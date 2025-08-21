package main

import (
	"fmt"
	"log/slog"
	"os"
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

//
// ─── CONFIG PARSING ─────────────────────────────────────────────────────────────
//

// parseConfig parses a single folder config string in the format:
//
//	/path:uid:gid:mode
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

//
// ─── PERMISSION ENFORCEMENT ─────────────────────────────────────────────────────
//

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

//
// ─── WATCHER ────────────────────────────────────────────────────────────────────
//

// watchFolder sets up a watcher for a folder and enforces permissions on changes.
func watchFolder(cfg FolderConfig) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	if err := watcher.Add(cfg.Path); err != nil {
		slog.Error("failed to watch folder", "path", cfg.Path, "error", err)
		os.Exit(1)
	}

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

	// Event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Filter out noisy CHMOD events
			if event.Op&fsnotify.Chmod != 0 {
				slog.Debug("ignoring chmod event", "path", event.Name)
				continue
			}

			slog.Info("event detected", "path", event.Name, "op", event.Op.String())

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

//
// ─── LOGGING & ENV SETUP ───────────────────────────────────────────────────────
//

// setupLogging configures slog based on DEBUG and TZ env vars.
func setupLogging() {
	// Timezone
	if tz := os.Getenv("TZ"); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			slog.Warn("invalid TZ, falling back to UTC", "tz", tz, "error", err)
			time.Local = time.UTC
		} else {
			time.Local = loc
		}
	}

	// Log level
	level := slog.LevelInfo
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		level = slog.LevelDebug
	}

	slog.Info("Ownarr starting up",
		"tz", time.Now().Location().String(),
		"debug", level == slog.LevelDebug,
	)
}

//
// ─── MAIN ──────────────────────────────────────────────────────────────────────
//

func main() {
	setupLogging()

	env := os.Getenv("WATCH_FOLDERS")
	if env == "" {
		slog.Error("WATCH_FOLDERS environment variable not set")
		os.Exit(1)
	}

	configs := strings.Split(env, ",")
	if len(configs) == 0 {
		slog.Error("no folder configs provided in WATCH_FOLDERS")
		os.Exit(1)
	}

	for _, cfgStr := range configs {
		cfgStr = strings.TrimSpace(cfgStr)
		if cfgStr == "" {
			continue
		}

		cfg, err := parseConfig(cfgStr)
		if err != nil {
			slog.Error("invalid config", "config", cfgStr, "error", err)
			os.Exit(1)
		}

		go watchFolder(cfg)
	}

	// Block forever
	select {}
}
