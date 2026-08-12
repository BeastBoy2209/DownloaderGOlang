# 📥 DownloaderGOlang

**An asynchronous, concurrent file-download service written in idiomatic Go — built as a hands-on exercise in clean architecture, goroutine lifecycle management, and reliable background processing.**

<p align="left">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/PostgreSQL-storage-336791?logo=postgresql&logoColor=white">
  <img alt="Echo" src="https://img.shields.io/badge/Echo-v5-3E863D">
  <img alt="Status" src="https://img.shields.io/badge/status-academic%20project-yellow">
</p>

---

## ✨ Overview

`DownloaderGOlang` is a pet/academic backend project that accepts a batch of URLs, downloads them **concurrently in the background**, and lets clients poll for status and stream the resulting files back out — all without blocking the initial request.

It was built as a capstone exercise to practice production-grade Go patterns rather than to ship a product: clean architecture layering, interface-driven design, context propagation, controlled goroutine fan-out, and graceful shutdown.

## 🧩 How it works

1. **Submit a job** — `POST /downloads` with a list of URLs and a timeout. The server immediately creates a job record and returns its `id` with status `PROCESS`.
2. **Download in the background** — a pool of goroutines fetches each URL in parallel, bounded by a concurrency limit and a single timeout that applies to the *whole batch*, not per file.
3. **Partial failure is fine** — if one URL fails (timeout, 404, network error, etc.), the rest keep downloading. The job is marked `DONE` once every file has either succeeded or been cut off by the deadline.
4. **Retrieve results** — `GET /downloads/{id}` reports the status of every file (its `file_id` on success, an error code on failure). `GET /downloads/{id}/files/{file_id}` streams the raw bytes back.

## 🔌 API

### `POST /downloads`
Create a download job. Returns immediately.

```json
// Request
{
  "files": [
    { "url": "https://google.com" },
    { "url": "https://somehost.com/test.pdf" }
  ],
  "timeout": "60s"
}

// Response
{
  "id": 12,
  "status": "PROCESS"
}
```

### `GET /downloads/{id}`
Check job status.

```json
// Happy path
{
  "id": 12,
  "status": "DONE",
  "files": [
    { "url": "https://google.com", "file_id": 79 },
    { "url": "https://bing.com",  "file_id": 80 }
  ]
}

// Partial failure
{
  "id": 12,
  "status": "DONE",
  "files": [
    { "url": "https://google.com", "error": { "code": "TIMEOUT" } },
    { "url": "https://bing.com", "file_id": 80 }
  ]
}
```

### `GET /downloads/{id}/files/{file_id}`
Streams the downloaded file's raw bytes (`Content-Type: application/octet-stream`).

## 🏗 Architecture & design principles

The service is built around a few deliberate engineering constraints:

| Principle | How it's applied |
|---|---|
| **Clean architecture** | Responsibilities are split across `transport`, `domain`, `usecases`, and `adapters`. Dependencies point inward — the domain never depends on HTTP or storage concerns. |
| **Interfaces everywhere** | File storage is an abstract `repository` interface (implemented on top of PostgreSQL), enabling dependency inversion and easy mocking in tests. |
| **Context-aware** | The incoming request context is propagated through every downstream call, with explicit timeouts on all outbound work. |
| **Bounded concurrency** | Downloads run in goroutines coordinated with `errgroup`/semaphores, with a hard concurrency cap and context-based cancellation the moment a goroutine's work is no longer needed. |
| **Graceful shutdown** | OS signals are caught and the service winds down in-flight work via context cancellation, within a fixed shutdown budget. |
| **Sane error semantics** | Business errors (e.g. file too large) return `2xx` with a reason; client errors return `4xx`; only genuine server failures return `5xx`. |
| **Middleware** | Panic recovery and request-ID injection (generated as a UUID when absent) on every HTTP request. |
| **Tested usecases** | Core usecases/handlers are covered by mocked-dependency tests for both the happy path and failure paths. |

## 🛠 Tech stack

- **Language:** Go `1.26`
- **HTTP:** [`labstack/echo`](https://github.com/labstack/echo) v5
- **Database:** PostgreSQL via [`jackc/pgx`](https://github.com/jackc/pgx) + [`jmoiron/sqlx`](https://github.com/jmoiron/sqlx)
- **Configuration:** [`caarlos0/env`](https://github.com/caarlos0/env) + [`joho/godotenv`](https://github.com/joho/godotenv)
- **Concurrency:** `golang.org/x/sync` (errgroup)
- **Testing:** [`stretchr/testify`](https://github.com/stretchr/testify) + [`uber-go/mock`](https://github.com/uber-go/mock)

## 📁 Project structure

```
.
├── cmd/app/            # Application entrypoint
├── db/migrations/       # SQL migrations for PostgreSQL
├── internal/            # Core application code (transport, usecases, domain, adapters)
├── go.mod / go.sum      # Go module definition
```

## 🚀 Getting started

### Prerequisites
- Go 1.26+
- A running PostgreSQL instance

### Setup

```bash
# 1. Clone the repository
git clone https://github.com/BeastBoy2209/DownloaderGOlang.git
cd DownloaderGOlang

# 2. Configure environment variables
cp .env.example .env   # then fill in your DB credentials and server settings

# 3. Apply database migrations
# (run the SQL files in db/migrations against your PostgreSQL instance)

# 4. Install dependencies
go mod download

# 5. Run the service
go run ./cmd/app
```

### Running tests

```bash
go test ./...
```

## 📚 About this project

This repository was built as a **final/capstone assignment** for a Go backend course, aimed at demonstrating a solid understanding of:

- Designing an async job-processing HTTP service from a written contract/spec
- Structuring a Go service with clean, testable layers
- Managing goroutine lifecycles safely under timeouts and cancellation
- Writing meaningful, mocked unit tests

It is **not** intended for production use — it's a learning artifact, shared for portfolio and educational purposes.

## 📄 License

No license has been specified for this repository. All rights reserved by the author unless stated otherwise.