# PulseStream 🚀

Real-time social media analytics engine built in Go.

## Features

- Concurrent post generator
- Worker pool processing
- PostgreSQL persistence
- REST API
- WebSocket real-time broadcasting

## Architecture

Generator → Worker Pool → DB  
                        ↓  
                   WebSocket Hub

## How to Run

1. Start Postgres
2. Run `go run cmd/api/main.go`
3. Open examples/test_client.html
