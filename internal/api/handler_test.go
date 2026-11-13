package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/executor"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/jobs"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/storage"
)

// mockExecutor implements executor.Executor for testing
type mockExecutor struct{}

func (m *mockExecutor) Execute(ctx context.Context, job *models.Job, timeout time.Duration) executor.Result {
	return executor.Result{Status: models.StatusCompleted}
}
func (m *mockExecutor) SourceFilename() string { return "main.py" }
func (m *mockExecutor) DockerImage() string    { return "mock" }

func setupAPI() (*Handler, *http.ServeMux) {
	store := storage.NewStore()
	queue := jobs.NewQueue(10)
	reg := make(executor.Registry)
	// Just add a dummy executor to pass validation
	reg["python"] = &mockExecutor{}

	handler := NewHandler(store, queue, reg, 4, 10*time.Second)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, http.NotFoundHandler())
	return handler, mux
}

func TestSubmitJob_Success(t *testing.T) {
	_, mux := setupAPI()

	reqBody := models.SubmitRequest{
		Language: "python",
		Code:     "print('hello')",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var res models.SubmitResponse
	json.NewDecoder(w.Body).Decode(&res)
	if res.ID == "" {
		t.Errorf("expected non-empty job ID")
	}
	if res.Status != models.StatusQueued {
		t.Errorf("expected status QUEUED, got %s", res.Status)
	}
}

func TestSubmitJob_InvalidLanguage(t *testing.T) {
	_, mux := setupAPI()

	reqBody := models.SubmitRequest{
		Language: "rust",
		Code:     "fn main() {}",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
