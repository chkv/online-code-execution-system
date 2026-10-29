# Testing Strategy

## 1. Overview
The testing strategy for the Online Code Execution System focuses on unit testing the core concurrent components and ensuring the API layer behaves correctly. Since this system relies heavily on concurrency (goroutines, channels, mutexes), a key part of our strategy is using Go's built-in Data Race Detector.

## 2. Running the Tests

To run the standard unit tests:
```bash
go test ./...
```

To run the tests with the **Race Detector** enabled (highly recommended for this project):
```bash
go test -race ./...
```

## 3. What is Tested

### Storage (`internal/storage`)
- **Unit Tests:** Verify that jobs can be added, retrieved, and updated correctly.
- **Concurrency Tests:** `TestStore_Concurrency` launches 100 goroutines that simultaneously attempt to read (`Get`) and write (`Update`) the same job. 
- **Purpose:** Proves that our `sync.RWMutex` correctly prevents map data races. If the mutex is removed, `go test -race` will immediately fail.

### Job Queue (`internal/jobs`)
- **Unit Tests:** Verify that jobs can be enqueued and dequeued, and that `TryEnqueue` correctly rejects jobs when the buffer is full.
- **Concurrency Tests:** `TestQueue_Concurrency` creates 50 producer goroutines and 50 consumer goroutines that push and pull from the queue simultaneously.
- **Purpose:** Proves that the underlying Go channel handles concurrent access safely without manual locking.

### API Layer (`internal/api`)
- **Integration Tests:** We use the `net/http/httptest` package to simulate HTTP requests without starting an actual network server.
- **Mocks:** The tests inject a mock `Executor` into the handler to avoid requiring Docker to run the API tests.
- **Validation:** Tests verify proper HTTP status codes (e.g., `202 Accepted` on success, `400 Bad Request` on invalid language).

## 4. Manual Testing & Demo
For full end-to-end testing including Docker execution, we use the provided demonstration script:
```bash
./scripts/demo_concurrency.sh
```
This script uses `curl` to fire multiple HTTP POST requests simultaneously, allowing us to visually verify in the server logs that the Worker Pool processes jobs concurrently.
