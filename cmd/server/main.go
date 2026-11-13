// Online Code Execution System — main entry point.
//
// This file wires all components together:
//   1. Load configuration from environment.
//   2. Create the in-memory job store.
//   3. Create the job queue (buffered channel).
//   4. Create the executor registry (one executor per language).
//   5. Create and start the worker pool.
//   6. Register HTTP routes.
//   7. Start the HTTP server.
//
// Concurrency overview:
//   - Main goroutine runs the HTTP server.
//   - N worker goroutines run concurrently, reading from the job queue.
//   - All shared state (store) is protected by sync.RWMutex.
//   - The job queue channel is safe for concurrent producer/consumer access.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keerthivardhanchandana/online-code-execution-system/internal/api"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/config"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/executor"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/jobs"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/storage"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/worker"
)

func main() {
	// Configure structured logging — output goes to stdout.
	// Using log/slog (Go 1.21+) for structured key=value log lines.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// --- Phase 1: Load configuration ---
	cfg := config.Load()
	slog.Info("configuration loaded",
		"port", cfg.Port,
		"workers", cfg.WorkerCount,
		"queue_size", cfg.QueueSize,
		"execution_timeout", cfg.ExecutionTimeout,
	)

	// --- Phase 2: Initialize core components ---

	// Job store: thread-safe in-memory map.
	// (Why not Redis? We run as one process. A mutex-protected map is simpler
	//  and demonstrates the synchronization concept just as well.)
	store := storage.NewStore()

	// Job queue: buffered Go channel.
	// (Why a channel? Channels are Go's built-in concurrent producer-consumer
	//  mechanism. They are safe, idiomatic, and easy to reason about.)
	queue := jobs.NewQueue(cfg.QueueSize)

	// Executor registry: maps language strings to Docker-based executors.
	registry := executor.NewRegistry(cfg.DockerImages)

	slog.Info("supported languages", "languages", registry.SupportedLanguages())

	// --- Phase 3: Start worker pool ---
	// This launches cfg.WorkerCount goroutines, each waiting for jobs.
	// THIS IS WHERE CONCURRENT EXECUTION HAPPENS.
	pool := worker.NewPool(cfg.WorkerCount, queue, store, registry, cfg.ExecutionTimeout)
	pool.Start()

	slog.Info("worker pool started", "workers", pool.WorkerCount())

	// --- Phase 4: Set up HTTP server ---
	// Serve frontend static files from the web/ directory.
	fs := http.FileServer(http.Dir("web"))
	handler := api.NewHandler(store, queue, registry, cfg.WorkerCount, cfg.ExecutionTimeout)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, fs)

	// Apply middleware: logging + CORS.
	httpHandler := api.CORSMiddleware(api.LoggingMiddleware(mux))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// --- Phase 5: Start server + graceful shutdown ---
	// Listen for OS signals so we can stop cleanly (finish in-progress jobs).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info(fmt.Sprintf("server started on http://localhost:%s", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until a shutdown signal is received.
	<-quit
	slog.Info("shutdown signal received")

	// Give in-flight HTTP requests 10 seconds to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	// Close the job queue. Workers will finish their current job then stop.
	queue.Close()
	pool.Stop()

	slog.Info("server stopped cleanly")
}
