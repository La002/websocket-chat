# WebSocket Chat

A real-time multi-room chat application built with Go, demonstrating WebSocket architecture, Go concurrency patterns, and distributed messaging with Redis.

## Features

- Real-time messaging via WebSockets
- Multi-room support
- JWT-based authentication (stateless, no database)
- Redis Pub/Sub for horizontal scaling
- Room-owned concurrency model (actor pattern)
- Buffered channels with backpressure handling

## Architecture

```
┌─────────────┐     ┌─────────────┐
│  Browser 1  │     │  Browser 2  │
└──────┬──────┘     └──────┬──────┘
       │ WebSocket         │ WebSocket
       ▼                   ▼
┌─────────────────────────────────┐
│           Go Server             │
│  ┌─────────┐    ┌─────────┐    │
│  │ Room 0  │    │ Room 1  │    │
│  │(goroutine)   │(goroutine)   │
│  └────┬────┘    └────┬────┘    │
└───────┼──────────────┼─────────┘
        │              │
        ▼              ▼
┌─────────────────────────────────┐
│         Redis Pub/Sub           │
│   (enables multiple servers)    │
└─────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker (for Redis)

### Run

```bash
# Start Redis
docker run -d --name redis -p 6379:6379 redis

# Copy env file
cp .env.example .env

# Run server
go run cmd/server/main.go
```

Open http://localhost:8080 in your browser.

## Project Structure

```
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── auth/            # JWT authentication
│   ├── config/          # Configuration & logging
│   ├── pubsub/          # Redis Pub/Sub client
│   └── websocket/       # WebSocket handling
│       ├── client.go    # Per-connection handler
│       ├── manager.go   # Connection management
│       ├── room.go      # Room actor (goroutine per room)
│       └── event.go     # Message types
└── frontend/            # Simple HTML/CSS/JS UI
```

## Key Concepts

### Room-Owned Concurrency

Each room runs as an independent goroutine (actor model):
- Owns its client list
- Handles join/leave via channels
- Receives messages from Redis subscription

### Redis Pub/Sub

Enables horizontal scaling:
- Messages published to `room:{name}` channel
- All server instances subscribe and fan out to local clients

### JWT Authentication

Stateless guest authentication:
- No user database needed
- Username stored in JWT claims
- Token validated on WebSocket connection

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Server port |
| LOG_LEVEL | info | debug, info, warn, error |
| REDIS_ADDR | localhost:6379 | Redis address |

## License

MIT
