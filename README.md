# PulseStream ⚡

PulseStream is a real-time, distributed event processing backend built with Go, Redis Streams, and PostgreSQL.

It simulates a high-throughput social media ingestion system using an event-driven architecture with consumer groups and containerized infrastructure.

---

## 🚀 Architecture

Generator → Redis Stream → Consumer Group Workers → PostgreSQL → WebSocket Broadcast

- Producer generates posts continuously
- Redis Streams acts as a durable message log
- Workers (consumer group) process messages
- PostgreSQL stores data permanently
- WebSocket broadcasts real-time updates

---

## 🧠 Key Concepts Implemented

- ✅ Event-driven architecture
- ✅ Redis Streams with Consumer Groups
- ✅ At-least-once message delivery
- ✅ Idempotent database writes (`ON CONFLICT DO NOTHING`)
- ✅ Structured logging using `log/slog`
- ✅ Graceful shutdown with context & WaitGroup
- ✅ Retry logic for service readiness
- ✅ Fully containerized infrastructure (Docker + Compose)
- ✅ Competing consumers pattern
- ✅ Backpressure handling via stream backlog

---

## 🏗 Tech Stack

- **Go 1.24**
- **Redis 7 (Streams + Consumer Groups)**
- **PostgreSQL 15**
- **Docker & Docker Compose**
- **WebSockets**
- **pgx (Postgres driver)**

---

## 🐳 Running the Project

Make sure Docker is installed.

```bash
docker compose up --build