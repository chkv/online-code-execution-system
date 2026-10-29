// Package config holds all runtime configuration for the server.
// Values are read from environment variables with sensible defaults.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configurable parameters for the server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// WorkerCount is how many goroutine workers process jobs concurrently.
	WorkerCount int

	// QueueSize is the maximum number of jobs that can be buffered in the queue.
	QueueSize int

	// ExecutionTimeout is the maximum wall-clock time allowed for one job.
	ExecutionTimeout time.Duration

	// DockerImages maps language names to Docker image names.
	DockerImages map[string]string
}

// Load reads environment variables and returns a Config.
// If an environment variable is not set, the default value is used.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		WorkerCount:      getEnvInt("WORKER_COUNT", 4),
		QueueSize:        getEnvInt("QUEUE_SIZE", 100),
		ExecutionTimeout: getEnvDuration("EXECUTION_TIMEOUT", 10*time.Second),
		DockerImages: map[string]string{
			"python": getEnv("DOCKER_IMAGE_PYTHON", "oces-python"),
			"go":     getEnv("DOCKER_IMAGE_GO", "oces-go"),
			"cpp":    getEnv("DOCKER_IMAGE_CPP", "oces-cpp"),
			"java":   getEnv("DOCKER_IMAGE_JAVA", "oces-java"),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
