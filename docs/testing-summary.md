# Verified Testing Summary

Load-test recordings: 2026-11-12.

## Automated Checks

```bash
go test -race -timeout 60s ./...
```

The existing test suite passed with the race detector. Checks include job-store
snapshot isolation, concurrent Docker-client initialization, cleanup after
container-start failures, setup timeouts, request-body size boundaries, and
bounded output capture. Output tests also check that stderr remains readable
after stdout exceeds its capture limit.

## Observed Live Checks

| Check | Observed result |
| --- | --- |
| Python, Go, C++, and Java examples | Completed successfully |
| Python stdin containing 5 and 3 | Printed a sum of 8 |
| Python input request with empty stdin | Runtime error, exit 1, with EOFError |
| C++ missing semicolon | Compile error, exit 1, with compiler diagnostics |
| Python infinite loop | Timeout, exit 124, at approximately 10.16 seconds |
| Fresh Python timeout cleanup | No matching containers remained after the fresh test; an earlier stale container had been removed manually |
| Oversized JSON request: 1,048,641 bytes | HTTP 413 with the size-limit error |
| Python output exceeding both stream limits | Completed, exit 0; stdout and stderr each contained 1,048,576 bytes and a truncation notice |

The Go example completed after rebuilding its image with a warmed standard
library cache and limited compiler parallelism.

## Load-Test Method

- Four workers, queue capacity 100, and a 10-second shared execution deadline.
- The backend was running with the Go race detector enabled.
- Each job ran Python in its own Docker container, printed, slept for two
  seconds, and printed again.
- Each batch size was run once; all 85 jobs returned the expected output.
- These measurements describe this workload on the local Docker environment.

| Jobs | Successful | Batch time (s) | Successful jobs/s | Peak worker jobs |
| ---: | ---: | ---: | ---: | ---: |
| 10 | 10 | 13.197 | 0.758 | 4 |
| 25 | 25 | 25.556 | 0.978 | 4 |
| 50 | 50 | 54.757 | 0.913 | 4 |

| Jobs | Mean queue wait (s) | Mean worker time (s) | P95 job latency (s) |
| ---: | ---: | ---: | ---: |
| 10 | 3.626 | 4.472 | 12.882 |
| 25 | 10.895 | 3.958 | 24.940 |
| 50 | 25.826 | 4.271 | 52.507 |

## Timing Definitions

- **Batch time:** client time from the start of submission until all completions
  were observed, including HTTP and polling overhead.
- **Queue wait:** server `created_at` to `started_at`.
- **Worker time:** server `started_at` to `finished_at`, including container setup,
  execution, and cleanup.
- **Job latency:** queue wait plus worker time. P95 uses the nearest-rank method.
- **Peak worker jobs:** maximum overlap of server start/finish intervals.

Queue waiting explains why job latency can exceed ten seconds even when
execution completes successfully within its configured budget.

## Raw Results

- [load-10-20261112-113522-188051.json](load-tests/load-10-20261112-113522-188051.json)
- [load-25-20261112-113641-799346.json](load-tests/load-25-20261112-113641-799346.json)
- [load-50-20261112-113833-329229.json](load-tests/load-50-20261112-113833-329229.json)

## Final verification evidence

The isolated queue test, corrected runtime sandbox probe, and post-seccomp four-language check passed. Their scope and raw evidence are in [Final verification](FINAL_VERIFICATION.md).
