#!/usr/bin/env python3
"""Check a test server with one worker and one queue slot."""

import json
import time
from datetime import datetime
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen

BASE = "http://localhost:8081"


def request_json(path, code=None):
    request = Request(BASE + path)
    if code is not None:
        request = Request(
            BASE + path,
            data=json.dumps({
                "language": "python", "code": code, "stdin": ""
            }).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
    try:
        with urlopen(request, timeout=5) as response:
            return response.status, json.load(response)
    except HTTPError as error:
        with error:
            return error.code, json.load(error)


def require(condition, message):
    if not condition:
        raise SystemExit("FAIL: " + message)


def main():
    status, health = request_json("/api/health")
    require(status == 200, f"health check returned HTTP {status}")
    require(
        health.get("worker_count") == 1
        and health.get("queue_cap") == 1
        and health.get("execution_timeout_seconds") == 10,
        "use port 8081 with one worker, one queue slot, and a 10s timeout",
    )
    require(health.get("queue_len") == 0, "test queue is already occupied")

    first_status, first = request_json(
        "/api/jobs", "import time\ntime.sleep(4)\nprint('first done')\n"
    )
    require(
        first_status == 202 and bool(first.get("id")),
        f"first submission: {first_status}, {first}",
    )
    first_id = first["id"]
    deadline = time.monotonic() + 5

    while True:
        status, running = request_json(f"/api/jobs/{first_id}")
        require(status == 200, f"job lookup returned HTTP {status}")
        if running["status"] == "RUNNING":
            break
        require(
            running["status"] == "QUEUED",
            f"first job ended too early: {running}",
        )
        require(time.monotonic() < deadline, "first job never reached RUNNING")
        time.sleep(0.05)

    print("Job 1: accepted and RUNNING", flush=True)

    second_status, second = request_json(
        "/api/jobs", "print('second done')\n"
    )
    require(
        second_status == 202 and bool(second.get("id")),
        f"second submission: {second_status}, {second}",
    )
    second_id = second["id"]
    require(second_id != first_id, "duplicate job IDs returned")

    status, queued = request_json(f"/api/jobs/{second_id}")
    require(
        status == 200 and queued.get("status") == "QUEUED",
        f"second job was not waiting: {queued}",
    )
    print("Job 2: accepted and QUEUED", flush=True)

    rejected_status, rejected = request_json(
        "/api/jobs", "print('should not run')\n"
    )
    require(
        rejected_status == 429,
        f"expected HTTP 429, got {rejected_status}: {rejected}",
    )
    require(
        "queue" in rejected.get("error", "").lower(),
        f"unexpected rejection: {rejected}",
    )
    print(f"Job 3: HTTP 429 — {rejected['error']}", flush=True)

    expected = {first_id: "first done\n", second_id: "second done\n"}
    results = {}
    deadline = time.monotonic() + 45

    while len(results) < 2:
        require(
            time.monotonic() < deadline,
            "accepted jobs did not finish within 45 seconds",
        )
        for job_id in expected:
            if job_id in results:
                continue
            status, job = request_json(f"/api/jobs/{job_id}")
            require(status == 200, f"job lookup returned HTTP {status}")
            if job["status"] in {"QUEUED", "RUNNING"}:
                continue
            require(
                job["status"] == "COMPLETED"
                and job.get("exit_code") == 0
                and job.get("stdout") == expected[job_id],
                f"accepted job did not complete correctly: {job}",
            )
            results[job_id] = job
            print(
                f"{job_id[:8]}: COMPLETED | exit=0 | expected output received",
                flush=True,
            )
        if len(results) < 2:
            time.sleep(0.2)

    now = datetime.now().astimezone()
    folder = Path(__file__).resolve().parents[1] / "docs" / "verification"
    folder.mkdir(parents=True, exist_ok=True)
    output = folder / f"queue-full-{now:%Y%m%d-%H%M%S-%f}.json"

    report = {
        "recorded_at": now.isoformat(),
        "test": "queue saturation with one worker and one waiting slot",
        "base_url": BASE,
        "health": health,
        "submission_http_statuses": [
            first_status, second_status, rejected_status
        ],
        "running_job_observed": running,
        "queued_job_observed": queued,
        "rejection": rejected,
        "accepted_jobs": [results[job_id] for job_id in expected],
        "passed": True,
    }
    output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"Saved: {output}")
    print("PASS: full queue returned HTTP 429; both accepted jobs completed correctly.")


if __name__ == "__main__":
    main()
