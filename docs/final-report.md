# Online Code Execution System — Final Project Report

**Course:** ITCS-5102: Survey of Programming Languages

**Team:** Hruthik Narayan Sarva; Pranay Paul Yatelli; Geetha Sri Mandalapu; Keerthi Vardhan Chandana

**Verification updated:** 2026-11-13

## 1. Project goal

We built a web application where users select Python, Go, C++, or Java, enter source code and optional standard input, and receive the program's status and output. The project demonstrates Go concurrency, HTTP communication, shared-state synchronization, data structures, and error handling. It runs locally using a Go server and Docker Desktop.

## 2. Architecture and execution flow

The frontend sends a JSON submission to POST /api/jobs. The handler validates the request, creates a job record, and tries to put it into a bounded Go channel. An accepted request receives HTTP 202 and a job ID. The browser polls GET /api/jobs/{id} until a terminal status is available; output is returned with the result rather than streamed live.

Worker goroutines receive jobs from the channel. Each worker marks its job RUNNING and selects a language executor. The executor creates a fresh Docker container, uploads the source through a tar archive, compiles when needed, runs the program, captures output, and removes the container. Python runs directly; Go, C++, and Java compile before execution. Results include status, stdout, stderr, exit code, and timing information.

## 3. Concurrency and synchronization

The normal configuration has four workers and 100 waiting slots. Each channel send is received by one worker. A full queue causes an immediate HTTP 429 response. Worker assignment order is not fixed, and four active workers do not imply four physical CPU cores running at full speed.

Job records are held in an in-memory map protected by sync.RWMutex. Locking the map alone was insufficient while callers shared pointers to mutable jobs. The store now copies jobs on writes and returns independent snapshots on reads. Workers update their own job objects and publish snapshots to the store. Docker client initialization is also protected by a mutex. The race detector tests exercised these concurrency paths without reporting races; this is evidence for the tested paths, not a proof that every possible race is absent.

## 4. Execution limits and error handling

Each container is configured with networking disabled, all Linux capabilities dropped, no-new-privileges enabled, 128 MiB memory, a 0.5-core CPU quota, and a limit of 32 processes/threads. The language images declare non-root users. The final Python probe verified UID 1000 and the corresponding runtime restrictions. Docker's built-in seccomp profile was explicitly enabled on the demonstration machine and verified inside the job.

The default 10-second deadline covers container setup, compilation, and execution together. Queue waiting is excluded. Cleanup has a separate bounded context, so reported completion time can slightly exceed the execution deadline. Container removal is registered immediately after successful creation so a later startup failure also triggers cleanup.

Complete JSON submissions are limited to 1 MiB and oversized bodies receive HTTP 413. Captured stdout and stderr are each limited to 1 MiB per command, with truncation notices included within that limit. Excess output is drained and discarded, allowing the process to finish. Results distinguish successful completion, compilation errors, runtime errors, timeouts, and infrastructure failures.

## 5. Problems resolved

The original Go example exceeded the compilation deadline. We rebuilt its image with a populated standard-library build cache, disabled CGO, and limited compiler parallelism. The Go example then ran successfully within the existing deadline.

We also fixed shared job pointers, concurrent Docker client initialization, cleanup after startup failures, and timeout handling. The browser now keeps polling queued jobs and ignores stale responses after cancellation or a new submission. Worker and timeout labels come from server configuration. During final verification, an inactive tunnel interface exposed an overly strict test assumption, and the runtime probe identified a disabled Docker seccomp profile. Both were addressed and verified.

## 6. Results

After enabling the built-in seccomp profile, all four language checks completed with exit code 0 and exact expected output:

| Language | Status | Exit code | Worker execution time (ms) |
|---|---|---:|---:|
| python | COMPLETED | 0 | 1314 |
| go | COMPLETED | 0 | 4984 |
| cpp | COMPLETED | 0 | 3188 |
| java | COMPLETED | 0 | 5904 |

The measured load tests completed 85 of 85 jobs:

| Jobs | Successful | Batch seconds | Jobs/second | Peak overlapping worker jobs | Mean queue wait (s) | P95 job latency (s) |
|---:|---:|---:|---:|---:|---:|---:|
| 10 | 10 | 13.197 | 0.758 | 4 | 3.626 | 12.882 |
| 25 | 25 | 25.556 | 0.978 | 4 | 10.895 | 24.940 |
| 50 | 50 | 54.757 | 0.913 | 4 | 25.826 | 52.507 |

These batches used Python jobs containing a two-second sleep, a four-worker server, and race detection. Each batch size was run once. Increasing batch size mainly increased queue waiting. A job can wait longer than ten seconds in the queue without violating its ten-second execution deadline. Full-queue rejection was separately verified with one worker and one waiting slot: two submissions were accepted and the third was rejected with HTTP 429.

Additional checks covered stdin, compile/runtime errors, infinite-loop timeout handling, request/output limits, and selected cleanup scenarios. The complete evidence and measurement definitions are in [Final verification](FINAL_VERIFICATION.md) and [Testing summary](testing-summary.md).

## 7. Limitations and future work

This is a single-process local demonstration. Job history is in memory, is not retained across restarts, and has no automatic retention policy. Listing jobs returns up to 50 entries without a guaranteed order. The system has no user authentication or per-user rate limits. Docker availability and startup latency affect jobs; a cold Docker startup may consume the shared deadline. The sandbox checks inspect runtime settings and do not constitute an escape-resistance audit. Future work could add persistence, retention, authentication, and broader performance/security testing.

## 8. Conclusion

The application demonstrates a complete submission-to-result workflow across four languages, bounded concurrent processing, synchronization, and Docker-based execution controls. The original Go compilation issue and the identified stability problems were resolved, and the results above provide reproducible evidence for the completed scope.
