// Package storage provides an in-memory job store.
package storage

import (
	"sync"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// Store keeps job snapshots protected by a mutex.
type Store struct {
	mu   sync.RWMutex
	jobs map[string]models.Job
}

func NewStore() *Store {
	return &Store{
		jobs: make(map[string]models.Job),
	}
}

// Add stores a copy, so later changes to the original job
// do not modify the stored snapshot.
func (s *Store) Add(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = *job
}

// Get returns a separate copy for the caller.
func (s *Store) Get(id string) (*models.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}

	return &job, true
}

// Update replaces the stored snapshot with a copy.
func (s *Store) Update(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = *job
}

// List returns independent copies of all stored jobs.
func (s *Store) List() []*models.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		snapshot := job
		result = append(result, &snapshot)
	}

	return result
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.jobs)
}
