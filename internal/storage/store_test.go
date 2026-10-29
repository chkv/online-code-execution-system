package storage

import (
	"sync"
	"testing"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

func TestStore_AddAndGet(t *testing.T) {
	s := NewStore()
	job := &models.Job{ID: "123", Status: models.StatusQueued}

	s.Add(job)

	retrieved, ok := s.Get("123")
	if !ok {
		t.Fatalf("expected to find job 123")
	}
	if retrieved.ID != "123" {
		t.Errorf("expected ID 123, got %s", retrieved.ID)
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore()
	job := &models.Job{ID: "123", Status: models.StatusQueued}
	s.Add(job)

	job.Status = models.StatusRunning
	s.Update(job)

	retrieved, _ := s.Get("123")
	if retrieved.Status != models.StatusRunning {
		t.Errorf("expected status RUNNING, got %s", retrieved.Status)
	}
}

func TestStore_Concurrency(t *testing.T) {
	s := NewStore()
	job := &models.Job{ID: "123", Status: models.StatusQueued}
	s.Add(job)

	var wg sync.WaitGroup
	// Concurrently read and write to the same job
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("123")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Update(&models.Job{ID: "123", Status: models.StatusCompleted})
		}()
	}
	wg.Wait()
}
