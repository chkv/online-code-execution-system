# Architecture

## 1. Overview
The Online Code Execution System is a concurrent job-processing application built in Go. It accepts source code submissions via an HTTP API, queues them, executes them in isolated Docker containers using a pool of worker goroutines, and returns the results.

## 2. System Architecture
```mermaid
graph TD
    Client[Web Frontend / API Client] -->|HTTP POST /api/jobs| APIServer(Go HTTP Server)
    APIServer -->|Job ID| Client
    
    APIServer -->|Save Job| Store[(In-Memory Job Store)]
    APIServer -->|Enqueue Job| Queue([Job Queue - Go Channel])
    
    subgraph Worker Pool
        W1[Worker 1]
        W2[Worker 2]
        W3[Worker 3]
        W4[Worker 4]
    end
    
    Queue -->|Dequeue Job| W1
    Queue -->|Dequeue Job| W2
    Queue -->|Dequeue Job| W3
    Queue -->|Dequeue Job| W4
    
    W1 -->|Execute| Docker1[Docker Sandbox]
    W2 -->|Execute| Docker2[Docker Sandbox]
    W3 -->|Execute| Docker3[Docker Sandbox]
    W4 -->|Execute| Docker4[Docker Sandbox]
    
    Docker1 -->|Result| W1
    Docker2 -->|Result| W2
    Docker3 -->|Result| W3
    Docker4 -->|Result| W4
    
    W1 -->|Update Status| Store
    W2 -->|Update Status| Store
    W3 -->|Update Status| Store
    W4 -->|Update Status| Store
    
    Client -->|HTTP GET /api/jobs/{id}| APIServer
    APIServer -->|Read Job| Store
```

## 3. Components

### API Server (`internal/api`)
Handles incoming HTTP requests. It validates the request, creates a new `Job` object, saves it to the `Store`, and pushes it to the `Queue`. It also serves the frontend UI and provides endpoints for clients to poll job status.

### Job Queue (`internal/jobs`)
A buffered Go channel (`chan *models.Job`). It decouples job submission from job execution. If the queue is full, the API server rejects new submissions with a `429 Too Many Requests` status, ensuring the system is not overwhelmed.

### Worker Pool (`internal/worker`)
A configurable number of worker goroutines (default 4). Each worker runs an infinite loop, pulling jobs from the queue channel. When a job is received, the worker determines the required language executor, runs the code, and updates the `Store` with the result.

### Job Store (`internal/storage`)
An in-memory map protected by a `sync.RWMutex`. It stores the state of all jobs. The mutex prevents data races when workers update a job's status concurrently with API handlers reading the status for client polling.

### Executor & Docker Sandbox (`internal/executor`)
The executor interface defines how code is run. For each job, the system creates a new, tightly restricted Docker container (no network, limited CPU/memory, dropped capabilities). The source code is copied into the container via a tar stream, compiled (if necessary), and executed. Output and exit codes are captured and returned.

## 4. Job Lifecycle
1. **QUEUED**: Job is created, saved to the store, and pushed to the channel.
2. **RUNNING**: A worker pulls the job from the channel and begins execution.
3. **COMPLETED / COMPILE_ERROR / RUNTIME_ERROR / TIMEOUT / FAILED**: The execution finishes, the container is destroyed, and the final state is saved to the store.
