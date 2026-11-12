package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/keerthivardhanchandana/online-code-execution-system/internal/models"
)

func TestExecutors_SetupTimeoutAndCleanup(t *testing.T) {
	cases := []struct {
		name     string
		executor Executor
	}{
		{"python", NewPythonExecutor("test-image")},
		{"go", NewGoExecutor("test-image")},
		{"cpp", NewCppExecutor("test-image")},
		{"java", NewJavaExecutor("test-image")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const containerID = "timeout-container-123456"
			const base = "/v1.45/containers/"
			removed := make(chan bool, 1)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == http.MethodPost && r.URL.Path == base+"create":
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]string{"Id": containerID})

				case r.Method == http.MethodPost && r.URL.Path == base+containerID+"/start":
					// Simulate Docker taking too long to start the container.
					select {
					case <-r.Context().Done():
						return
					case <-time.After(3 * time.Second):
						http.Error(w, "startup stalled", http.StatusGatewayTimeout)
					}

				case r.Method == http.MethodDelete && r.URL.Path == base+containerID:
					removed <- r.URL.Query().Get("force") == "1"
					w.WriteHeader(http.StatusNoContent)

				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()

			testClient, err := client.NewClientWithOpts(
				client.WithHost(server.URL),
				client.WithVersion("1.45"),
				client.WithHTTPClient(server.Client()),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer testClient.Close()

			dockerClientMu.Lock()
			previousClient := dockerClient
			dockerClient = testClient
			dockerClientMu.Unlock()

			defer func() {
				dockerClientMu.Lock()
				dockerClient = previousClient
				dockerClientMu.Unlock()
			}()

			result := tc.executor.Execute(
				context.Background(),
				&models.Job{
					ID:       "timeout-test",
					Language: tc.name,
					Code:     "unused during setup",
				},
				500*time.Millisecond,
			)

			if result.Status != models.StatusTimeout || result.ExitCode != 124 {
				t.Fatalf("expected TIMEOUT/124, got %s/%d: %s",
					result.Status, result.ExitCode, result.ErrorMessage)
			}

			select {
			case forced := <-removed:
				if !forced {
					t.Fatal("cleanup did not request forced removal")
				}
			default:
				t.Fatal("cleanup was not attempted after timeout")
			}
		})
	}
}
