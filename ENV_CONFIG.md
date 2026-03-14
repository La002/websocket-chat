# Environment Variables Configuration

This application can be configured using the following environment variables:

## Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port number for the HTTP server |
| `FRONTEND_DIR` | `./frontend` | Directory containing frontend files |

## Logger Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `ENV` | `development` | Environment mode (`development`, `production`) |

## Example Usage

```bash
# Development (with pretty console logs)
PORT=3000 LOG_LEVEL=debug ENV=development ./websocket-chat

# Production (with JSON logs)
PORT=8080 LOG_LEVEL=info ENV=production ./websocket-chat
```

## Configuration Priority

1. Environment variables (highest priority)
2. Default values in code (fallback)
