package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 30, cfg.PollInterval)
	assert.Empty(t, cfg.WatchDirs)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				LogLevel:     "info",
				PollInterval: 30,
				WatchDirs: []WatchDir{
					{
						Path:      "/tmp",
						Recursive: true,
						FileMode:  "0644",
						DirMode:   "0755",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid poll interval",
			config: &Config{
				LogLevel:     "info",
				PollInterval: 0,
				WatchDirs:    []WatchDir{},
			},
			wantErr: true,
		},
		{
			name: "missing watch dir path",
			config: &Config{
				LogLevel:     "info",
				PollInterval: 30,
				WatchDirs: []WatchDir{
					{
						Path:      "",
						Recursive: true,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
log_level: "debug"
poll_interval: 60
watch_dirs:
  - path: "/data/media"
    recursive: true
    exclude:
      - "temp"
      - "*.tmp"
    include:
      - "*.mp4"
      - "*.mkv"
    file_mode: "0644"
    dir_mode: "0755"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Remove(tmpfile.Name()))
	}()

	_, err = tmpfile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Load the config
	cfg, err := Load(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 60, cfg.PollInterval)
	assert.Len(t, cfg.WatchDirs, 1)

	watchDir := cfg.WatchDirs[0]
	assert.Equal(t, "/data/media", watchDir.Path)
	assert.True(t, watchDir.Recursive)
	assert.Equal(t, []string{"temp", "*.tmp"}, watchDir.Exclude)
	assert.Equal(t, []string{"*.mp4", "*.mkv"}, watchDir.Include)
	assert.Equal(t, "0644", watchDir.FileMode)
	assert.Equal(t, "0755", watchDir.DirMode)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file not found")
}
