# ownarr

Ownarr enforces file ownership and permissions on your media folders.

## Usage

You can configure ownarr using either environment variables or a YAML configuration file. 
Environment variables take precedence over config file settings.

### Environment Variables

- `FOLDERS` - Required. Comma-separated list of folder configurations in format: `/path:uid:gid:mode`
- `PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `POLL_INTERVAL` - How often to check permissions (default: 30s)
- `TZ` - Timezone (default: UTC)
- `CONFIG_FILE` - Optional YAML config file path

### Configuration File

Instead of environment variables, you can use a YAML configuration file. 
To use a config file, set the `CONFIG_FILE` environment variable to the path of your config file:

```bash
export CONFIG_FILE="/path/to/config.yaml"
./ownarr
```

Or with Docker:
```bash
docker run -e CONFIG_FILE="/config.yaml" -v ./config.yaml:/config.yaml -v ./data:/data ghcr.io/keksiqc/ownarr:latest
```

The configuration file supports all the same options as the environment variables:

```yaml
# Port to listen on (default: 8080)
port: 8080

# Log level (default: "info")
logLevel: "info"

# Polling interval for checking folder changes (default: "30s")
pollInterval: "30s"

# Timezone for logging and scheduling (default: "UTC")
timezone: "UTC"

# List of folders to monitor (new format - recommended)
folders:
  - path: "/data/media"
    uid: 1000
    gid: 1000
    mode: 755
  - path: "/data/downloads"
    uid: 1000
    gid: 1000
    mode: 755
```

See [config.example.yaml](config.example.yaml) for a complete example with comments.

### Legacy Configuration Format

The legacy folder configuration format is still supported for backward compatibility:

```yaml
folders:
  - "/data/media:1000:1000:755"
  - "/data/downloads:1000:1000:755"
```

### Examples

**Docker Compose:**
```yaml
services:
  ownarr:
    image: ghcr.io/keksiqc/ownarr:latest
    environment:
      FOLDERS: "/movies:1000:1000:775,/tv:1000:1000:775"
      TZ: "Europe/Berlin"
    volumes:
      - ./movies:/movies
      - ./tv:/tv
```

**Docker:**
```bash
docker run -e FOLDERS="/data:1000:1000:775" -v ./data:/data ghcr.io/keksiqc/ownarr:latest
```

**Binary:**
```bash
export FOLDERS="/data:1000:1000:775"
./ownarr
```

## Health Check

The service exposes a health check endpoint at `http://localhost:8080/health`

## Building

```bash
docker build -t ownarr .
```

## Configuration Format

Each folder configuration uses the format: `/path:uid:gid:mode`

- **path**: Absolute path to the folder
- **uid**: User ID to set ownership to
- **gid**: Group ID to set ownership to
- **mode**: Octal permissions (e.g., 775 for rwxrwxr-x)

Multiple folders can be specified by separating with commas.
