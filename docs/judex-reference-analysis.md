# Judex Reference Analysis

> **Purpose**: This document records our analysis of the Judex open-source project, which we used as a **technical reference** before designing and building our own Online Code Execution System. All conclusions below are based on direct inspection of the Judex source code in `/Users/keerthivardhanchandana/Desktop/va-project/judex/`.

---

## Judex Overview

Judex is a distributed, production-oriented online judge backend written in Go. It is authored by Dharshan2208 (GitHub module: `github.com/Dharshan2208/judex`). It handles code submission, execution inside Docker containers, result storage, and exposes REST APIs. Its architecture is designed for multi-instance horizontal scaling, making it significantly more complex than what a single-machine university project requires.

---

## Architecture Observed

Judex runs as **two separate binaries** (confirmed from `cmd/api/main.go` and `cmd/worker/`):

| Binary | Role |
|--------|------|
| `api`  | Accepts HTTP submissions, stores job metadata in Redis, pushes job IDs onto a Redis list |
| `worker` | Reads jobs from Redis list, executes code in warm Docker containers, writes results back to Redis |

These two binaries communicate exclusively through **Redis** — there is no direct HTTP or gRPC link between the API and worker processes.

```
HTTP Client
    |
    v
API Server (cmd/api) <-- Redis --> Worker Server (cmd/worker)
    |                                     |
    | Redis Store (job metadata)          | Warm Container Pool (Docker)
    |                                     |
    +------------- Redis ----------------+
                   ^
                   | Shared queue + store
```

Supporting infrastructure (from `docker-compose.yml`):
- **Redis 7** — job queue + result store
- **Loki + Promtail + Grafana** — log aggregation and dashboards
- **Prometheus** — metrics collection

---

## Job Lifecycle (Verified)

Based on reading `handler.go`, `queue.go`, `worker.go`, `store/redis_store.go`:

1. **Client** sends `POST /judex/run` with `{language, code}`.
2. **SubmitHandler** generates a UUID job ID, creates a `models.Job` struct with `status: "pending"`.
3. **RedisStore.Add()** serializes job to JSON and stores it in Redis with key `job:<id>` and 24-hour TTL.
4. **Queue.TryPush()** checks current queue length (pending + processing) against capacity. If not full, it serializes the job again and pushes JSON onto the Redis list `pending_jobs` using `LPush`.
5. **Worker** calls `Queue.Claim()` which uses Redis `BLMove` (blocking list move) to atomically move a job from `pending_jobs` to `processing_jobs`. This is a reliable queue pattern.
6. **Worker.Process()** marks job `status: "running"`, saves to Redis, then acquires a warm Docker container from `PoolManager`.
7. **Executor** (e.g. `CppExecutor`) runs compile + run commands inside the container via Docker exec API.
8. **Result** is written back to the job struct and saved to Redis via `Store.Update()`.
9. **Queue.Ack()** removes job from `processing_jobs` list.
10. **Client** polls `GET /judex/result/<id>`, which reads from Redis store.

---

## Queue / Worker Design (Verified)

### Queue (internal/queue/queue.go)

- Backed by **Redis lists** (`pending_jobs`, `processing_jobs`)
- Uses **`BLMove`** — a blocking atomic move from pending → processing list
- Capacity is enforced by summing lengths of both lists
- Has a `StartRecovery()` goroutine that re-queues stuck jobs (those that have been in `processing_jobs` too long without completing)
- **Ack pattern**: after execution, worker calls `LRem` to remove job from `processing_jobs`

### Worker Pool (internal/worker/pool.go, worker.go)

- Worker pool is a slice of `*Worker` structs
- Each worker runs as an independent goroutine via `go worker.Start()`
- `Start()` is an infinite loop: claim → process → ack → repeat
- Workers are stateless with respect to each other; all shared state goes through Redis

---

## Execution Model (Verified)

### Warm Container Pool (internal/sandbox/pool.go)

Judex uses a **pre-warmed container pool** — a significant engineering optimization:
- On startup, `PoolManager` creates N Docker containers per language (N = worker count)
- Containers run `tail -f /dev/null` indefinitely, staying warm
- When a job arrives, worker **acquires** (borrows) a container from a channel-based pool
- After execution, container is **sanitized** (workspace wiped, user processes killed) and **returned** to the pool
- If sanitization fails, container is killed and replaced

### Code Execution (internal/sandbox/docker.go)

