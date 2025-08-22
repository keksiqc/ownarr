package enforcer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/keksiqc/ownarr/internal/config"
)

type Enforcer struct {
	config *config.Config
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func New(cfg *config.Config) *Enforcer {
	return &Enforcer{
		config: cfg,
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

	log.Printf("Started watching %s (uid=%d gid=%d mode=%o)",
		folder.Path, folder.UID, folder.GID, folder.Mode)

	for {
		select {
		case <-ticker.C:
			e.enforceTree(folder)
		case <-ctx.Done():
			log.Printf("Stopped watching %s", folder.Path)
			return
		}
	}
}

func (e *Enforcer) enforceTree(folder config.Folder) {
	var fixed, skipped, failed int

	err := filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing %s: %v", path, err)
			failed++
			return nil
		}

		changed, err := e.enforceFile(folder, path, info)
		if err != nil {
			log.Printf("Error enforcing %s: %v", path, err)
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
		log.Printf("Error walking %s: %v", folder.Path, err)
	}

	if fixed > 0 || failed > 0 {
		log.Printf("Enforcement complete for %s: fixed=%d skipped=%d failed=%d",
			folder.Path, fixed, skipped, failed)
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
