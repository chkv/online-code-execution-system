package executor

import (
	"sync"
	"testing"

	"github.com/docker/docker/client"
)

func TestGetDockerClient_ConcurrentInitialization(t *testing.T) {
	// Start with an uninitialized client.
	dockerClientMu.Lock()
	previousClient := dockerClient
	dockerClient = nil
	dockerClientMu.Unlock()

	t.Cleanup(func() {
		dockerClientMu.Lock()
		testClient := dockerClient
		dockerClient = previousClient
		dockerClientMu.Unlock()

		if testClient != nil {
			if err := testClient.Close(); err != nil {
				t.Errorf("closing test client: %v", err)
			}
		}
	})

	const callers = 50
	clients := make([]*client.Client, callers)
	errors := make([]error, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			clients[index], errors[index] = getDockerClient()
		}(i)
	}

	close(start)
	wg.Wait()

	for i := 0; i < callers; i++ {
		if errors[i] != nil {
			t.Fatalf("caller %d: %v", i, errors[i])
		}
		if clients[i] == nil {
			t.Fatalf("caller %d received a nil client", i)
		}
		if clients[i] != clients[0] {
			t.Fatalf("caller %d received a different client", i)
		}
	}
}
