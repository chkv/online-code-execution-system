# Judex vs Our System

This document compares the architecture and design of our reference project (Judex) against our implementation, highlighting intentional simplifications made for educational purposes.

| Area | Judex | Our System | Reason |
|------|-------|------------|--------|
| **Architecture** | Distributed; two separate binaries (`api` and `worker`). | Single unified binary. | A single binary is easier to understand, run, and deploy for a university project, while still demonstrating concurrency. |
| **Job Queue** | Redis lists (`pending_jobs`, `processing_jobs`) with `BLMove`. | Go buffered channel (`chan *models.Job`). | Redis is required for inter-process communication in Judex. Our single-process app can use Go's native, highly efficient channels. |
| **Worker Pool** | Goroutines pulling from Redis. | Goroutines pulling from the Go channel. | Same concept, simplified backend. |
| **Job Storage** | Redis Key-Value store with JSON serialization and 24h TTL. | In-memory `map` protected by `sync.RWMutex`. | Removes external dependencies. The mutex demonstrates manual synchronization concepts required for the course. |
| **Container Model** | Pre-warmed container pool per language; sanitized and reused. | Fresh container created and destroyed per job. | Pre-warming is a complex optimization for high-throughput production systems. Fresh containers guarantee clean state and simplify the codebase significantly. |
| **Execution** | Uses Docker SDK `ExecCreate` / `ExecAttach`. | Uses Docker SDK `ExecCreate` / `ExecAttach`. | Adapted from Judex to provide proper sandboxing and output capture. |
| **Security** | Drops capabilities, disables network, limits resources. | Identical restrictions. | Security is critical; we kept Judex's robust container isolation settings. |
| **Monitoring** | Prometheus metrics, Loki log aggregation, Grafana. | Structured logging via `log/slog`. | Advanced observability stack is out of scope. Structured logging provides sufficient visibility for demonstrations. |
| **Frontend** | API only (no frontend included). | Complete HTML/CSS/JS frontend. | Required to make the project usable and demonstrable during a presentation. |
