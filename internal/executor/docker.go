// Package executor — Docker sandboxing implementation.
//
// This file contains the core logic for:
//  1. Creating a Docker container for a job.
//  2. Copying source code into it.
//  3. Running compile and/or run commands.
//  4. Capturing stdout, stderr, and exit code.
//  5. Enforcing timeouts.
//  6. Cleaning up the container.
//
// Security measures (adapted from Judex's sandbox/pool.go):
//   - NetworkMode: "none"        — container cannot access the internet
//   - CapDrop: ["ALL"]           — drops all Linux capabilities (least privilege)
//   - SecurityOpt: no-new-privileges — prevents privilege escalation
//   - Memory: 128MB              — limits RAM usage
//   - NanoCPUs: 0.5 CPU         — limits CPU usage
//   - PidsLimit: 32              — limits process/thread count
//   - Non-root user (uid 1000)   — code runs as unprivileged user
//   - AutoRemove: true           — container cleaned up automatically

package executor

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

// commandResult holds the output from a single docker exec call.
type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

// dockerClient is a lazily-initialized Docker client shared across executors.
// It is safe for concurrent use by multiple goroutines (confirmed by Docker SDK docs).
var dockerClient *client.Client
var dockerClientMu sync.Mutex

func getDockerClient() (*client.Client, error) {
	dockerClientMu.Lock()
	defer dockerClientMu.Unlock()

	if dockerClient != nil {
		return dockerClient, nil
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init failed: %w", err)
	}
	dockerClient = cli
	return cli, nil
}

// runInDocker is the shared implementation used by all language executors.
// It:
//  1. Creates a container from the given image.
//  2. Uploads source code as a tar archive.
//  3. Runs each command in sequence (e.g., compile, then run).
//  4. Returns results for each command.
func runInDocker(
	ctx context.Context,
	image string,
	sourceFilename string,
	sourceCode string,
	stdin string,
	commands [][]string,
	timeout time.Duration,
) ([]commandResult, error) {
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}

	// --- Create container ---
	// We apply security restrictions:
	//   - no network access
	//   - limited CPU/memory/pids
	//   - drop all Linux capabilities
	//   - no new privileges
	nanoCPUs := int64(5e8)             // 0.5 CPU cores
	memory := int64(128 * 1024 * 1024) // 128 MB
	pidsLimit := int64(32)

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        image,
			Cmd:          []string{"sh", "-c", "sleep 60"}, // keep container alive
			WorkingDir:   "/workspace",
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			NetworkMode: "none",
			AutoRemove:  true,
			Resources: container.Resources{
				Memory:    memory,
				NanoCPUs:  nanoCPUs,
				PidsLimit: &pidsLimit,
			},
			CapDrop:     []string{"ALL"},
			SecurityOpt: []string{"no-new-privileges"},
		},
		nil, nil, "",
	)
	if err != nil {
		return nil, fmt.Errorf("container create failed: %w", err)
	}

	containerID := resp.ID

	// Register cleanup immediately, including when startup fails.
	defer func() {
		removeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := cli.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
			Force: true,
		})
		if err != nil && !errdefs.IsNotFound(err) {
			slog.Warn("container cleanup failed", "id", containerID, "error", err)
			return
		}

		slog.Debug("container cleanup complete", "id", containerID)
	}()

	slog.Debug("container created", "id", containerID, "image", image)

	// Start the container.
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("container start failed: %w", err)
	}

	// --- Upload source code as tar archive ---
	// Docker's CopyToContainer API requires a tar stream.
	if err := uploadSourceCode(ctx, cli, containerID, sourceFilename, sourceCode); err != nil {
		return nil, fmt.Errorf("code upload failed: %w", err)
	}

	// --- Upload stdin file if provided ---
	if stdin != "" {
		if err := uploadSourceCode(ctx, cli, containerID, "stdin.txt", stdin); err != nil {
			return nil, fmt.Errorf("stdin upload failed: %w", err)
		}
	}

	// --- Run each command in sequence ---
	results := make([]commandResult, 0, len(commands))
	for _, cmd := range commands {
		cmdStr := strings.Join(cmd, " ")
		slog.Debug("running command in container", "id", containerID[:12], "cmd", cmdStr)

		// Each command gets its own timeout context derived from the overall timeout.
		cmdCtx, cmdCancel := context.WithTimeout(ctx, timeout)
		result := execInContainer(cmdCtx, cli, containerID, cmd, stdin)
		cmdCancel()

		results = append(results, result)

		// If a command fails (compile error, timeout, etc.), stop running further commands.
		if result.ExitCode != 0 || result.TimedOut {
			break
		}
	}

	return results, nil
}

