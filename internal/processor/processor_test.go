package processor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/keksiqc/ownarr/internal/config"
	"github.com/keksiqc/ownarr/internal/watcher"
	"github.com/stretchr/testify/assert"
)

func TestProcessor(t *testing.T) {
	// Create a test logger that discards output
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Minimize test output

	processor := New(logger)
	assert.NotNil(t, processor)

	// Create test channels
	events := make(chan watcher.Event, 1)
	errors := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start processing in a goroutine
	go processor.Process(ctx, events, errors)

	// Send a test event
	testEvent := watcher.Event{
		Path:      "/tmp/testfile.txt",
		Operation: "CREATE",
		WatchDir: config.WatchDir{
			Path:     "/tmp",
			FileMode: "0644",
			DirMode:  "0755",
		},
		Timestamp: time.Now(),
	}

	events <- testEvent

	// Wait for context to complete
	<-ctx.Done()

	// Close channels
	close(events)
	close(errors)
}

func TestHandleEvent(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)

	processor := New(logger)

	testEvent := watcher.Event{
		Path:      "/tmp/testfile.txt",
		Operation: "CREATE",
		WatchDir: config.WatchDir{
			Path:     "/tmp",
			FileMode: "0644",
			DirMode:  "0755",
		},
		Timestamp: time.Now(),
	}

	// This should not panic
	processor.handleEvent(testEvent)

	// Test with different operations
	operations := []string{"CREATE", "WRITE", "REMOVE", "RENAME", "CHMOD", "UNKNOWN"}
	for _, op := range operations {
		testEvent.Operation = op
		processor.handleEvent(testEvent)
	}
}
