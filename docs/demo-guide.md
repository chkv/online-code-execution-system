# Demo Guide

This guide will walk you through setting up and demonstrating the system on presentation day.

## 1. Setup (Before Presentation)
1. Open Docker Desktop and ensure it is running.
2. Open a terminal and navigate to the project directory:
   ```bash
   cd /path/to/online-code-execution-system
   ```
3. Start the backend server:
   ```bash
   go run cmd/server/main.go
   ```
   *You should see logs indicating the worker pool has started and the server is listening on port 8080.*
4. Open your web browser and navigate to `http://localhost:8080`.

## 2. Basic Code Execution Demo
1. In the UI, select **Python** from the language dropdown.
2. Click the **Example** button to load the Python example.
3. Click **Run Code**.
4. Show the audience the progress bar, and then point out the `stdout` tab showing the output.
5. Highlight the **Execution Time** and **Exit Code** on the right panel.

## 3. Concurrency Demo (The "Wow" Factor)
This demo proves that the system handles multiple users simultaneously without blocking.
1. Open a *second* terminal window alongside the first one (where the server is running).
2. Run the concurrency test script:
   ```bash
   ./scripts/demo_concurrency.sh
   ```
3. Immediately look at the server logs in the first terminal.
4. **Point this out to the audience:**
   - "Notice how 5 jobs were submitted at the exact same time."
   - "Look at the logs: `[Worker 1]`, `[Worker 2]`, `[Worker 3]`, and `[Worker 4]` all instantly picked up a job."
   - "Because we only have 4 workers, Job 5 waited in the Go channel."
   - "As soon as Worker 1 finished, it immediately picked up Job 5 from the queue."

## 4. Security & Error Handling Demo
1. Switch the language to **C++**.
2. Type in a broken program (e.g., missing a semicolon).
3. Click **Run Code**.
4. Show how the UI gracefully handles the error, turning the status badge red (`COMPILE ERROR`) and displaying the exact compiler error in the `stderr` tab.
5. Switch the language to **Python**.
6. Type an infinite loop:
   ```python
   while True:
       pass
   ```
7. Click **Run Code**.
8. Wait exactly 10 seconds.
9. Show how the system kills the container, returning a `TIMEOUT` status, proving that malicious code cannot freeze the server.
