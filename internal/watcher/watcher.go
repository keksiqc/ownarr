package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
	"github.com/keksiqc/ownarr/internal/config"
)

// Event represents a file system event with associated metadata
type Event struct {
	Path      string          // Full path to the file or directory
	Operation string          // Type of operation (CREATE, WRITE, REMOVE, etc.)
	WatchDir  config.WatchDir // Associated watch directory configuration
	Timestamp time.Time       // When the event occurred
}

// Watcher watches directories for file changes
type Watcher struct {
	logger    *log.Logger
	fsWatcher *fsnotify.Watcher
	events    chan Event
	errors    chan error
	config    *config.Config
	done      chan struct{}  // For coordinating shutdown
	wg        sync.WaitGroup // Wait for goroutines to finish
}

// New creates a new directory watcher
func New(cfg *config.Config, logger *log.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fs watcher: %w", err)
	}

	return &Watcher{
		logger:    logger,
		fsWatcher: fsWatcher,
		events:    make(chan Event, 100),
		errors:    make(chan error, 10),
		config:    cfg,
		done:      make(chan struct{}),
	}, nil
}

// Start begins watching the configured directories
func (w *Watcher) Start(ctx context.Context) error {
	// Add watches for each configured directory
	for _, watchDir := range w.config.WatchDirs {
		if err := w.addWatch(watchDir); err != nil {
			return fmt.Errorf("failed to add watch for %s: %w", watchDir.Path, err)
		}
		w.logger.Info("Started watching directory", "path", watchDir.Path, "recursive", watchDir.Recursive)
	}

	// Start event processing goroutine
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.processEvents(ctx)
	}()

	// Start polling goroutine if poll interval is configured
	if w.config.PollInterval > 0 {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			w.startPolling(ctx)
		}()
		w.logger.Info("Started polling", "interval_seconds", w.config.PollInterval)
	}

	return nil
}

// Events returns the events channel
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Errors returns the errors channel
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Close closes the watcher and releases resources
func (w *Watcher) Close() error {
	// Signal shutdown to all goroutines
	select {
	case <-w.done:
		// Already closed
		return nil
	default:
		close(w.done)
	}

	// Close fsnotify watcher first to stop new events
	var fsErr error
	if w.fsWatcher != nil {
		fsErr = w.fsWatcher.Close()
		if fsErr != nil {
			w.logger.Error("Error closing fsnotify watcher", "error", fsErr)
		}
	}

	// Wait for all goroutines to finish
	w.wg.Wait()

	// Close channels after goroutines are done
	close(w.events)
	close(w.errors)

	return fsErr
}

// startPolling starts the periodic polling process
func (w *Watcher) startPolling(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.config.PollInterval) * time.Second)
	defer ticker.Stop()

	w.logger.Debug("Polling started", "interval", w.config.PollInterval)

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Stopping polling due to context cancellation")
			return
		case <-w.done:
			w.logger.Debug("Stopping polling due to watcher shutdown")
			return
		case <-ticker.C:
			w.logger.Debug("Starting periodic permissions check")
			w.performPeriodicCheck()
		}
	}
}

// performPeriodicCheck walks through all watched directories and checks permissions
func (w *Watcher) performPeriodicCheck() {
	for _, watchDir := range w.config.WatchDirs {
		w.checkDirectoryPermissions(watchDir)
	}
}

