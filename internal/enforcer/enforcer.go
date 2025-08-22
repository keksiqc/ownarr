package enforcer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/keksiqc/ownarr/internal/config"
	"github.com/keksiqc/ownarr/internal/logger"
)

type Enforcer struct {
	config *config.Config
	logger *logger.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func New(cfg *config.Config, logger *logger.Logger) *Enforcer {
	return &Enforcer{
		config: cfg,
		logger: logger.With("component", "enforcer"),
	}
}

func (e *Enforcer) Start(ctx context.Context) error {
	ctx, e.cancel = context.WithCancel(ctx)

	for _, folder := range e.config.Folders {
		e.wg.Add(1)
		go e.watchFolder(ctx, folder)
	}

	return nil
}

func (e *Enforcer) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
}

func (e *Enforcer) watchFolder(ctx context.Context, folder config.Folder) {
	defer e.wg.Done()

	// Initial enforcement
	e.enforceTree(folder)

	// Set up ticker for periodic enforcement
	ticker := time.NewTicker(e.config.PollInterval)
	defer ticker.Stop()

	e.logger.Info("Started watching folder",
		"path", folder.Path,
		"uid", folder.UID,
		"gid", folder.GID,
		"mode", fmt.Sprintf("%o", folder.Mode))

	for {
		select {
		case <-ticker.C:
			e.enforceTree(folder)
		case <-ctx.Done():
			e.logger.Info("Stopped watching folder", "path", folder.Path)
			return
		}
	}
}

func (e *Enforcer) enforceTree(folder config.Folder) {
	var fixed, skipped, failed int

	err := filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			e.logger.Error("Error accessing path", "path", path, "error", err)
			failed++
			return nil
		}

		changed, err := e.enforceFile(folder, path, info)
		if err != nil {
			e.logger.Error("Error enforcing file", "path", path, "error", err)
			failed++
			return nil
		}

		if changed {
			fixed++
		} else {
			skipped++
		}

		return nil
	})

	if err != nil {
		e.logger.Error("Error walking folder", "path", folder.Path, "error", err)
	}

	if fixed > 0 || failed > 0 {
		e.logger.Info("Enforcement complete",
			"folder", folder.Path,
			"fixed", fixed,
			"skipped", skipped,
			"failed", failed)
	}
}

func (e *Enforcer) enforceFile(folder config.Folder, path string, info os.FileInfo) (bool, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false, nil
	}

	changed := false

	// Check ownership
	if int(stat.Uid) != folder.UID || int(stat.Gid) != folder.GID {
		if err := os.Chown(path, folder.UID, folder.GID); err != nil {
			return false, fmt.Errorf("chown: %w", err)
		}
		changed = true
	}

	// Check permissions
	currentMode := info.Mode() & os.ModePerm
	targetMode := folder.Mode & os.ModePerm
	if currentMode != targetMode {
		if err := os.Chmod(path, folder.Mode); err != nil {
			return false, fmt.Errorf("chmod: %w", err)
		}
		changed = true
	}

	return changed, nil
}
