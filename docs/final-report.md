# Final Project Report: Online Code Execution System

## 1. Executive Summary
This report details the design and implementation of the "Online Code Execution System" university project. The system successfully demonstrates a scalable, concurrent backend architecture capable of receiving, queuing, and securely executing user-submitted code in multiple languages.

## 2. Project Requirements Met
- **Submit Programs:** Users can submit code via a modern web interface.
- **Select Language:** Support for Python, Go, C++, and Java.
- **Real-time Status:** Asynchronous polling allows users to see queue and execution status.
- **Security:** Code runs inside isolated Docker containers (no network, limited CPU/RAM, dropped capabilities).
- **Concurrency:** The system handles multiple simultaneous submissions efficiently using a worker pool pattern.
- **Error Handling:** Gracefully handles and reports compilation errors, runtime crashes, and execution timeouts.

## 3. Technical Implementation
- **Language:** Go (Golang)
- **Concurrency Model:** Goroutines (Worker Pool), Channels (Job Queue), `sync.RWMutex` (State Storage).
- **Sandboxing:** Docker Engine SDK for Go.
- **Frontend:** HTML5, CSS3 (Vanilla), Vanilla JavaScript.

## 4. Architectural Decisions
We analyzed a production-grade open-source system (Judex) and adapted its concepts for an academic environment:
1. **Single Binary:** Instead of a complex distributed microservices architecture, we unified the API and Worker pool into a single Go application. This makes the system understandable and easy to demonstrate.
2. **Channel-based Queue:** Instead of using Redis for the job queue, we leveraged Go's native buffered channels. This perfectly demonstrates concurrent programming concepts taught in computer science courses.
3. **Ephemeral Containers:** Instead of managing a complex pool of pre-warmed containers, we create and destroy a fresh Docker container for every job. While slightly slower, it guarantees a clean, secure environment and simplifies the codebase.

## 5. Security Model
Executing untrusted code is inherently dangerous. We implemented a strict security profile:
- `NetworkMode: "none"`: The code cannot access the internet.
- `CapDrop: ["ALL"]`: The container runs with zero Linux capabilities (e.g., cannot mount file systems or change network configs).
- `SecurityOpt: ["no-new-privileges"]`: Prevents privilege escalation.
- `Memory`: Capped at 128MB.
- `NanoCPUs`: Capped at 0.5 cores.
- `User`: The code executes as an unprivileged user (UID 1000/1001), not as root.
- `Timeout`: Hard limit of 10 seconds per job.

## 6. Challenges and Learnings
- **Concurrency Bugs:** Early development highlighted the need for careful synchronization when reading and writing to the job map. Implementing `sync.RWMutex` and validating with the Go Race Detector solved these issues.
- **Docker Integration:** Interacting with Docker programmatically required managing tar streams to inject code into containers and multiplexing stdout/stderr streams to capture output safely.

## 7. Conclusion
The Online Code Execution System is a complete, robust demonstration of backend engineering principles. It successfully combines web APIs, concurrent processing, and containerized sandboxing into a cohesive application.
