// Package api implements the HTTP handlers for the REST API.
//
// Endpoints:
//   POST   /api/jobs         — submit code for execution
//   GET    /api/jobs/{id}    — get job status and result
//   GET    /api/jobs         — list recent jobs
//   GET    /api/languages    — list supported languages
//   GET    /api/health       — server health check
//   GET    /                 — serve the frontend

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/executor"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/jobs"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/storage"
)

// Handler holds all dependencies for the API handlers.
// Passing dependencies explicitly (instead of using globals) makes
// the code testable — tests can inject mock stores and queues.
type Handler struct {
	store    *storage.Store
	queue    *jobs.Queue
	registry executor.Registry
}

// NewHandler creates an API Handler with all required dependencies.
func NewHandler(s *storage.Store, q *jobs.Queue, reg executor.Registry) *Handler {
	return &Handler{store: s, queue: q, registry: reg}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, fs http.Handler) {
	// API routes
	mux.HandleFunc("/api/jobs", h.jobsRouter)
	mux.HandleFunc("/api/jobs/", h.getJobHandler)
	mux.HandleFunc("/api/languages", h.languagesHandler)
	mux.HandleFunc("/api/health", h.healthHandler)

	// Serve frontend from web/ directory
	mux.Handle("/", fs)
}

// jobsRouter dispatches POST (submit) and GET (list) on /api/jobs
func (h *Handler) jobsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.submitJobHandler(w, r)
	case http.MethodGet:
		h.listJobsHandler(w, r)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// -----------------------------------------------------------------------
// POST /api/jobs — submit a job
// -----------------------------------------------------------------------

func (h *Handler) submitJobHandler(w http.ResponseWriter, r *http.Request) {
	// Parse JSON request body.
	var req models.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("submit: invalid JSON", "error", err)
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate language.
	if req.Language == "" {
		writeError(w, "language is required", http.StatusBadRequest)
		return
	}
	if _, ok := h.registry.Get(req.Language); !ok {
		writeError(w, "unsupported language: "+req.Language, http.StatusBadRequest)
		return
	}

	// Validate code.
	if strings.TrimSpace(req.Code) == "" {
		writeError(w, "code is required", http.StatusBadRequest)
		return
	}

	// Create a new Job.
	job := &models.Job{
		ID:        uuid.New().String(),
		Language:  req.Language,
		Code:      req.Code,
		Stdin:     req.Stdin,
		Status:    models.StatusQueued,
		CreatedAt: time.Now(),
	}

	// Save to store first, so GET /api/jobs/{id} works immediately.
	h.store.Add(job)
	slog.Info("job created", "job_id", job.ID, "language", job.Language)

	// Try to enqueue the job.
	// TryEnqueue is non-blocking — returns false if the queue is full.
	if !h.queue.TryEnqueue(job) {
		// Queue is full: remove the job from the store and tell the client.
		// Jobs are never silently lost.
		h.store.Update(&models.Job{
			ID:           job.ID,
			Status:       models.StatusFailed,
			ErrorMessage: "server is busy, please try again later",
			CreatedAt:    job.CreatedAt,
			FinishedAt:   time.Now(),
		})
		slog.Warn("queue full, job rejected", "job_id", job.ID)
		writeError(w, "queue is full, try again later", http.StatusTooManyRequests)
		return
	}

	slog.Info("job queued", "job_id", job.ID, "queue_len", h.queue.Len())

	// Respond with the job ID immediately.
	// The client should then poll GET /api/jobs/{id}.
	writeJSON(w, http.StatusAccepted, models.SubmitResponse{
		ID:     job.ID,
		Status: models.StatusQueued,
	})
}

// -----------------------------------------------------------------------
// GET /api/jobs/{id} — get job status and result
// -----------------------------------------------------------------------

func (h *Handler) getJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from URL: /api/jobs/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	if id == "" {
		writeError(w, "job ID is required", http.StatusBadRequest)
		return
	}

	job, ok := h.store.Get(id)
	if !ok {
		writeError(w, "job not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// -----------------------------------------------------------------------
// GET /api/jobs — list all jobs (most recent first, up to 50)
// -----------------------------------------------------------------------

func (h *Handler) listJobsHandler(w http.ResponseWriter, r *http.Request) {
	allJobs := h.store.List()

	// Limit to 50 most recent (simple: just take all, sort not guaranteed in map).
	max := 50
	if len(allJobs) < max {
		max = len(allJobs)
	}
	responses := make([]models.JobResponse, 0, max)
	for _, job := range allJobs[:max] {
		responses = append(responses, job.ToResponse())
	}

	writeJSON(w, http.StatusOK, responses)
}

// -----------------------------------------------------------------------
// GET /api/languages — list supported languages
// -----------------------------------------------------------------------

func (h *Handler) languagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"languages": h.registry.SupportedLanguages(),
	})
}

// -----------------------------------------------------------------------
// GET /api/health — health check
// -----------------------------------------------------------------------

type healthResponse struct {
	Status    string `json:"status"`
	QueueLen  int    `json:"queue_len"`
	QueueCap  int    `json:"queue_cap"`
	JobsTotal int    `json:"jobs_total"`
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		QueueLen:  h.queue.Len(),
		QueueCap:  h.queue.Cap(),
		JobsTotal: h.store.Count(),
	})
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// writeJSON serializes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode error", "error", err)
	}
}

// writeError writes a JSON error response.
// Internal error details are NOT exposed to clients.
func writeError(w http.ResponseWriter, message string, status int) {
	slog.Warn("API error", "message", message, "status", status)
	writeJSON(w, status, models.ErrorResponse{Error: message})
}

// CORSMiddleware adds CORS headers so the frontend can talk to the API
// even if served from a different port during development.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs every HTTP request with method, path, and status.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		slog.Info("HTTP",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