- **Source code is uploaded** via tar archive using Docker `CopyToContainer` API
- **Commands run** via Docker `ContainerExecCreate` + `ContainerExecAttach`
- **Stdout/stderr** captured using `stdcopy.StdCopy` (Docker's multiplexed stream)
- **Exit code** inspected via `ContainerExecInspect`

### Executor Abstraction (internal/executor/executor.go)

```go
type Executor interface {
    Execute(ctx context.Context, sb *sandbox.Sandbox) Result
}
```

Each language has its own executor struct:
- `PythonExecutor` — runs `python3 /workspace/main.py` directly
- `GoExecutor` — runs `go build` then `./app`
- `CppExecutor` — runs `g++` then compiled binary
- `JavaExecutor` — runs `javac` then `java Main`

Compile/runtime error distinction: if compile step returns non-zero exit code, returns `compile_error` status. If run step fails, returns `runtime_error`. If stderr equals "execution timeout", returns `timeout`.

---

## Docker Usage (Verified)

- **API container**: just runs the Go HTTP server, does NOT touch Docker directly
- **Worker container**: mounts `/var/run/docker.sock` to communicate with host Docker daemon (Docker-in-Docker socket pattern)
- **Sandbox images** (in `docker/`): custom minimal images per language:
  - `judex-python`: `python:3.12-slim` + writable `/workspace`
  - `judex-cpp`: `ubuntu:24.04` + `g++`
  - `judex-go`: `golang:1.25-alpine` + Go cache directory
  - `judex-java`, `judex-c`: similar
- **Security settings** (from `pool.go`):
  - `NetworkMode: "none"` — no network access
  - `CapDrop: ["ALL"]` — drop all Linux capabilities
  - `SecurityOpt: ["no-new-privileges"]` — prevents privilege escalation
  - `Memory: 256MB`, `NanoCPUs: 1e9` (1 CPU core), `PidsLimit: 64`
  - User `1000` (non-root) for exec commands

---

## Redis Usage (Verified)

Judex uses Redis for two purposes:

1. **Job Queue**: Redis lists (`pending_jobs`, `processing_jobs`) with `BLMove` for reliable dequeue
2. **Job Store**: Redis key-value (`job:<id>` → JSON blob) with 24-hour TTL

**Why Redis in Judex?** Because Judex is designed to scale horizontally — multiple API instances and multiple worker instances all need a shared queue and shared job store. Redis provides a consistent, distributed data layer.

---

## Concurrency Concepts (Verified)

From `pool.go` (sandbox):
- `sync.RWMutex` protects the `pools` map (which stores per-language container channels)
- Container pool per language is a Go `chan *WarmContainer` — channel used as a concurrent-safe pool

From `worker/pool.go`:
- Workers started with `go worker.Start()` — true goroutine concurrency

From `queue/stats.go` (inferred from usage):
- Atomic counters for submitted/completed/failed job counts

---

## Useful Ideas We Adapt

| Concept | What Judex Does | What We Adapt |
|---------|----------------|---------------|
| Executor interface | `Execute(ctx, sandbox) Result` | Same interface pattern, but our executor takes a `Job` directly |
| Two-phase execution | Compile step + run step for Go/C++ | Same approach |
| Error classification | `compile_error`, `runtime_error`, `timeout`, `success` | Same status names |
| Docker security | `NetworkMode: none`, CapDrop, PidsLimit, non-root user | Same restrictions, simpler startup |
| Worker pool | Goroutines over a slice of Worker structs | Simpler: goroutines over a buffered Go channel |
| Sandbox Dockerfiles | Minimal images per language | Same base images, simplified setup |
| Structured logging | `logutil` with key-value pairs | Similar style using `log/slog` |

---

## Ideas We Do Not Need

| Judex Feature | Why We Skip It |
|---------------|----------------|
| Redis | Single-machine project; a Go channel + sync.Map is sufficient |
| Separate API + Worker binaries | One binary is simpler and shows the architecture more clearly |
| Warm container pool | Simplifies our code; we spin up/tear down containers per job |
| Prometheus metrics | Not required for the university demo |
| Loki / Grafana / Promtail | Monitoring stack not needed for university project |
| Job recovery / re-queue mechanism | Out of scope; a simple in-memory store is sufficient |
| Rate limiter (Redis-based) | Not needed for university context |
| Middleware chain (otelhttp, etc.) | We write simple focused middleware |
| `BLMove` reliable queue | Not needed without Redis |

---

## Differences Between Judex and Our Project

| Area | Judex | Our System |
|------|-------|------------|
| Architecture | Two separate binaries (API + Worker) | Single binary (server + workers together) |
| Queue backend | Redis lists | Buffered Go channel (`chan *Job`) |
| Job store | Redis with 24h TTL | In-memory `sync.Map` |
| Container model | Pre-warmed persistent containers (pool) | Fresh container per job (simpler) |
| Concurrency mechanism | Redis `BLMove` + goroutines | Go channel + goroutines + `sync.RWMutex` |
| Monitoring | Prometheus + Grafana + Loki | Structured `log/slog` to stdout |
| Auth / Rate limit | Redis-based rate limiter | None (out of scope) |
| stdin support | Not observed in Job model | Added: `stdin` field in Job |
| Frontend | None (API-only) | Full HTML/CSS/JS frontend |
| Deployment | docker-compose multi-service | Single process + Docker for sandboxing |
| Scope | Production-grade distributed system | Educational single-machine system |

---

## Attribution

Judex is referenced purely as a technical study. No Judex source code was copied into our project. Specific design decisions inspired by Judex (e.g., executor interface, container security settings, two-phase compile+run pattern) are documented above and cited in our architecture and final report documents.

**Judex repository**: `github.com/Dharshan2208/judex`
