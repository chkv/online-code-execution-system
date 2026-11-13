#!/usr/bin/env python3
"""Check all four languages after enabling Docker syscall filtering."""

import json
import subprocess
import time
from datetime import datetime
from pathlib import Path
from urllib.request import Request, urlopen

from verify_queue_full import BASE, request_json, require

PROGRAMS = {
    "python": 'print("hello python")\n',
    "go": 'package main\nimport "fmt"\nfunc main() { fmt.Println("hello go") }\n',
    "cpp": '#include <iostream>\nint main() { std::cout << "hello cpp\\n"; return 0; }\n',
    "java": 'public class Main { public static void main(String[] args) { System.out.println("hello java"); } }\n',
}


def main():
    status, health = request_json("/api/health")
    require(status == 200, f"health check returned HTTP {status}")

    security = json.loads(subprocess.check_output(
        ["docker", "info", "--format", "{{json .SecurityOptions}}"],
        text=True, timeout=20,
    ))
    require(
        "name=seccomp,profile=builtin" in security,
        f"unexpected Docker security settings: {security}",
    )

    results = []
    for language, code in PROGRAMS.items():
        request = Request(
            BASE + "/api/jobs",
            data=json.dumps({
                "language": language, "code": code, "stdin": ""
            }).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urlopen(request, timeout=10) as response:
            require(
                response.status == 202,
                f"{language}: submission returned HTTP {response.status}",
            )
            submitted = json.load(response)

        job_id = submitted["id"]
        deadline = time.monotonic() + 45
        while True:
            require(
                time.monotonic() < deadline,
                f"{language}: result not received within 45 seconds",
            )
            status, job = request_json(f"/api/jobs/{job_id}")
            require(status == 200, f"{language}: lookup returned HTTP {status}")
            if job["status"] not in {"QUEUED", "RUNNING"}:
                break
            time.sleep(0.2)

        passed = (
            job["status"] == "COMPLETED"
            and job.get("exit_code") == 0
            and job.get("stdout") == f"hello {language}\n"
        )
        results.append({"language": language, "passed": passed, "job": job})
        print(
            f"{'PASS' if passed else 'FAIL'}: {language} | {job['status']} | "
            f"exit={job.get('exit_code')} | time={job.get('execution_time_ms')}ms",
            flush=True,
        )
        if not passed:
            print(json.dumps(job, indent=2), flush=True)

    now = datetime.now().astimezone()
    folder = Path(__file__).resolve().parents[1] / "docs" / "verification"
    folder.mkdir(parents=True, exist_ok=True)
    output = folder / f"languages-{now:%Y%m%d-%H%M%S-%f}.json"
    passed = all(result["passed"] for result in results)

    report = {
        "recorded_at": now.isoformat(),
        "test": "four-language smoke test after enabling Docker builtin seccomp",
        "base_url": BASE,
        "health": health,
        "docker_security_options": security,
        "results": results,
        "passed": passed,
    }
    output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"Saved: {output}")

    require(passed, "one or more language checks failed; inspect the saved report")
    print("PASS: 4/4 language checks passed with Docker's builtin seccomp profile.")


if __name__ == "__main__":
    main()
