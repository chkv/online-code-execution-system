# Presentation Outline

## 1. Introduction (2 mins)
- Introduce the "Online Code Execution System"
- Explain the problem: Running untrusted code securely while supporting multiple users simultaneously.
- Mention the key technologies: Go, Docker, HTML/JS/CSS.

## 2. Live Demonstration (3 mins)
- Open the web interface.
- **Demo 1 (Basic):** Write a simple Python "Hello World" and run it. Show the fast execution.
- **Demo 2 (Concurrency):** Run the `scripts/demo_concurrency.sh` script in the terminal. Switch back to the UI or logs to show multiple workers processing the jobs at the exact same time.
- **Demo 3 (Errors & Timeouts):** Run a C++ program with a syntax error to show compiler output. Run a Python infinite loop to show the 10-second timeout enforcement.

## 3. Architecture & Concurrency (4 mins)
- Show the architecture diagram (from `docs/architecture.md`).
- Explain the **Single Binary** approach.
- **The Job Queue:** Explain how Go channels act as the queue.
- **The Worker Pool:** Explain goroutines. "We have 4 workers running in parallel, taking jobs from the channel."
- **Synchronization:** Explain the data race problem (API reading status while worker updates it) and how `sync.RWMutex` solves it. Mention that we verified this with the Go Race Detector.

## 4. Security & Sandboxing (3 mins)
- Explain why running `system("rm -rf /")` on the host server is bad.
- Explain the Docker Sandboxing approach:
  - Ephemeral containers (created and destroyed per job).
  - No network access.
  - Dropped Linux capabilities.
  - Resource limits (CPU/Memory).
  
## 5. Design Choices & Inspirations (2 mins)
- Mention "Judex", the open-source reference project.
- Explain how we simplified the architecture for this course (e.g., removing Redis in favor of Go channels, using a unified binary instead of microservices).
- Why this is a better fit for an educational demonstration.

## 6. Q&A (1 min)
- Open the floor for questions from the professor and peers.
