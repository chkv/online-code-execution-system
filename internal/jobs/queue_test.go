package jobs

import (
	"sync"
	"testing"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

func TestQueue_TryEnqueueAndDequeue(t *testing.T) {
	q := NewQueue(2)
	job1 := &models.Job{ID: "1"}
	job2 := &models.Job{ID: "2"}
	job3 := &models.Job{ID: "3"}

	if !q.TryEnqueue(job1) {
		t.Errorf("expected TryEnqueue to succeed")
	}
	if !q.TryEnqueue(job2) {
		t.Errorf("expected TryEnqueue to succeed")
	}
	if q.TryEnqueue(job3) {
		t.Errorf("expected TryEnqueue to fail because queue is full")
	}

	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}

	out1 := q.Dequeue()
	if out1.ID != "1" {
		t.Errorf("expected job 1, got %s", out1.ID)
	}
}

func TestQueue_Concurrency(t *testing.T) {
	q := NewQueue(100)
	var wg sync.WaitGroup

	// Multiple producers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			q.TryEnqueue(&models.Job{ID: "job"})
		}(i)
	}

	// Multiple consumers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Use select with timeout to avoid blocking forever if something goes wrong
			select {
			case <-q.Chan():
				// Got job
			case <-time.After(1 * time.Second):
				t.Errorf("timeout waiting for job")
			}
		}()
	}

	wg.Wait()
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got %d", q.Len())
	}
}
