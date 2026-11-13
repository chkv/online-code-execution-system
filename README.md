# Online Code Execution System

An online code execution system with a Go backend, a browser interface, and a configurable worker pool. Runs Python, Go, C++, and Java in Docker containers.

## Features

- **Multi-language Support:** Python, Go, C++, Java.
- **Concurrent Processing:** Uses a Go worker pool and buffered channels to process multiple jobs simultaneously.
- **Docker Isolation:** Runs submitted code with networking disabled and CPU, memory, and process limits.
- **Sleek UI:** A responsive, dark-mode web interface for writing code and viewing results.
- **Robust Error Handling:** Detects and reports compilation errors, runtime crashes, and execution timeouts.

## Getting Started

### Prerequisites

- Go 1.26.1 or newer (see `go.mod`)
- Docker (must be running)
- Python 3.11+ for the measured load-test script

### 1. Build the Sandbox Images

The system requires Docker images for each supported language. We provide a script to build them.
*(Note: Building `oces-cpp` and `oces-java` might take a few minutes depending on your internet speed).*
```bash
docker build -t oces-python docker/python/
docker build -t oces-go docker/go/
docker build -t oces-cpp docker/cpp/
docker build -t oces-java docker/java/
```

### 2. Run the Server

```bash
go run ./cmd/server
```
The server will start on `http://localhost:8080`.

### 3. Open the UI

Navigate to `http://localhost:8080` in your web browser.

## Runtime Configuration and Limits

| Environment variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `WORKER_COUNT` | `4` | Concurrent worker goroutines |
| `QUEUE_SIZE` | `100` | Waiting-job capacity |
| `EXECUTION_TIMEOUT` | `10s` | Shared container-setup, compilation, and execution deadline |

Queue waiting is excluded from the execution deadline. Container cleanup has
a separate bounded timeout, so reported worker time can exceed the deadline.

- The complete JSON submission body is limited to **1 MiB**; oversized requests return HTTP **413**.
- Captured program output is limited to **1 MiB per stdout/stderr stream per command**.
  Excess output is discarded, and a truncation notice is included within the limit.
- A program can complete successfully with truncated output.
- Containers use a 0.5-CPU limit, 128 MiB memory limit, and a PID limit of 32.
- Job history is held in memory and is cleared when the server restarts.
- The UI reads worker count and timeout from `/api/health`.
- Browser polling continues while a job is queued or running.

Additional image settings are defined in `internal/config/config.go`.

### Docker Desktop

If the Go client needs the Docker CLI's selected context, start the server with:

```bash
DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')" go run -race ./cmd/server
```

Stop the existing server with Ctrl+C before restarting it.

## Testing

Run the automated tests with the Go race detector:
```bash
go test -race -timeout 60s ./...
```

To see concurrency in action, start the server and run the demo script in a separate terminal:
```bash
bash scripts/demo_concurrency.sh
```

## Measured Load Tests

With the server running, execute these sequentially in another terminal:

```bash
python3 scripts/load_test.py 10
python3 scripts/load_test.py 25
python3 scripts/load_test.py 50
```

Each Python job prints a start message, sleeps for two seconds, and prints a
completion message. The script checks every job's status, exit code, and stdout.
It saves summary metrics and individual job results under `docs/load-tests/`.

Recorded results from 2026-11-12, using four workers and the server's race detector:

| Jobs | Successful | Batch time (s) | Successful jobs/s | Peak worker jobs |
| ---: | ---: | ---: | ---: | ---: |
| 10 | 10 | 13.197 | 0.758 | 4 |
| 25 | 25 | 25.556 | 0.978 | 4 |
| 50 | 50 | 54.757 | 0.913 | 4 |

Each batch size was run once. Batch time includes HTTP submission and polling.
See [Verified Testing Summary](docs/testing-summary.md) for timing definitions,
additional checks, and links to the raw results.

## Documentation

All project documentation is located in the `docs/` directory:
- [Architecture](docs/architecture.md)
- [Concurrency Design](docs/concurrency.md)
- [API Reference](docs/api.md)
- [Testing Strategy](docs/testing.md)
- [Verified Testing Summary](docs/testing-summary.md)
- [Final Report](docs/final-report.md)
- [Student Explanation Guide](docs/student-explanation-guide.md)

## Final verification evidence

The queue rejection, runtime sandbox, and four-language checks are recorded in [Final verification](docs/FINAL_VERIFICATION.md). See [Demo guide](docs/demo-guide.md), [Final report](docs/final-report.md), and [Presentation outline](docs/presentation-outline.md).
