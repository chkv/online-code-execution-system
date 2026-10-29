# Online Code Execution System

A concurrent, scalable, and secure online code execution backend and web interface. Built for a university software engineering project.

## Features
- **Multi-language Support:** Python, Go, C++, Java.
- **Concurrent Processing:** Uses a Go worker pool and buffered channels to process multiple jobs simultaneously.
- **Secure Sandboxing:** Executes untrusted code inside isolated, resource-constrained Docker containers.
- **Sleek UI:** A responsive, dark-mode web interface for writing code and viewing results.
- **Robust Error Handling:** Detects and reports compilation errors, runtime crashes, and execution timeouts.

## Getting Started

### Prerequisites
- Go 1.21+
- Docker (must be running)

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
go run cmd/server/main.go
```
The server will start on `http://localhost:8080`.

### 3. Open the UI
Navigate to `http://localhost:8080` in your web browser.

## Testing
Run the unit tests (including the Go Race Detector to verify concurrent safety):
```bash
go test -race ./...
```

To see concurrency in action, start the server and run the demo script in a separate terminal:
```bash
./scripts/demo_concurrency.sh
```

## Documentation
All project documentation is located in the `docs/` directory:
- [Architecture](docs/architecture.md)
- [Concurrency Design](docs/concurrency.md)
- [API Reference](docs/api.md)
- [Testing Strategy](docs/testing.md)
- [Final Report](docs/final-report.md)
- [Student Explanation Guide](docs/student-explanation-guide.md)
