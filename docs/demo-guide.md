# Demo Guide

## Prepare the normal server

Use a terminal in the project root. Start Docker Desktop and verify that all four oces-* images are built. Check the profile:

```bash
docker info --format '{{json .SecurityOptions}}'
docker run --rm --network none oces-python python3 -c 'print("Docker ready")'
```

The recorded final setup uses profile=builtin. In Docker Desktop's Docker Engine JSON, explicitly setting `"seccomp-profile": "builtin"` resolved an unconfined default on the test machine. Preserve other settings when editing and apply/restart Docker afterward. Confirm actual filtering with the sandbox probe.

Start one normal server (stop an existing server on the same port before restarting):

```bash
DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')" go run -race ./cmd/server
```

Open http://localhost:8080 and refresh the page. Defaults are four workers, 100 waiting slots, and a shared 10-second setup/compile/run deadline. Docker Desktop wake-up delays can consume this deadline; warm Docker immediately before the demonstration. Resource Saver was a possible explanation for an observed setup timeout, not a confirmed diagnosis.

## Demonstration sequence

1. Select Python and load its example. Enter `5` and `3` on separate lines in stdin before running. Show the sum of 8 and exit code 0.
2. Show a Go, C++, or Java example. Keep saved screenshots available if local startup timing varies.
3. Run `bash scripts/demo_concurrency.sh` in a second terminal. Explain that four workers may process jobs concurrently and another waits. Use the worker IDs in the logs and the saved measured results as evidence; a particular worker does not always receive a particular job.
4. Run the C++ program below to show a compile error.
5. Run the Python loop below to show TIMEOUT and exit code 124. The elapsed time includes setup and cleanup, so do not promise an exact ten-second screen update.
6. Show FINAL_VERIFICATION.md and the saved JSON reports for queue rejection, limits, sandbox observations, and load measurements.

### Compile-error example

```cpp
#include <iostream>
int main() {
    std::cout << "Hello"
    return 0;
}
```

### Timeout example

```python
print("Started", flush=True)
while True:
    pass
```

## Repeat the isolated verification

In a separate terminal, start the verification server:

```bash
PORT=8081 WORKER_COUNT=1 QUEUE_SIZE=1 EXECUTION_TIMEOUT=10s DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')" go run -race ./cmd/server
```

From a free terminal, warm Docker and run the checks sequentially:

```bash
docker run --rm --network none oces-python python3 -c 'print("Docker ready")'
python3 scripts/verify_queue_full.py
python3 scripts/verify_sandbox.py
python3 scripts/verify_languages.py
```

Stop the 8081 verification server with Ctrl+C when finished. The normal 8080 server is the one to use for the four-worker demonstration. Reports are saved under docs/verification/. Load tests use the normal server and are documented in testing-summary.md.
