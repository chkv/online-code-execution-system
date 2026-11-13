# Concurrency and Synchronization

## 1. Overview
The core requirement of this project is to process multiple code execution jobs simultaneously. This is achieved using Go's built-in concurrency primitives: **goroutines**, **channels**, and **mutexes**.

## 2. Goroutines
Goroutines are lightweight threads managed by the Go runtime. 
In our system:
- The main HTTP server runs in its own goroutine.
- Each incoming HTTP request is handled in a separate goroutine automatically by `net/http`.
- **Worker Pool**: When the server starts, it spawns a configured number of worker goroutines (e.g., 4). Each worker runs an infinite loop waiting for jobs.

## 3. Channels
A channel is a conduit that allows goroutines to communicate and synchronize safely.
- **Job Queue**: We use a buffered channel (`chan *models.Job`) as our job queue.
- The API handler (producer) sends jobs into the channel.
- The worker goroutines (consumers) read from the channel.
- Channels are inherently thread-safe; Go guarantees that a job sent to the channel is received by exactly one worker, preventing race conditions during job assignment.

## 4. Synchronization (Mutex)
While channels handle the passing of jobs, we need to store the jobs so users can query their status. This is done using an in-memory map (`map[string]models.Job`).

### The Problem
If Worker 1 is updating the status of Job A to "COMPLETED" while an API handler is simultaneously reading Job A to return it to the user, a **data race** occurs. Go will crash the program if a map is read and written concurrently without synchronization.

### The Solution: `sync.RWMutex`
We protect the map using a Read-Write Mutex:
- `RLock()` (Read Lock): Allows multiple API handlers to read job statuses simultaneously without blocking each other.
- `Lock()` (Write Lock): Used by workers when updating a job. It waits for all readers to finish, locks the map exclusively, performs the update, and unlocks it.

## 5. Concurrent Job Processing Flow
1. **Time T0**: User submits 5 jobs simultaneously.
2. **Time T1**: The API handler adds all 5 jobs to the `Store` and pushes them into the `Queue` channel.
3. **Time T2**: Workers 1, 2, 3, and 4 immediately pull the first 4 jobs from the channel and start executing them in parallel. Job 5 waits in the channel.
4. **Time T3**: Worker 2 finishes its job early. It updates the store, then pulls Job 5 from the channel and starts executing it.

By leveraging these primitives, the system efficiently utilizes system resources and processes multiple submissions without blocking.

## Snapshot ownership and measurement

Add and Update copy the submitted job into the store. Get and List return independent job snapshots. The RWMutex protects the map operations; copying prevents callers from mutating stored records outside that lock. Each worker owns its queued job object and publishes updates through the store.

Worker scheduling is nondeterministic. Examples showing particular workers picking particular jobs are illustrative. The load-test script measures overlapping start/finish intervals; this is concurrency evidence, not a measurement of physical CPU parallelism. Queue waiting is excluded from the per-job execution deadline. On shutdown, closing the channel lets workers drain queued jobs before the pool stops.
