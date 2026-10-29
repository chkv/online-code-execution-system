// Package storage provides the in-memory job store.
//
// Why not Redis?
// Judex uses Redis because it runs as multiple separate processes that need
// shared state. Our project runs as a single process, so a Go-native in-memory
// store protected by a sync.RWMutex is sufficient and much simpler.
//
// Concurrency safety:
// Multiple goroutines (the HTTP handler and N worker goroutines) read and write
// jobs concurrently. Without synchronization, two goroutines could read the same
// job simultaneously while another goroutine is writing — this is a data race.
//
// We use sync.RWMutex:
//   - Multiple readers can hold the read lock simultaneously (Get calls).
//   - A write lock is exclusive (Add, Update operations).
package storage

import (
	"sync"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// Store is a thread-safe in-memory store for Job objects.
//
// Shared state: the jobs map is read by the GET /api/jobs/:id handler
// AND written by worker goroutines when they update job status/results.
// Without the mutex, these concurrent accesses would cause a race condition.
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*models.Job
}

// NewStore creates and returns an empty Store.
func NewStore() *Store {
	return &Store{
		jobs: make(map[string]*models.Job),
	}
}

// Add saves a new job.
// The write lock ensures no other goroutine can read or write during the operation.
func (s *Store) Add(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Get retrieves a job by ID.
// The read lock allows multiple concurrent reads, but blocks if a write is happening.
func (s *Store) Get(id string) (*models.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

// Update replaces an existing job's data.
// Called by workers to save execution results.
func (s *Store) Update(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// List returns a snapshot of all jobs in the store.
// We copy the slice to avoid holding the lock while the caller iterates.
func (s *Store) List() []*models.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result
}

// Count returns the total number of stored jobs.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}
