# Ownarr Copilot Instructions

## Project Overview
Ownarr is a Go service that enforces file ownership and permissions on media folders through periodic polling. It's designed for containerized environments where maintaining consistent file permissions is critical (e.g., media servers with multiple services).

## Architecture Pattern
- **Single-binary service** with 4 main components in `internal/`:
  - `config/`: Dual configuration (env vars + YAML) with validation
  - `enforcer/`: Core business logic using `filepath.Walk` + goroutines per folder
  - `server/`: Minimal HTTP server (health endpoint only)  
  - `logger/`: Structured logging wrapper
- **Entrypoint**: `cmd/ownarr/main.go` orchestrates startup, graceful shutdown
- **No database**: Stateless design, all config from env/YAML

## Key Development Patterns

### Configuration Loading (`internal/config/config.go`)
- Environment variables override YAML config (precedence order matters)
- Folder configs support granular permissions: `mode` (fallback), `fileMode`, `dirMode`
- All paths must be absolute, validated during config load
- Use `config.Load()` - never construct Config directly

### Enforcement Logic (`internal/enforcer/enforcer.go`)
- One goroutine per configured folder, managed by sync.WaitGroup
- Uses `syscall.Stat_t` for owner checking, not just `os.FileInfo`
- Enforces ownership (`os.Chown`) before permissions (`os.Chmod`) 
- Logs at Info level only when changes made, Debug for inspection

### Error Handling Convention
- Log errors with context, continue processing (don't fail fast)
- Use `logger.WithError(err)` for structured error logging
- Validation errors fail at startup, runtime errors are logged and skipped

## Build & Development

### Local Development
```bash
make run                    # Development server
make build                  # Build to bin/ownarr
go run cmd/ownarr/main.go   # Direct execution
```

### Docker Workflow  
- Multi-stage build: golang:alpine → scratch
- Binary is statically linked and UPX compressed
- Uses distroless base for health checking tools
- Default port 8080, configurable via PORT env var

### Configuration Examples
**Environment**: `FOLDERS='[{"path":"/data","uid":1000,"gid":1000,"mode":755,"fileMode":644}]'`
**YAML**: See `config.example.yaml` for complete examples with comments

## Testing & Debugging
- Health endpoint: `GET /health` (returns 200 OK)
- Enable debug logging: `LOG_LEVEL=debug`
- Test config loading: Check startup logs for "Configuration loaded" message
- Monitor enforcement: Look for "Fixed file" or "Enforcement complete" logs

## Critical Implementation Notes
- **File vs Directory modes**: Always consider if you need separate `fileMode`/`dirMode`
- **Absolute paths required**: Config validation will reject relative paths
- **Goroutine lifecycle**: Use context cancellation for clean shutdown
- **Octal permission format**: Use `fmt.Sprintf("%o", mode)` for logging permissions
- **Container context**: Designed for volume mounts, not host filesystem manipulation