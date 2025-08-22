# ownarr

Ownarr enforces file ownership and permissions on your media folders.

## Usage

### Environment Variables

- `FOLDERS` - Required. Comma-separated list of folder configurations in format: `/path:uid:gid:mode`
- `PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `POLL_INTERVAL` - How often to check permissions (default: 30s)
- `TZ` - Timezone (default: UTC)
- `CONFIG_FILE` - Optional YAML config file path

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
