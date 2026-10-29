# API Reference

The Online Code Execution System exposes a RESTful HTTP API.

## Base URL
All endpoints are relative to the root URL of the server (e.g., `http://localhost:8080`).

---

## 1. Submit a Job
Submits source code for execution.

**Endpoint:** `POST /api/jobs`
**Content-Type:** `application/json`

### Request Body
```json
{
  "language": "python",
  "code": "print('Hello, World!')",
  "stdin": "Optional input string to pass to the program"
}
```
- `language` (string, required): One of `python`, `go`, `cpp`, `java`.
- `code` (string, required): The source code to execute.
- `stdin` (string, optional): Data to be fed to the program's standard input.

### Responses
**202 Accepted** (Job successfully queued)
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "QUEUED"
}
```

**400 Bad Request** (Invalid language or missing code)
**429 Too Many Requests** (Queue is full)

---

## 2. Get Job Status
Retrieves the current status and results of a submitted job.

**Endpoint:** `GET /api/jobs/{id}`

### Responses
**200 OK**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "language": "python",
  "status": "COMPLETED",
  "stdout": "Hello, World!\n",
  "stderr": "",
  "exit_code": 0,
  "execution_time_ms": 45,
  "error_message": "",
  "created_at": "2023-10-01T12:00:00Z",
  "started_at": "2023-10-01T12:00:01Z",
  "finished_at": "2023-10-01T12:00:02Z"
}
```
*Note: Clients should poll this endpoint (e.g., every 500ms) until `status` is a terminal state (`COMPLETED`, `COMPILE_ERROR`, `RUNTIME_ERROR`, `TIMEOUT`, `FAILED`).*

---

## 3. List Recent Jobs
Returns up to the 50 most recent jobs.

**Endpoint:** `GET /api/jobs`

### Responses
**200 OK**
```json
[
  {
    "id": "...",
    "language": "python",
    "status": "COMPLETED",
    ...
  }
]
```

---

## 4. Get Supported Languages
Returns the list of languages the system can execute.

**Endpoint:** `GET /api/languages`

### Responses
**200 OK**
```json
{
  "languages": [
    "cpp",
    "go",
    "java",
    "python"
  ]
}
```

---

## 5. Health Check
Returns the current health and queue metrics of the server.

**Endpoint:** `GET /api/health`

### Responses
**200 OK**
```json
{
  "status": "ok",
  "queue_len": 2,
  "queue_cap": 100,
  "jobs_total": 42
}
```
