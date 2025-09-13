# ownarr

Ownarr enforces file ownership and permissions on your media folders with support for separate file and folder permissions, and automatic Trash Guides folder structure setup.

## Features

- 🔧 **Separate File & Folder Permissions**: Configure different permissions for files vs directories
- 📁 **Trash Guides Integration**: Automatically create and manage the recommended Trash Guides folder structure
- ⚙️ **Flexible Configuration**: YAML configuration with environment variable support
- 🔄 **Periodic Monitoring**: Configurable polling interval for continuous enforcement
- 🌐 **Health Monitoring**: Built-in HTTP health check endpoint
- 📊 **Rich Logging**: Beautiful colored logging with configurable levels

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

### Trash Guides Integration

Ownarr can automatically create and manage the [Trash Guides](https://trash-guides.info/) recommended folder structure:

```yaml
trashGuides:
  enabled: true
  type: "usenet"  # "usenet" or "torrent"
  rootPath: "/data"
  mediaTypes: ["movies", "tv", "music", "books"]
  createStructure: true
  uid: 1000
  gid: 1000
  folderMode: 755
  fileMode: 644
```

This will create the recommended structure:
- **Usenet**: `/data/{media,torrents,usenet}/{movies,tv,music,books}` with `usenet/complete` and `usenet/incomplete`
- **Torrent**: `/data/{media,torrents}/{movies,tv,music,books}`

### Separate File & Folder Permissions

Configure different permissions for files and directories:

```yaml
folders:
  - path: "/data/media"
    uid: 1000
    gid: 1000
    folderMode: 755  # rwxr-xr-x for directories
    fileMode: 644    # rw-r--r-- for files
```

The configuration file supports all the same options as the environment variables plus advanced features:

```yaml
# Basic configuration
port: 8080
logLevel: "info"
pollInterval: "30s"
timezone: "UTC"

# Trash Guides folder structure setup (optional)
trashGuides:
  enabled: true
  type: "usenet"  # "usenet" or "torrent"
  rootPath: "/data"
  mediaTypes: ["movies", "tv", "music", "books"]
  createStructure: true
  uid: 1000
  gid: 1000
  folderMode: 755
  fileMode: 644

# Folder monitoring with separate file/folder permissions
folders:
  - path: "/data/media"
    uid: 1000
    gid: 1000
    folderMode: 755  # Permissions for directories
    fileMode: 644    # Permissions for files
  - path: "/data/downloads"
    uid: 1000
    gid: 1000
    mode: 755        # Legacy: same permissions for files and folders
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

**Docker Compose with Trash Guides:**
```yaml
services:
  ownarr:
    image: ghcr.io/keksiqc/ownarr:latest
    environment:
      CONFIG_FILE: "/config.yaml"
      TZ: "Europe/Berlin"
    volumes:
      - ./ownarr-config.yaml:/config.yaml
      - ./data:/data
```

**Traditional Docker:**
```bash
docker run -e FOLDERS="/data:1000:1000:775" -v ./data:/data ghcr.io/keksiqc/ownarr:latest
```

**Binary with separate permissions:**
```bash
export CONFIG_FILE="./config.yaml"
./ownarr
```

## Health Check

The service exposes a health check endpoint at `http://localhost:8080/health`

## Building

```bash
make build
```

Or with Docker:
```bash
docker build -t ownarr .
```

## Configuration Reference

### Folder Configuration Formats

**New format with separate permissions (recommended):**
```yaml
folders:
  - path: "/path/to/folder"
    uid: 1000
    gid: 1000
    folderMode: 755  # Permissions for directories (rwxr-xr-x)
    fileMode: 644    # Permissions for files (rw-r--r--)
```

**Legacy format (backward compatible):**
```yaml
folders:
  - path: "/path/to/folder"
    uid: 1000
    gid: 1000
    mode: 755  # Same permissions for both files and directories
```

**Environment variable format:**
Each folder configuration uses the format: `/path:uid:gid:mode`

- **path**: Absolute path to the folder
- **uid**: User ID to set ownership to
- **gid**: Group ID to set ownership to
- **mode**: Octal permissions (e.g., 775 for rwxrwxr-x)

Multiple folders can be specified by separating with commas.

### Trash Guides Configuration

| Option | Type | Description |
|--------|------|-------------|
| `enabled` | bool | Enable Trash Guides folder structure |
| `type` | string | Either "usenet" or "torrent" |
| `rootPath` | string | Base path where structure will be created |
| `mediaTypes` | []string | Media types to create folders for (default: movies, tv, music, books) |
| `createStructure` | bool | Whether to create directories automatically |
| `uid` | int | User ID for created directories |
| `gid` | int | Group ID for created directories |
| `folderMode` | int | Permissions for directories (octal) |
| `fileMode` | int | Permissions for files (octal) |
