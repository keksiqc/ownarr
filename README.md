# ownarr

A lightweight, efficient file watcher and permission manager written in Go.

## Features

- **Real-time File System Watching**: Monitor directories for file changes using efficient OS-native file system notifications (fsnotify)
- **Periodic Polling**: Optional periodic permission checking to ensure compliance
- **Permission Management**: Automatically set correct file and directory permissions
- **Pattern Matching**: Include/exclude files based on configurable patterns
- **Recursive Watching**: Optionally watch subdirectories recursively  
- **Structured Logging**: Beautiful, structured logs using charmbracelet/log
- **Configuration**: Flexible YAML configuration using koanf
- **Lightweight**: Small memory footprint and efficient resource usage
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### From Source

```bash
git clone https://github.com/keksiqc/ownarr.git
cd ownarr
make build
```

The binary will be available in `./build/ownarr`.

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/keksiqc/ownarr/releases).

## Quick Start

1. **Create a configuration file**:
```bash
cp config.example.yaml config.yaml
```

2. **Edit the configuration** to match your needs:
```yaml
log_level: "info"
poll_interval: 30

watch_dirs:
  - path: "/your/media/path"
    recursive: true
    exclude:
      - "*.tmp"
      - ".DS_Store"
    include:
      - "*.mp4"
      - "*.mkv"
      - "*.avi"
    file_mode: "0644"
    dir_mode: "0755"
```

3. **Run ownarr**:
```bash
./build/ownarr -config config.yaml
```

## Usage
### Command Line Options

```bash
./ownarr -help
```

Available options:
- `-config`: Path to configuration file (default: "config.yaml")
- `-version`: Show version information
- `-help`: Show help information

### Basic Usage

```bash
# Use default config file (config.yaml)
./ownarr

# Use custom config file
./ownarr -config /path/to/my-config.yaml

# Show version
./ownarr -version
```

## Configuration

ownarr uses YAML configuration files. See [config.example.yaml](config.example.yaml) for a complete example.

### Configuration Structure

```yaml
# Logging level: debug, info, warning, error, critical
log_level: "info"

# Interval in seconds between periodic permission checks
# Set to 0 to disable polling (only real-time events)
poll_interval: 30

# Directories to watch for changes
watch_dirs:
  - path: "/data/media"           # Required: directory path to watch
    recursive: true               # Optional: watch subdirectories (default: false)
    exclude:                      # Optional: patterns to exclude from processing
      - "temp"
      - "*.tmp"
      - "*.bak"
      - ".DS_Store"
    include:                      # Optional: patterns to explicitly include
      - "*.mp4"                   # If specified, only matching files are processed
      - "*.mkv"
      - "*.avi"
    file_mode: "0644"            # Required: permissions for files (octal format)
    dir_mode: "0755"             # Required: permissions for directories (octal format)
```

### Configuration Options

#### Global Settings
- **log_level**: Controls logging verbosity (`debug`, `info`, `warning`, `error`, `critical`)
- **poll_interval**: Seconds between periodic permission checks (0 = disabled, real-time only)

#### Watch Directory Settings
- **path**: Absolute path to directory to monitor (required)
- **recursive**: Whether to watch subdirectories recursively (default: false)
- **exclude**: List of glob patterns to exclude from processing
- **include**: List of glob patterns to explicitly include (if empty, all non-excluded files processed)
- **file_mode**: Octal permissions for files (e.g., "0644", "0600")
- **dir_mode**: Octal permissions for directories (e.g., "0755", "0700")

### Pattern Matching

Patterns support standard shell glob syntax:
- `*.tmp` - matches all .tmp files
- `temp*` - matches files starting with "temp"
- `???.log` - matches 3-character files with .log extension
- `.DS_Store` - matches exact filename

**Pattern Priority**: Exclude patterns override include patterns.

## How It Works

ownarr operates in two modes:

### 1. Real-time Monitoring (fsnotify)
- Uses OS-native file system notifications (inotify on Linux, FSEvents on macOS)
- Immediate response to file system changes
- Low CPU usage, event-driven
- Handles: CREATE, WRITE, REMOVE, RENAME, CHMOD events

