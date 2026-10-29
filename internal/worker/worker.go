// Package worker implements the concurrent worker pool.
//
// Architecture:
//   - A Pool holds N Worker goroutines (N = config.WorkerCount).
//   - Each Worker runs an infinite loop: receive job → execute → save result.
//   - Workers read jobs from a shared buffered channel (the job queue).
//   - Because Go channels are safe for concurrent access, N workers can all
//     call queue.Dequeue() simultaneously without any explicit lock.
//   - Each worker writes results back to the Store using its thread-safe methods.
//
// Why multiple workers?
//   Imagine 4 Python jobs are submitted simultaneously. With 1 worker they
//   would execute one-at-a-time (total time ≈ 4 × job time). With 4 workers
//   they run in parallel (total time ≈ 1 × job time).
//
// Concurrency without data races:
//   - The queue channel is intrinsically safe.
//   - The store uses sync.RWMutex (see storage package).
//   - Each worker operates on its own local variables during execution.
//   - The only "shared" thing between workers is the queue (channel) and
//     store, both of which are already synchronized.

package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/executor"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/jobs"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/storage"
)

// Worker is a goroutine that continuously picks jobs from the queue and executes them.
type Worker struct {
	id       int
	queue    *jobs.Queue
	store    *storage.Store
	registry executor.Registry
	timeout  time.Duration
}

// newWorker creates a Worker (not exported; use Pool to create workers).
func newWorker(id int, q *jobs.Queue, s *storage.Store, reg executor.Registry, timeout time.Duration) *Worker {
	return &Worker{
		id:       id,
		queue:    q,
		store:    s,
		registry: reg,
		timeout:  timeout,
	}
}

// start runs the worker loop. It should be called as a goroutine.
// The worker reads from the queue channel until it is closed (shutdown signal).
func (w *Worker) start(wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Info(fmt.Sprintf("[Worker %d] started", w.id))

	// Range over the channel: blocks until a job is available, exits when channel is closed.
	for job := range w.queue.Chan() {
		w.processJob(job)
	}

	slog.Info(fmt.Sprintf("[Worker %d] stopped", w.id))
}

// processJob executes a single job end-to-end:
//  1. Mark RUNNING in the store.
//  2. Look up the language executor.
//  3. Run code in Docker.
//  4. Save results and mark final status.
func (w *Worker) processJob(job *models.Job) {
	start := time.Now()
	slog.Info(fmt.Sprintf("[Worker %d] picked job %s (%s)", w.id, job.ID[:8], job.Language))

	// --- Step 1: Mark the job as RUNNING ---
	job.Status = models.StatusRunning
	job.StartedAt = time.Now()
	w.store.Update(job)

	// --- Step 2: Find the executor for this language ---
	exec, ok := w.registry.Get(job.Language)
	if !ok {
		slog.Warn(fmt.Sprintf("[Worker %d] unsupported language: %s", w.id, job.Language),
			"job_id", job.ID)
		job.Status = models.StatusFailed
		job.ErrorMessage = fmt.Sprintf("unsupported language: %s", job.Language)
		job.FinishedAt = time.Now()
		w.store.Update(job)
		return
	}

	// --- Step 3: Execute the code inside Docker ---
	// We pass a background context; the executor creates its own timeout context internally.
	ctx := context.Background()
	result := exec.Execute(ctx, job, w.timeout)

	// --- Step 4: Save results back to the job ---
	job.Stdout = result.Stdout
	job.Stderr = result.Stderr
	job.ExitCode = result.ExitCode
	job.Status = result.Status
	job.ErrorMessage = result.ErrorMessage
	job.ExecutionTimeMs = result.ExecutionTimeMs
	job.FinishedAt = time.Now()
	w.store.Update(job)

	elapsed := time.Since(start).Milliseconds()
	slog.Info(fmt.Sprintf("[Worker %d] finished job %s | status=%s | exec=%dms | total=%dms",
		w.id, job.ID[:8], job.Status, result.ExecutionTimeMs, elapsed))
}

// -----------------------------------------------------------------------
// Pool
// -----------------------------------------------------------------------

// Pool manages a set of Worker goroutines.
type Pool struct {
	workers []*Worker
	wg      sync.WaitGroup
}

// NewPool creates N workers and returns the Pool.
// Workers are not started until Start() is called.
func NewPool(count int, q *jobs.Queue, s *storage.Store, reg executor.Registry, timeout time.Duration) *Pool {
	slog.Info("worker pool: creating", "count", count)
	p := &Pool{}
	for i := 1; i <= count; i++ {
		p.workers = append(p.workers, newWorker(i, q, s, reg, timeout))
	}
	return p
}

// Start launches all workers as goroutines.
// Each goroutine reads from the shared job queue channel.
// This is where real concurrency begins — all N workers run simultaneously.
func (p *Pool) Start() {
	for _, w := range p.workers {
		p.wg.Add(1)
		go w.start(&p.wg) // goroutine: each worker runs independently
	}
	slog.Info("worker pool: all workers started", "count", len(p.workers))
}

// Stop waits for all workers to finish processing their current job
// after the queue channel is closed.
func (p *Pool) Stop() {
	p.wg.Wait()
	slog.Info("worker pool: all workers stopped")
}

// WorkerCount returns how many workers are in the pool.
func (p *Pool) WorkerCount() int {
	return len(p.workers)
}
