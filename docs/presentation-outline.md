# Presentation Outline

Adjust the length to the professor's final instructions. Suggested duration: 8–10 minutes, excluding questions.

1. **Problem and goal:** Run programs in four languages and return useful results; introduce the team and local Go/Docker setup.
2. **Live workflow:** Submit Python with stdin, show the result, and explain queued/running/terminal status polling.
3. **Architecture:** Browser, HTTP handlers, buffered channel, worker goroutines, snapshot store, and per-job Docker containers.
4. **Concurrency:** Explain channel ownership, independent job snapshots, RWMutex, and the race detector. Demonstrate overlapping worker jobs.
5. **Execution controls:** Explain the shared deadline, container restrictions, syscall filtering, and input/output size limits. Show one error or timeout.
6. **Measured results:** Present the 10/25/50-job table from final-report.md, the 85/85 successful jobs, the observed peak of four overlapping worker jobs, and the final four-language check.
7. **Limitations and learning:** Explain in-memory history, local environment dependence, and the limited scope of security testing. Invite questions.

Suggested speaking split: introduction/demo; concurrency; Docker/limits; results/limitations. Agree on the actual speakers as a team rather than treating this as a record of individual contributions.
