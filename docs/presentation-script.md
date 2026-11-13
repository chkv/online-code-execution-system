# Presentation Speaking Notes

## Introduction

Our project is an Online Code Execution System built with Go. A user selects Python, Go, C++, or Java, enters code and optional input, and receives the execution result. The version we are demonstrating runs locally using Docker Desktop.

## Submission and results

When we click Run, the frontend sends an HTTP request to the Go server. The server returns a job ID after accepting the job into the queue. The browser polls that ID for status. Output appears when the result is available; this is polling, not live output streaming.

## Concurrency

The normal server has four worker goroutines and a buffered channel with 100 waiting slots. Each worker takes a job, executes it, and stores its result. When the queue is full, a new submission receives HTTP 429. In our isolated one-slot test, one job ran, another waited, and the third was rejected. Both accepted jobs still finished correctly.

The job store uses an RWMutex and independent job snapshots. This matters because protecting the map alone would not prevent a caller from modifying a shared job pointer after the lock was released. We also synchronized Docker client initialization and ran the Go race tests.

## Docker and deadlines

Each job gets a fresh container. We configure no external network attachment, non-root execution through the language images, dropped capabilities, no-new-privileges, and CPU, memory, and process limits. We enabled Docker's built-in seccomp profile and checked its active mode inside a submitted Python job. These are verified restrictions, not a claim that containers cannot be escaped.

The normal ten-second deadline includes container setup, compilation, and execution. Time waiting in the queue is separate. A timeout ends the job and triggers cleanup. Oversized request bodies are rejected, and captured output is bounded so a printing loop cannot keep growing the output buffers indefinitely.

## Results and lessons

All four final language checks completed after syscall filtering was enabled. The measured Python batches of 10, 25, and 50 jobs all succeeded, for 85 successful jobs overall. The highest observed overlap was four worker jobs. These programs included a two-second sleep, so we present the numbers as measurements of that workload on this machine, not a general CPU benchmark.

One important fix was warming the Go standard-library build cache while building the image. We also learned that Docker availability and startup delays affect the shared execution deadline. The application now demonstrates the complete workflow, while persistence, user authentication, and broader security testing remain future work. We can use the saved reports to explain the results during questions.
