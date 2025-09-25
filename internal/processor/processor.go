package processor

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/keksiqc/ownarr/internal/watcher"
)

// Processor handles file system events
type Processor struct {
	logger *log.Logger
}

// New creates a new event processor
func New(logger *log.Logger) *Processor {
	return &Processor{
		logger: logger,
	}
}

// Process processes file system events
func (p *Processor) Process(ctx context.Context, events <-chan watcher.Event, errors <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-events:
			if !ok {
				return
			}
			p.handleEvent(event)

		case err, ok := <-errors:
			if !ok {
				return
			}
			p.logger.Error("Watcher error", "error", err)
		}
	}
}

// handleEvent processes a single file system event
func (p *Processor) handleEvent(event watcher.Event) {
	p.logger.Info("Processing file event",
		"path", event.Path,
		"operation", event.Operation,
		"timestamp", event.Timestamp.Format(time.RFC3339),
	)

	switch event.Operation {
	case "CREATE":
		p.handleCreate(event)
	case "WRITE":
		p.handleWrite(event)
	case "REMOVE":
		p.handleRemove(event)
	case "RENAME":
		p.handleRename(event)
	case "CHMOD":
		p.handleChmod(event)
	case "POLL_CHECK":
		p.handlePollCheck(event)
	case "POLL_CHECK_DIR":
		p.handlePollCheckDir(event)
	default:
		p.logger.Warn("Unknown operation", "operation", event.Operation, "path", event.Path)
	}
}

// handleCreate handles file/directory creation events
func (p *Processor) handleCreate(event watcher.Event) {
	stat, err := os.Stat(event.Path)
	if err != nil {
		p.logger.Error("Failed to stat created file", "path", event.Path, "error", err)
		return
	}

	if stat.IsDir() {
		p.logger.Info("Directory created", "path", event.Path)
		p.fixPermissions(event.Path, event.WatchDir.DirMode, true)
	} else {
		p.logger.Info("File created", "path", event.Path, "size", stat.Size())
		p.fixPermissions(event.Path, event.WatchDir.FileMode, false)
	}
}

// handleWrite handles file modification events
func (p *Processor) handleWrite(event watcher.Event) {
	stat, err := os.Stat(event.Path)
	if err != nil {
		p.logger.Error("Failed to stat modified file", "path", event.Path, "error", err)
		return
	}

	p.logger.Info("File modified", "path", event.Path, "size", stat.Size())
	p.fixPermissions(event.Path, event.WatchDir.FileMode, false)
}

// handleRemove handles file/directory removal events
func (p *Processor) handleRemove(event watcher.Event) {
	p.logger.Info("File or directory removed", "path", event.Path)
}

// handleRename handles file/directory rename events
func (p *Processor) handleRename(event watcher.Event) {
	p.logger.Info("File or directory renamed", "path", event.Path)
}

// handleChmod handles permission change events
func (p *Processor) handleChmod(event watcher.Event) {
	p.logger.Debug("File permissions changed", "path", event.Path)
}

// handlePollCheck handles periodic permission checks for files
func (p *Processor) handlePollCheck(event watcher.Event) {
	stat, err := os.Stat(event.Path)
	if err != nil {
		// File might have been deleted between poll generation and processing
		p.logger.Debug("Failed to stat file during polling", "path", event.Path, "error", err)
		return
	}

	if !stat.IsDir() {
		p.logger.Debug("Polling check: file", "path", event.Path, "size", stat.Size())
		p.fixPermissions(event.Path, event.WatchDir.FileMode, false)
	}
}

// handlePollCheckDir handles periodic permission checks for directories
func (p *Processor) handlePollCheckDir(event watcher.Event) {
	stat, err := os.Stat(event.Path)
	if err != nil {
		p.logger.Debug("Failed to stat directory during polling", "path", event.Path, "error", err)
		return
	}

	if stat.IsDir() {
		p.logger.Debug("Polling check: directory", "path", event.Path)
		p.fixPermissions(event.Path, event.WatchDir.DirMode, true)
	}
}

// fixPermissions sets the correct permissions on a file or directory
func (p *Processor) fixPermissions(path string, modeStr string, isDir bool) {
	// Validate mode string is not empty
	if modeStr == "" {
		p.logger.Warn("Empty mode string provided", "path", path)
		return
	}

	// Parse the mode string (e.g., "0644" -> 0644)
	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		p.logger.Error("Invalid file mode format", "mode", modeStr, "path", path, "error", err)
		return
	}

	fileMode := os.FileMode(mode)

	// Get current permissions
	stat, err := os.Stat(path)
	if err != nil {
		p.logger.Error("Failed to stat file for permission fix", "path", path, "error", err)
		return
	}

	currentMode := stat.Mode().Perm()

	// Only change permissions if they're different
	if currentMode != fileMode {
		if err := os.Chmod(path, fileMode); err != nil {
			p.logger.Error("Failed to fix permissions", "path", path, "mode", modeStr, "error", err)
			return
		}

		entityType := "file"
		if isDir {
			entityType = "directory"
		}

		p.logger.Info("Fixed permissions",
			"path", path,
			"type", entityType,
			"old_mode", currentMode,
			"new_mode", fileMode,
		)
	}
}
