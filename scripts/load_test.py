#!/usr/bin/env python3
"""Measure Python jobs that each sleep for two seconds.

Batch time includes HTTP submission and polling. Queue and worker timings
come from the server's job timestamps; this is not a CPU benchmark.
"""

import argparse
import json
import math
import statistics
import time
from datetime import datetime
from pathlib import Path
from urllib.request import Request, urlopen

BASE = "http://localhost:8080"


def request_json(path, payload=None):
    request = Request(BASE + path)
    if payload is not None:
        request = Request(
            BASE + path,
            data=json.dumps(payload).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
    with urlopen(request, timeout=10) as response:
        return json.load(response)


def timestamp(value):
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("jobs", nargs="?", type=int, default=10)
    args = parser.parse_args()
    if not 1 <= args.jobs <= 50:
        parser.error("choose between 1 and 50 jobs")

    expected = {}
    results = {}
    batch_started = time.monotonic()

    for index in range(1, args.jobs + 1):
        code = (
            "import time\n"
            f"print('Job {index} starting', flush=True)\n"
            "time.sleep(2)\n"
            f"print('Job {index} done')\n"
        )
        submitted = request_json("/api/jobs", {
            "language": "python", "code": code, "stdin": ""
        })
        job_id = submitted["id"]
        if job_id in expected:
            raise SystemExit(f"FAIL: duplicate job ID returned: {job_id}")
        expected[job_id] = f"Job {index} starting\nJob {index} done\n"

    print(f"Submitted {args.jobs} jobs. Waiting for completion...", flush=True)
    pending = set(expected)
    deadline = batch_started + 300

    while pending:
        if time.monotonic() >= deadline:
            raise SystemExit(
                f"FAIL: {len(pending)} jobs unfinished after 300 seconds."
            )
        for job_id in tuple(pending):
            job = request_json(f"/api/jobs/{job_id}")
            if job["status"] not in {"QUEUED", "RUNNING"}:
                results[job_id] = job
                pending.remove(job_id)
        if pending:
            time.sleep(0.5)

    elapsed = time.monotonic() - batch_started
    events, queue_ms, worker_ms, total_ms, failures = [], [], [], [], []

    for job_id, job in results.items():
        if (job["status"] != "COMPLETED" or job.get("exit_code") != 0
                or job.get("stdout") != expected[job_id]):
            failures.append(job_id)

        created = timestamp(job["created_at"])
        started = timestamp(job["started_at"])
        finished = timestamp(job["finished_at"])
        if not created <= started <= finished:
            raise SystemExit(f"FAIL: inconsistent timestamps for {job_id}.")

        queue_ms.append((started - created).total_seconds() * 1000)
        worker_ms.append((finished - started).total_seconds() * 1000)
        total_ms.append((finished - created).total_seconds() * 1000)
        if finished > started:
            events.extend([(started, 1), (finished, -1)])

    active = peak = 0
    # At equal timestamps, process finishes before starts.
    for _, change in sorted(events):
        active += change
        peak = max(peak, active)

    successful = len(results) - len(failures)
    summary = {
        "jobs_submitted": args.jobs,
        "jobs_successful": successful,
        "batch_elapsed_seconds": round(elapsed, 3),
        "successful_jobs_per_second": round(successful / elapsed, 3),
        "peak_overlapping_worker_jobs": peak,
        "mean_queue_wait_ms": round(statistics.mean(queue_ms), 2),
        "mean_worker_time_ms": round(statistics.mean(worker_ms), 2),
        "p95_job_latency_ms": round(
            sorted(total_ms)[math.ceil(0.95 * len(total_ms)) - 1], 2
        ),
    }

    recorded_at = datetime.now().astimezone()
    folder = Path(__file__).resolve().parents[1] / "docs" / "load-tests"
    folder.mkdir(parents=True, exist_ok=True)
    output = folder / f"load-{args.jobs}-{recorded_at:%Y%m%d-%H%M%S-%f}.json"

    report = {
        "recorded_at": recorded_at.isoformat(),
        "workload": "Python: print, sleep 2 seconds, print; one container per job",
        "timing_note": (
            "Batch time includes HTTP/polling; job timings use server timestamps. "
            "Worker time includes container setup and cleanup."
        ),
        "summary": summary,
        "failed_job_ids": failures,
        "jobs": [results[job_id] for job_id in expected],
    }
    output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")

    print(json.dumps(summary, indent=2))
    print(f"Saved: {output}")
    if failures:
        raise SystemExit(
            f"FAIL: {len(failures)} jobs failed the result checks; "
            "inspect the saved report."
        )
    print(f"PASS: {successful}/{args.jobs} jobs completed with the expected output.")


if __name__ == "__main__":
    main()
