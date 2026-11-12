package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

func TestRunInDocker_CleansUpAfterStartFailure(t *testing.T) {
	const containerID = "test-container-123456"
	const base = "/v1.45/containers/"
	removed := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == base+"create":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"Id": containerID})

		case r.Method == http.MethodPost && r.URL.Path == base+containerID+"/start":
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "simulated startup failure",
			})

		case r.Method == http.MethodDelete && r.URL.Path == base+containerID:
			removed <- r.URL.Query().Get("force") == "1"
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Errorf("unexpected Docker request: %s %s", r.Method, r.URL.Path)
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

	// Temporarily use the mock Docker API.
	dockerClientMu.Lock()
	previousClient := dockerClient
	dockerClient = testClient
	dockerClientMu.Unlock()

	defer func() {
		dockerClientMu.Lock()
		dockerClient = previousClient
		dockerClientMu.Unlock()
	}()

	_, err = runInDocker(
		context.Background(),
		"test-image",
		"main.py",
		"print('hello')",
		"",
		[][]string{{"python3", "main.py"}},
		time.Second,
	)

	if err == nil || !strings.Contains(err.Error(), "container start failed") {
		t.Fatalf("expected container startup error, got: %v", err)
	}

	select {
	case forced := <-removed:
		if !forced {
			t.Fatal("cleanup did not request forced removal")
		}
	default:
		t.Fatal("container cleanup was not attempted")
	}
}
