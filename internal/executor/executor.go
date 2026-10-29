// Package executor defines the Executor interface and a Result type.
//
// An Executor is responsible for running a single job inside a Docker container
// and returning the result. Each language has its own Executor implementation.
//
// Design (inspired by Judex's executor abstraction):
//   type Executor interface {
//       Execute(ctx, job) Result
//   }
//
// Our version differs from Judex: instead of taking a sandbox.Sandbox (which
// requires pre-warmed containers), our Execute method receives the full Job
// and handles Docker container creation/teardown internally. This trades a
// tiny bit of startup latency for dramatically simpler code.
package executor

import (
	"context"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// Result is the output of running a job in a Docker container.
type Result struct {
	Stdout          string
	Stderr          string
	ExitCode        int
	Status          models.Status
	ExecutionTimeMs int64
	ErrorMessage    string
}

// Executor is the interface every language executor must implement.
// Adding a new language means implementing this interface and registering
// it in the executor registry — no other code needs to change.
type Executor interface {
	// Execute runs the job inside a Docker container and returns the Result.
	// The ctx carries the execution timeout; implementations must respect it.
	Execute(ctx context.Context, job *models.Job, timeout time.Duration) Result

	// SourceFilename returns the filename the source code should be saved as
	// inside the container (e.g., "main.py", "main.go", "main.cpp").
	SourceFilename() string

	// DockerImage returns the Docker image name to use for this language.
	DockerImage() string
}
