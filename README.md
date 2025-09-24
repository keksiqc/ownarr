# ownarr

Ownarr enforces file ownership and permissions on your media folders.

## Usage

You can configure ownarr using either environment variables or a YAML configuration file. 
Environment variables take precedence over config file settings.

### Environment Variables

- `FOLDERS` - Required. YAML list of folder configurations (see format below)
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

# List of folders to monitor
# You can use just 'mode' for all files/directories, or specify 'fileMode' and 'dirMode' separately
folders:
  - path: "/data/media"
    uid: 1000
    gid: 1000
    mode: 755        # Default mode for the folder, and fallback for files/dirs if not specified
    fileMode: 644    # Optional: specific permissions for files (uses 'mode' if not specified)
    dirMode: 755     # Optional: specific permissions for subdirectories (uses 'mode' if not specified)
  - path: "/data/downloads"
    uid: 1000
    gid: 1000
    mode: 755        # If fileMode and dirMode are omitted, everything uses this mode
```

See [config.example.yaml](config.example.yaml) for a complete example with comments.


### Examples

**Docker Compose:**
```yaml
services:
  ownarr:
    image: ghcr.io/keksiqc/ownarr:latest
    environment:
      FOLDERS: '[{"path":"/movies","uid":1000,"gid":1000,"mode":775},{"path":"/tv","uid":1000,"gid":1000,"mode":775}]'
      TZ: "Europe/Berlin"
    volumes:
      - ./movies:/movies
      - ./tv:/tv
```

**Docker:**
```bash
docker run -e FOLDERS='[{"path":"/data","uid":1000,"gid":1000,"mode":775}]' -v ./data:/data ghcr.io/keksiqc/ownarr:latest
```

**Binary:**
```bash
export FOLDERS='[{"path":"/data","uid":1000,"gid":1000,"mode":755}]'
./ownarr
```

## Health Check

The service exposes a health check endpoint at `http://localhost:8080/health`

## Building

```bash
docker build -t ownarr .
```

## Configuration Format

Each folder configuration in the YAML file uses the following format:

- **path**: Absolute path to the folder
- **uid**: User ID to set ownership to
- **gid**: Group ID to set ownership to
- **mode**: Default octal permissions for the main folder and fallback for files/dirs if fileMode/dirMode not specified
- **fileMode**: (Optional) Octal permissions for files within the folder (uses 'mode' if not specified)
- **dirMode**: (Optional) Octal permissions for subdirectories within the folder (uses 'mode' if not specified)

For environment variables, use a YAML array format:
```bash
FOLDERS='[{"path":"/data","uid":1000,"gid":1000,"mode":775,"fileMode":644,"dirMode":755}]'
```

You can use just 'mode' if you want the same permissions for everything, or specify 'fileMode' and/or 'dirMode' for more granular control.