// checkDirectoryPermissions recursively checks permissions in a directory
func (w *Watcher) checkDirectoryPermissions(watchDir config.WatchDir) {
	err := filepath.Walk(watchDir.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			w.logger.Warn("Error accessing path during polling", "path", path, "error", err)
			return nil // Continue walking
		}

		// Skip if file should not be processed based on patterns
		if !w.shouldProcess(path, watchDir) {
			return nil
		}

		// Create a synthetic event for the processor
		operation := "POLL_CHECK"
		if info.IsDir() {
			operation = "POLL_CHECK_DIR"
		}

		select {
		case w.events <- Event{
			Path:      path,
			Operation: operation,
			WatchDir:  watchDir,
			Timestamp: time.Now(),
		}:
			w.logger.Debug("Generated polling event", "path", path, "operation", operation)
		case <-w.done:
			return fmt.Errorf("shutdown requested") // Stop walking if shutting down
		default:
			w.logger.Warn("Event channel full during polling, skipping", "path", path)
		}

		return nil
	})

	if err != nil {
		w.logger.Error("Error during periodic check", "path", watchDir.Path, "error", err)
	}
}

// addWatch adds a watch for a directory and optionally its subdirectories
func (w *Watcher) addWatch(watchDir config.WatchDir) error {
	if _, err := os.Stat(watchDir.Path); err != nil {
		if os.IsNotExist(err) {
			w.logger.Warn("Watch directory does not exist", "path", watchDir.Path)
			return nil
		}
		return err
	}

	// Add watch for the directory itself
	if err := w.fsWatcher.Add(watchDir.Path); err != nil {
		return err
	}

	// If recursive, add watches for all subdirectories
	if watchDir.Recursive {
		return filepath.Walk(watchDir.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && path != watchDir.Path {
				if w.shouldExclude(path, watchDir) {
					return filepath.SkipDir
				}

				if err := w.fsWatcher.Add(path); err != nil {
					w.logger.Warn("Failed to add watch for subdirectory", "path", path, "error", err)
				}
			}
			return nil
		})
	}

	return nil
}

// processEvents processes file system events
func (w *Watcher) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Find the matching watch directory
			watchDir := w.findWatchDir(event.Name)
			if watchDir == nil {
				continue
			}

			// Check if the file should be processed
			if !w.shouldProcess(event.Name, *watchDir) {
				continue
			}

			// Convert fsnotify operation to string
			operation := w.operationToString(event.Op)

			// Send event
			select {
			case w.events <- Event{
				Path:      event.Name,
				Operation: operation,
				WatchDir:  *watchDir,
				Timestamp: time.Now(),
			}:
			case <-w.done:
				return
			default:
				w.logger.Warn("Event channel full, dropping event", "path", event.Name)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

			select {
			case w.errors <- err:
			case <-w.done:
				return
			default:
				w.logger.Error("Error channel full, dropping error", "error", err)
			}
		}
	}
}

// findWatchDir finds the watch directory configuration for a given path
func (w *Watcher) findWatchDir(path string) *config.WatchDir {
	for _, watchDir := range w.config.WatchDirs {
		if strings.HasPrefix(path, watchDir.Path) {
			return &watchDir
		}
	}
	return nil
}

// shouldProcess determines if a file should be processed based on include/exclude patterns
func (w *Watcher) shouldProcess(path string, watchDir config.WatchDir) bool {
	filename := filepath.Base(path)

	// Check exclude patterns first
	for _, pattern := range watchDir.Exclude {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return false
		}
	}

	// If include patterns are specified, file must match at least one
	if len(watchDir.Include) > 0 {
		for _, pattern := range watchDir.Include {
			if matched, _ := filepath.Match(pattern, filename); matched {
				return true
			}
		}
		return false
	}

	return true
}

// shouldExclude determines if a directory should be excluded from watching
func (w *Watcher) shouldExclude(path string, watchDir config.WatchDir) bool {
	dirname := filepath.Base(path)

	for _, pattern := range watchDir.Exclude {
		if matched, _ := filepath.Match(pattern, dirname); matched {
			return true
		}
	}
	return false
}

// operationToString converts fsnotify operation to string
func (w *Watcher) operationToString(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return "CREATE"
	case op&fsnotify.Write == fsnotify.Write:
		return "WRITE"
	case op&fsnotify.Remove == fsnotify.Remove:
		return "REMOVE"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "RENAME"
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return "CHMOD"
	default:
		return "UNKNOWN"
	}
}
