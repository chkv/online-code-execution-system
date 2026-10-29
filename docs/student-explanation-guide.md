# Student Explanation Guide

This guide is designed to help you (the student) confidently explain every major part of the code during your presentation or when answering questions from the professor.

---

## 1. How does the Queue work? (`internal/jobs/queue.go`)
**Concept:** Producer-Consumer pattern.
- **Explanation:** We use a Go "buffered channel" as our queue. The HTTP server (Producer) receives a job and sends it into the channel. The Workers (Consumers) sit in a loop pulling jobs out of the channel.
- **Why it's safe:** Go channels are thread-safe by default. We don't need to write manual locks (mutexes) for the queue. Go ensures that if 4 workers are waiting, only 1 worker gets the job.
- **What happens if it's full:** Because it's a *buffered* channel (e.g., size 100), if 100 jobs are waiting, the channel is full. The API uses a non-blocking `select` (`TryEnqueue`). If it's full, it immediately returns false, and we tell the user "Too Many Requests" instead of crashing.

## 2. How does the Worker Pool work? (`internal/worker/worker.go`)
**Concept:** Concurrency / Goroutines.
- **Explanation:** In `main.go`, we read `WorkerCount` (e.g., 4) from the config. The `Pool.Start()` method runs a `for` loop that spawns 4 goroutines (`go w.start()`). 
- **How they run:** All 4 goroutines run at the same time. They all call `queue.Dequeue()` which reads from the shared channel. If a job comes in, one worker wakes up, processes it, and goes back to waiting.

## 3. How do you prevent Data Races? (`internal/storage/store.go`)
**Concept:** Mutual Exclusion (Mutex).
- **Explanation:** We store the status of all jobs in a standard Go `map`. However, Go maps are NOT thread-safe. A worker might be updating a job to "COMPLETED" at the exact millisecond the HTTP API is reading the job to show the user the status. This is a data race.
- **The Fix:** We use `sync.RWMutex`. 
  - When the API reads a job, it calls `RLock()` (Read Lock). Multiple APIs can read at the same time.
  - When a worker updates a job, it calls `Lock()` (Write Lock). This waits until no one is reading, locks it exclusively, writes the update, and unlocks it.

## 4. How do you execute the code securely? (`internal/executor/docker.go`)
**Concept:** Containerization & Sandboxing.
- **Explanation:** We don't run the code directly on our server. We use the Docker SDK for Go to spin up a tiny virtual environment (a container) just for that job.
- **Security:** We pass specific flags to Docker:
  - `NetworkMode: "none"` so the code can't hack our network or download malware.
  - `Memory: 128MB` so an infinite loop can't crash our server by eating all RAM.
  - We run as a non-root user (UID 1000/1001) inside the container.
- **Cleanup:** We use `AutoRemove: true` so the container deletes itself the moment the code finishes running.

## 5. How do you handle timeouts? (`internal/executor/docker.go`)
**Concept:** Contexts & Timeouts.
- **Explanation:** Some student might submit `while(true) {}`. To prevent workers from getting stuck forever, we use `context.WithTimeout(ctx, 10*time.Second)`.
- **How it works:** If the command runs longer than 10 seconds, the context "cancels". Our code detects this cancellation, sends a `SIGKILL` to the Docker container, and marks the job status as `TIMEOUT`.

## 6. Why didn't you use Microservices or Redis (like Judex)?
**Concept:** Architecture trade-offs.
- **Explanation:** Judex is built for massive scale across multiple servers, requiring a central Redis queue and separate binaries. For this university project, a monolithic single-process architecture is better. It demonstrates the exact same concurrency concepts (queues, worker pools, synchronization) using Go's built-in tools (channels and mutexes) without the operational overhead of managing external databases.
