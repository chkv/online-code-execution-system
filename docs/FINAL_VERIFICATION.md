# Final Verification

Updated: 2026-11-13. Results below refer to the recorded local runs.

## Queue saturation

The isolated test server used port 8081, one worker, one waiting slot, and a 10-second execution deadline. The first job was observed RUNNING and the second QUEUED. The third submission received HTTP 429. Both accepted jobs completed with exit code 0 and the expected output. This verifies rejection in the one-slot test configuration; the normal configuration has four workers and 100 waiting slots.

## Sandbox observations

An API-submitted Python job reported real/effective UID 1000/1000, empty effective/permitted/bounding capability sets, NoNewPrivs=1, and Seccomp=2. Its kernel cgroup settings were 134,217,728 bytes of memory, 0.5 CPU cores, and 32 processes/threads. No non-loopback interface was administratively UP.

The original probe incorrectly required the interface list to contain only lo. Linux also exposed inactive tunnel interfaces, so the corrected check reads interface flags. The earlier report also revealed disabled seccomp. Docker Desktop was then explicitly configured with `"seccomp-profile": "builtin"`, and the revised probe passed all ten checks. Earlier failed reports are retained as historical evidence.

These are observations of a Python job and configured kernel limits. They are not resource-exhaustion tests or proof against sandbox escapes. Other language paths were checked for successful execution after the daemon configuration changed.

## Four-language check after enabling seccomp

| Language | Status | Exit code | Worker execution time (ms) |
|---|---|---:|---:|
| python | COMPLETED | 0 | 1314 |
| go | COMPLETED | 0 | 4984 |
| cpp | COMPLETED | 0 | 3188 |
| java | COMPLETED | 0 | 5904 |

Each job also produced the exact expected greeting. Jobs were submitted sequentially to the verification server.

## Load tests

| Jobs | Successful | Batch seconds | Jobs/second | Peak overlapping worker jobs | Mean queue wait (s) | P95 job latency (s) |
|---:|---:|---:|---:|---:|---:|---:|
| 10 | 10 | 13.197 | 0.758 | 4 | 3.626 | 12.882 |
| 25 | 25 | 25.556 | 0.978 | 4 | 10.895 | 24.940 |
| 50 | 50 | 54.757 | 0.913 | 4 | 25.826 | 52.507 |

The 85 load-test jobs were Python programs that printed, slept for two seconds, and printed again. The backend ran with Go's race detector, four workers, and a queue capacity of 100. Each batch size was measured once. Batch time includes client submission and polling. Job latency is creation-to-finish time; P95 uses the nearest-rank method. Overlap is derived from worker start/finish timestamps, not measured CPU-core utilization. These are local workload measurements, not production capacity estimates.

## Other checks already completed

- `go test -race -timeout 60s ./...` passed for the backend changes.
- Python stdin values 5 and 3 produced a sum of 8.
- Empty Python stdin produced an EOF runtime error; invalid C++ produced a compile error.
- An infinite Python loop returned TIMEOUT and exit code 124.
- A 1,048,641-byte submission received HTTP 413.
- Oversized stdout and stderr were each retained at 1,048,576 bytes, including a truncation notice; the job completed successfully.
- Container cleanup was checked after a fresh timeout, following manual removal of an older stale container. An automated test also covers cleanup after a container-start failure.
- The UI displayed the configured worker count and execution timeout.

## Evidence files

- [queue-full-20261113-000402-020132.json](verification/queue-full-20261113-000402-020132.json)
- [sandbox-20261113-004412-760864.json](verification/sandbox-20261113-004412-760864.json)
- [languages-20261113-004953-798008.json](verification/languages-20261113-004953-798008.json)
- [load-10-20261112-113522-188051.json](load-tests/load-10-20261112-113522-188051.json)
- [load-25-20261112-113641-799346.json](load-tests/load-25-20261112-113641-799346.json)
- [load-50-20261112-113833-329229.json](load-tests/load-50-20261112-113833-329229.json)

## Completion boundary

The implemented application and recorded verification are complete for this project scope. The professor's final submission format, required presentation duration, and upload requirements must still be checked against the final assignment instructions.
