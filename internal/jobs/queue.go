// Package jobs implements the job queue using a Go buffered channel.
//
// Why a channel instead of Redis?
// Judex uses Redis lists as its queue because it scales across multiple
// separate processes. Our project is a single Go process, so a buffered
// channel is the idiomatic Go way to implement a producer-consumer queue.
//
// How it works:
//   - The HTTP handler (producer) sends job pointers into the channel.
//   - Worker goroutines (consumers) receive from the channel.
//   - Go guarantees that each send is received by exactly one receiver.
//   - No explicit mutex is needed for the queue itself — channels are
//     intrinsically safe for concurrent use.
//
// Queue full behavior:
//   - The channel is buffered with capacity = QueueSize (from config).
//   - If the channel is full, TryEnqueue returns false immediately.
//   - The HTTP handler then returns 429 Too Many Requests to the client.
//   - Jobs are NEVER silently dropped.
package jobs

import (
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// Queue is a buffered channel that holds pointers to Job structs.
// Pointers are used so workers can update the same Job object that
// the store holds — without needing to re-lookup by ID.
type Queue struct {
	ch chan *models.Job
}

// NewQueue creates a Queue with the given buffer capacity.
// capacity = max jobs that can be waiting for a worker.
func NewQueue(capacity int) *Queue {
	return &Queue{
		ch: make(chan *models.Job, capacity),
	}
}

// TryEnqueue attempts to add a job to the queue without blocking.
// Returns true if the job was accepted, false if the queue is full.
func (q *Queue) TryEnqueue(job *models.Job) bool {
	select {
	case q.ch <- job:
		return true
	default:
		// Channel is full — reject the job immediately.
		return false
	}
}

// Dequeue blocks until a job is available and returns it.
// This is called in a loop by each worker goroutine.
func (q *Queue) Dequeue() *models.Job {
	return <-q.ch
}

// Chan returns the underlying channel, used to implement graceful shutdown
// (ranging over the channel until it is closed).
func (q *Queue) Chan() <-chan *models.Job {
	return q.ch
}

// Close signals workers to stop after draining remaining jobs.
// Must only be called once (typically on server shutdown).
func (q *Queue) Close() {
	close(q.ch)
}

// Len returns the current number of jobs waiting in the queue.
func (q *Queue) Len() int {
	return len(q.ch)
}

// Cap returns the maximum queue capacity.
func (q *Queue) Cap() int {
	return cap(q.ch)
}