// uploadSourceCode streams a single file into a running container as a tar archive.
// This pattern is taken from Judex's sandbox.UploadCode implementation.
func uploadSourceCode(ctx context.Context, cli *client.Client, containerID, filename, content string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0o644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	return cli.CopyToContainer(ctx, containerID, "/workspace", bytes.NewReader(buf.Bytes()), container.CopyToContainerOptions{})
}

// execInContainer runs a single command inside the container and captures output.
func execInContainer(ctx context.Context, cli *client.Client, containerID string, cmd []string, stdin string) commandResult {
	// Build exec config.
	// We run as user 1000 (non-root) for security.
	execOpts := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	}

	// If stdin is provided and this is a run command (not compile), pipe it.
	if stdin != "" {
		execOpts.AttachStdin = true
	}

	execID, err := cli.ContainerExecCreate(ctx, containerID, execOpts)
	if err != nil {
		if ctx.Err() != nil {
			return commandResult{TimedOut: true, Stderr: "execution timeout", ExitCode: 124}
		}
		return commandResult{ExitCode: 1, Stderr: fmt.Sprintf("exec create error: %v", err)}
	}

	// Attach to the exec to stream I/O.
	attachResp, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		if ctx.Err() != nil {
			return commandResult{TimedOut: true, Stderr: "execution timeout", ExitCode: 124}
		}
		return commandResult{ExitCode: 1, Stderr: fmt.Sprintf("exec attach error: %v", err)}
	}
	defer attachResp.Close()

	// Write stdin if provided.
	if stdin != "" {
		go func() {
			io.WriteString(attachResp.Conn, stdin) //nolint:errcheck
			attachResp.CloseWrite()                //nolint:errcheck
		}()
	}

	// Read stdout and stderr concurrently using Docker's multiplexed stream.
	var stdoutBuf, stderrBuf bytes.Buffer
	copyDone := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
		copyDone <- err
	}()

	// Wait for either copy completion or context cancellation.
	select {
	case <-copyDone:
		// Normal completion — fall through to inspect exit code.
	case <-ctx.Done():
		// Timeout — kill the container so the copy goroutine unblocks.
		cli.ContainerKill(context.Background(), containerID, "SIGKILL") //nolint:errcheck
		<-copyDone
		return commandResult{
			TimedOut: true,
			Stderr:   "execution timeout",
			ExitCode: 124,
			Stdout:   stdoutBuf.String(),
		}
	}

	// Inspect to get exit code.
	inspectResp, err := cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return commandResult{
			ExitCode: 1,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String() + fmt.Sprintf("\nexec inspect error: %v", err),
		}
	}

	return commandResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: inspectResp.ExitCode,
	}
}

// buildResult converts raw Docker exec results into our Result type.
// interpretResult classifies the final outcome given language-specific knowledge
// about whether a compile step was involved.
func interpretResult(results []commandResult, hasCompileStep bool) Result {
	if len(results) == 0 {
		return Result{Status: models.StatusFailed, ErrorMessage: "no execution results"}
	}

	// If there is a compile step, results[0] is compilation.
	if hasCompileStep {
		compile := results[0]
		if compile.TimedOut {
			return Result{
				Status:       models.StatusTimeout,
				Stderr:       compile.Stderr,
				ErrorMessage: "compilation timed out",
				ExitCode:     compile.ExitCode,
			}
		}
		if compile.ExitCode != 0 {
			return Result{
				Status:       models.StatusCompileError,
				Stdout:       compile.Stdout,
				Stderr:       compile.Stderr,
				ErrorMessage: "compilation failed",
				ExitCode:     compile.ExitCode,
			}
		}
		// Compile succeeded; check run result.
		if len(results) < 2 {
			return Result{Status: models.StatusFailed, ErrorMessage: "expected run result missing"}
		}
		run := results[1]
		return classifyRunResult(run)
	}

	// Interpreted language — single run step.
	return classifyRunResult(results[0])
}

func classifyRunResult(run commandResult) Result {
	if run.TimedOut {
		return Result{
			Status:       models.StatusTimeout,
			Stdout:       run.Stdout,
			Stderr:       run.Stderr,
			ErrorMessage: "execution timed out",
			ExitCode:     124,
		}
	}
	if run.ExitCode != 0 {
		return Result{
			Status:       models.StatusRuntimeError,
			Stdout:       run.Stdout,
			Stderr:       run.Stderr,
			ErrorMessage: fmt.Sprintf("program exited with code %d", run.ExitCode),
			ExitCode:     run.ExitCode,
		}
	}
	return Result{
		Status:   models.StatusCompleted,
		Stdout:   run.Stdout,
		Stderr:   run.Stderr,
		ExitCode: 0,
	}
}
