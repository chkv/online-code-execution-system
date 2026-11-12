package storage

import (
	"testing"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

func TestStore_SnapshotIsolation(t *testing.T) {
	store := NewStore()
	job := &models.Job{
		ID:     "snapshot-test",
		Status: models.StatusQueued,
	}

	checkStatus := func(stage string, want models.Status) {
		t.Helper()
		got, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("%s: job missing", stage)
		}
		if got.Status != want {
			t.Fatalf("%s: expected %s, got %s", stage, want, got.Status)
		}
	}

	// Changing the original after Add must not change stored data.
	store.Add(job)
	job.Status = models.StatusRunning
	checkStatus("after Add", models.StatusQueued)

	// Changing the original after Update must not change stored data.
	store.Update(job)
	job.Status = models.StatusCompleted
	checkStatus("after Update", models.StatusRunning)

	// Changing a Get result must not change stored data.
	snapshot, _ := store.Get(job.ID)
	snapshot.Status = models.StatusFailed
	checkStatus("after modifying Get result", models.StatusRunning)

	// Changing a List result must not change stored data.
	list := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 job, got %d", len(list))
	}
	list[0].Status = models.StatusTimeout
	checkStatus("after modifying List result", models.StatusRunning)
}