### 2. Periodic Polling (optional)
- Walks through all watched directories at configured intervals
- Ensures permissions stay correct even if real-time events are missed
- Configurable via `poll_interval` (set to 0 to disable)
- Useful for catching permission drift or missed events

## Examples

### Basic Media Directory Monitoring
```yaml
log_level: "info"
poll_interval: 300  # Check every 5 minutes

watch_dirs:
  - path: "/media/movies"
    recursive: true
    include:
      - "*.mp4"
      - "*.mkv"
      - "*.avi"
    exclude:
      - "*.tmp"
    file_mode: "0644"
    dir_mode: "0755"
```

### Multiple Directories with Different Settings
```yaml
log_level: "debug"
poll_interval: 60

watch_dirs:
  # Public media - more permissive
  - path: "/media/public"
    recursive: true
    file_mode: "0644"
    dir_mode: "0755"
    
  # Private documents - restrictive
  - path: "/data/private"
    recursive: false
    exclude:
      - "*.log"
      - "cache"
    file_mode: "0600"
    dir_mode: "0700"
```

### Real-time Only (No Polling)
```yaml
log_level: "info"
poll_interval: 0  # Disable periodic checks

watch_dirs:
  - path: "/fast/ssd/data"
    recursive: true
    file_mode: "0644"
    dir_mode: "0755"
```

## Building from Source

### Prerequisites
- Go 1.24+
- make
- golangci-lint (for linting)

### Build Commands
```bash
# Build binary
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Run all quality checks
make all
```

### Development
```bash
# Clone repository
git clone https://github.com/keksiqc/ownarr.git
cd ownarr

# Install dependencies
go mod download

# Build and test
make all

# Run with development config
./build/ownarr -config config.example.yaml
```

## Docker Support

### Dockerfile
The repository includes an optimized multi-stage Dockerfile for containerized deployments:

```dockerfile
# Build stage - uses Go 1.25 Alpine
FROM golang:1.25-alpine AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o /out/ownarr ./cmd/ownarr

# Final stage - minimal scratch image
FROM scratch
COPY --from=build /out/ownarr /ownarr
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ENTRYPOINT ["/ownarr"]
```

This produces a minimal ~8MB container image.

### Docker Build
```bash
docker build -t ownarr:latest .
```

### Docker Run Examples

**Basic usage with bind mount:**
```bash
docker run -v /your/data:/data -v ./config.yaml:/config.yaml ownarr:latest /ownarr -config /config.yaml
```

**With Docker Compose:**
```yaml
version: '3.8'
services:
  ownarr:
    build: .
    volumes:
      - /your/media:/data/media
      - ./config.yaml:/config.yaml:ro
    command: ["/ownarr", "-config", "/config.yaml"]
    restart: unless-stopped
```

**Using environment for simple setups:**
```bash
# Note: You'll need to create a config file and mount it
docker run -d \
  --name ownarr \
  -v /your/media:/data/media \
  -v ./config.yaml:/config.yaml:ro \
  ownarr:latest /ownarr -config /config.yaml
```

## Architecture

ownarr follows a clean, modular architecture:

- **config**: Configuration loading and validation using koanf
- **watcher**: File system monitoring using fsnotify with polling support
- **processor**: Event processing and permission management
- **main**: Application entry point and lifecycle management

The application is designed to be:
- **Memory efficient**: Minimal resource usage
- **Concurrent**: Handles multiple events simultaneously
- **Fault tolerant**: Graceful error handling and recovery
- **Observable**: Comprehensive logging for debugging

## Dependencies

- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications
- [koanf](https://github.com/knadh/koanf) - Configuration management
- [charmbracelet/log](https://github.com/charmbracelet/log) - Structured logging
- [testify](https://github.com/stretchr/testify) - Testing framework

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make test`)
6. Ensure linting is clean (`make lint`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release
- Real-time file system watching with fsnotify
- Periodic polling support
- Pattern-based include/exclude filtering
- YAML configuration with koanf
- Structured logging with charmbracelet/log
- Cross-platform support
- Comprehensive test coverage
- Race condition-free concurrent operations
