// Package models defines the core data types shared across the system.
package models

import "time"

// Status represents the lifecycle state of a submitted job.
// Using typed constants instead of scattered strings prevents typos
// and makes the code self-documenting.
type Status string

const (
	// StatusQueued means the job has been accepted and is waiting in the queue.
	StatusQueued Status = "QUEUED"

	// StatusRunning means a worker has picked up the job and is executing it.
	StatusRunning Status = "RUNNING"

	// StatusCompleted means the code ran successfully (exit code 0).
	StatusCompleted Status = "COMPLETED"

	// StatusCompileError means the code failed to compile (non-zero exit from compiler).
	StatusCompileError Status = "COMPILE_ERROR"

	// StatusRuntimeError means the code compiled but exited with a non-zero exit code.
	StatusRuntimeError Status = "RUNTIME_ERROR"

	// StatusTimeout means the code exceeded the maximum allowed execution time.
	StatusTimeout Status = "TIMEOUT"

	// StatusFailed means an internal error occurred (e.g., Docker unavailable).
	StatusFailed Status = "FAILED"
)

// Job is the central data structure that travels through the system.
// It starts as a submission request and accumulates result data as it executes.
type Job struct {
	// --- Identity ---
	ID       string `json:"id"`
	Language string `json:"language"`

	// --- Input ---
	Code  string `json:"code"`
	Stdin string `json:"stdin"`

	// --- Lifecycle ---
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`

	// --- Output ---
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`

	// --- Timing ---
	// ExecutionTimeMs is the wall-clock milliseconds from execution start to finish.
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// --- Error ---
	// ErrorMessage contains a human-readable error description when Status is not COMPLETED.
	ErrorMessage string `json:"error_message,omitempty"`
}

// SubmitRequest is the JSON body the frontend sends when submitting code.
type SubmitRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Stdin    string `json:"stdin"`
}

// SubmitResponse is returned immediately after a job is accepted.
type SubmitResponse struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

// JobResponse is what the frontend receives when polling for job status.
// It is identical to Job but defined separately so we can evolve them
// independently in the future.
type JobResponse struct {
	ID              string    `json:"id"`
	Language        string    `json:"language"`
	Status          Status    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	Stdout          string    `json:"stdout"`
	Stderr          string    `json:"stderr"`
	ExitCode        int       `json:"exit_code"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// ToResponse converts a Job to a JobResponse for the HTTP API.
func (j *Job) ToResponse() JobResponse {
	return JobResponse{
		ID:              j.ID,
		Language:        j.Language,
		Status:          j.Status,
		CreatedAt:       j.CreatedAt,
		StartedAt:       j.StartedAt,
		FinishedAt:      j.FinishedAt,
		Stdout:          j.Stdout,
		Stderr:          j.Stderr,
		ExitCode:        j.ExitCode,
		ExecutionTimeMs: j.ExecutionTimeMs,
		ErrorMessage:    j.ErrorMessage,
	}
}

// ErrorResponse is the JSON body returned when an API request fails.
type ErrorResponse struct {
	Error string `json:"error"`
}
