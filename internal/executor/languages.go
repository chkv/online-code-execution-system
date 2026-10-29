// Language-specific executors.
// Each executor implements the Executor interface.
// Pattern (adapted from Judex): each language knows its Docker image,
// source filename, and the sequence of commands to run.

package executor

import (
	"context"
	"log/slog"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// -----------------------------------------------------------------------
// Python
// -----------------------------------------------------------------------

// PythonExecutor runs Python 3 programs.
// Python is interpreted — no compilation step needed.
// Command: python3 main.py  (with optional stdin piped)
type PythonExecutor struct {
	image string
}

func NewPythonExecutor(image string) *PythonExecutor {
	return &PythonExecutor{image: image}
}

func (e *PythonExecutor) SourceFilename() string { return "main.py" }
func (e *PythonExecutor) DockerImage() string    { return e.image }

func (e *PythonExecutor) Execute(ctx context.Context, job *models.Job, timeout time.Duration) Result {
	start := time.Now()
	slog.Info("executor: running python", "job_id", job.ID)

	results, err := runInDocker(ctx, e.image, "main.py", job.Code, job.Stdin,
		[][]string{
			{"python3", "main.py"},
		},
		timeout,
	)
	if err != nil {
		slog.Error("executor: docker error", "job_id", job.ID, "error", err)
		return Result{Status: models.StatusFailed, ErrorMessage: err.Error()}
	}

	result := interpretResult(results, false /* no compile step */)
	result.ExecutionTimeMs = time.Since(start).Milliseconds()
	return result
}

// -----------------------------------------------------------------------
// Go
// -----------------------------------------------------------------------

// GoExecutor compiles and runs Go programs.
// Step 1: go build -o /workspace/app /workspace/main.go
// Step 2: /workspace/app
type GoExecutor struct {
	image string
}

func NewGoExecutor(image string) *GoExecutor {
	return &GoExecutor{image: image}
}

func (e *GoExecutor) SourceFilename() string { return "main.go" }
func (e *GoExecutor) DockerImage() string    { return e.image }

func (e *GoExecutor) Execute(ctx context.Context, job *models.Job, timeout time.Duration) Result {
	start := time.Now()
	slog.Info("executor: running go", "job_id", job.ID)

	results, err := runInDocker(ctx, e.image, "main.go", job.Code, job.Stdin,
		[][]string{
			{"go", "build", "-o", "/workspace/app", "/workspace/main.go"},
			{"/workspace/app"},
		},
		timeout,
	)
	if err != nil {
		slog.Error("executor: docker error", "job_id", job.ID, "error", err)
		return Result{Status: models.StatusFailed, ErrorMessage: err.Error()}
	}

	result := interpretResult(results, true /* has compile step */)
	result.ExecutionTimeMs = time.Since(start).Milliseconds()
	return result
}

// -----------------------------------------------------------------------
// C++
// -----------------------------------------------------------------------

// CppExecutor compiles and runs C++ programs.
// Step 1: g++ main.cpp -o app
// Step 2: /workspace/app
type CppExecutor struct {
	image string
}

func NewCppExecutor(image string) *CppExecutor {
	return &CppExecutor{image: image}
}

func (e *CppExecutor) SourceFilename() string { return "main.cpp" }
func (e *CppExecutor) DockerImage() string    { return e.image }

func (e *CppExecutor) Execute(ctx context.Context, job *models.Job, timeout time.Duration) Result {
	start := time.Now()
	slog.Info("executor: running cpp", "job_id", job.ID)

	results, err := runInDocker(ctx, e.image, "main.cpp", job.Code, job.Stdin,
		[][]string{
			{"g++", "/workspace/main.cpp", "-o", "/workspace/app"},
			{"/workspace/app"},
		},
		timeout,
	)
	if err != nil {
		slog.Error("executor: docker error", "job_id", job.ID, "error", err)
		return Result{Status: models.StatusFailed, ErrorMessage: err.Error()}
	}

	result := interpretResult(results, true /* has compile step */)
	result.ExecutionTimeMs = time.Since(start).Milliseconds()
	return result
}

// -----------------------------------------------------------------------
// Java
// -----------------------------------------------------------------------

// JavaExecutor compiles and runs Java programs.
// Step 1: javac Main.java
// Step 2: java -cp /workspace Main
type JavaExecutor struct {
	image string
}

func NewJavaExecutor(image string) *JavaExecutor {
	return &JavaExecutor{image: image}
}

func (e *JavaExecutor) SourceFilename() string { return "Main.java" }
func (e *JavaExecutor) DockerImage() string    { return e.image }

func (e *JavaExecutor) Execute(ctx context.Context, job *models.Job, timeout time.Duration) Result {
	start := time.Now()
	slog.Info("executor: running java", "job_id", job.ID)

	results, err := runInDocker(ctx, e.image, "Main.java", job.Code, job.Stdin,
		[][]string{
			{"javac", "/workspace/Main.java"},
			{"java", "-cp", "/workspace", "Main"},
		},
		timeout,
	)
	if err != nil {
		slog.Error("executor: docker error", "job_id", job.ID, "error", err)
		return Result{Status: models.StatusFailed, ErrorMessage: err.Error()}
	}

	result := interpretResult(results, true /* has compile step */)
	result.ExecutionTimeMs = time.Since(start).Milliseconds()
	return result
}

// -----------------------------------------------------------------------
// Registry — maps language name to Executor
// -----------------------------------------------------------------------

// Registry maps a language string (from the API request) to its Executor.
// To add a new language: add an entry here and build a Docker image.
type Registry map[string]Executor

// NewRegistry creates a populated Registry from the config image map.
func NewRegistry(images map[string]string) Registry {
	r := make(Registry)
	if img, ok := images["python"]; ok {
		r["python"] = NewPythonExecutor(img)
	}
	if img, ok := images["go"]; ok {
		r["go"] = NewGoExecutor(img)
	}
	if img, ok := images["cpp"]; ok {
		r["cpp"] = NewCppExecutor(img)
	}
	if img, ok := images["java"]; ok {
		r["java"] = NewJavaExecutor(img)
	}
	return r
}

// Get returns the Executor for the given language, or nil if unsupported.
func (r Registry) Get(language string) (Executor, bool) {
	e, ok := r[language]
	return e, ok
}

// SupportedLanguages returns a sorted list of supported language names.
func (r Registry) SupportedLanguages() []string {
	langs := make([]string, 0, len(r))
	for lang := range r {
		langs = append(langs, lang)
	}
	return langs
}
