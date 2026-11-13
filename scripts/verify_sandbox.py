#!/usr/bin/env python3
"""Inspect restrictions inside a Python job submitted through the API."""

import json
import time
from datetime import datetime
from pathlib import Path

from verify_queue_full import BASE, request_json, require

PROBE = r'''
import json
import os
import socket
from pathlib import Path

def read(path):
    try:
        return Path(path).read_text().strip()
    except OSError:
        return None

status = {}
for line in (read("/proc/self/status") or "").splitlines():
    key, _, value = line.partition(":")
    if key in {"CapEff", "CapPrm", "CapBnd", "NoNewPrivs", "Seccomp"}:
        status[key] = value.strip()

paths = {
    "memory_v2": "/sys/fs/cgroup/memory.max",
    "cpu_v2": "/sys/fs/cgroup/cpu.max",
    "pids_v2": "/sys/fs/cgroup/pids.max",
    "memory_v1": "/sys/fs/cgroup/memory/memory.limit_in_bytes",
    "quota_v1": "/sys/fs/cgroup/cpu/cpu.cfs_quota_us",
    "period_v1": "/sys/fs/cgroup/cpu/cpu.cfs_period_us",
    "quota_v1_alt": "/sys/fs/cgroup/cpu,cpuacct/cpu.cfs_quota_us",
    "period_v1_alt": "/sys/fs/cgroup/cpu,cpuacct/cpu.cfs_period_us",
    "pids_v1": "/sys/fs/cgroup/pids/pids.max",
}
print(json.dumps({
    "uid": os.getuid(),
    "euid": os.geteuid(),
    "interfaces": sorted(name for _, name in socket.if_nameindex()),
    "interface_flags": {
        name: read("/sys/class/net/" + name + "/flags")
        for _, name in socket.if_nameindex()
    },
    "process_status": status,
    "cgroup_files": {key: read(path) for key, path in paths.items()},
}))
'''


def inspect_probe(probe):
    files = probe["cgroup_files"]
    process = probe["process_status"]

    def first(*keys):
        return next(
            (files[key] for key in keys if files.get(key) is not None),
            None,
        )

    def number(value, base=10):
        try:
            return int(value, base)
        except (TypeError, ValueError):
            return None

    memory = number(first("memory_v2", "memory_v1"))
    pids = number(first("pids_v2", "pids_v1"))
    cpu = None

    if files.get("cpu_v2") is not None:
        parts = files["cpu_v2"].split()
        quota, period = (
            map(number, parts) if len(parts) == 2 else (None, None)
        )
    else:
        quota = number(first("quota_v1", "quota_v1_alt"))
        period = number(first("period_v1", "period_v1_alt"))

    if quota is not None and quota >= 0 and period is not None and period > 0:
        cpu = quota / period

    flags = {
        name: number(probe.get("interface_flags", {}).get(name), 16)
        for name in probe["interfaces"]
    }

    observed = {
        "uid": probe["uid"],
        "euid": probe["euid"],
        "network_interfaces": probe["interfaces"],
        "interface_flags": flags,
        "memory_limit_bytes": memory,
        "cpu_limit_cores": cpu,
        "process_limit": pids,
        "process_status": process,
    }
    checks = {
        "Non-root real and effective UID":
            probe["uid"] > 0 and probe["euid"] > 0,
        "No active non-loopback network interfaces": (
            "lo" in flags
            and all(value is not None for value in flags.values())
            and all(name == "lo" or (value & 1) == 0
                    for name, value in flags.items())
        ),
        "Effective capabilities are empty":
            number(process.get("CapEff"), 16) == 0,
        "Permitted capabilities are empty":
            number(process.get("CapPrm"), 16) == 0,
        "Capability bounding set is empty":
            number(process.get("CapBnd"), 16) == 0,
        "No-new-privileges is enabled":
            process.get("NoNewPrivs") == "1",
        "Seccomp syscall filtering is enabled": process.get("Seccomp") == "2",
        "Memory cgroup limit is 128 MiB":
            memory == 128 * 1024 * 1024,
        "CPU cgroup quota is 0.5 cores":
            cpu == 0.5,
        "Process/thread cgroup limit is 32":
            pids == 32,
    }
    return observed, checks


def main():
    status, health = request_json("/api/health")
    require(status == 200, f"health check returned HTTP {status}")

    status, submitted = request_json("/api/jobs", PROBE)
    require(
        status == 202 and bool(submitted.get("id")),
        f"submission failed: {status}, {submitted}",
    )
    job_id = submitted["id"]
    print(f"Submitted sandbox probe: {job_id}", flush=True)

    deadline = time.monotonic() + 45
    while True:
        require(
            time.monotonic() < deadline,
            "sandbox probe did not finish within 45 seconds",
        )
        status, job = request_json(f"/api/jobs/{job_id}")
        require(status == 200, f"job lookup returned HTTP {status}")
        if job["status"] not in {"QUEUED", "RUNNING"}:
            break
        time.sleep(0.2)

    require(
        job["status"] == "COMPLETED" and job.get("exit_code") == 0,
        f"probe execution failed: {job}",
    )
    probe = json.loads(job["stdout"])
    observed, checks = inspect_probe(probe)
    passed = all(checks.values())

    now = datetime.now().astimezone()
    folder = Path(__file__).resolve().parents[1] / "docs" / "verification"
    folder.mkdir(parents=True, exist_ok=True)
    output = folder / f"sandbox-{now:%Y%m%d-%H%M%S-%f}.json"

    report = {
        "recorded_at": now.isoformat(),
        "test": "runtime restrictions inside an API-submitted Python job",
        "scope": (
            "Inspects identity, network interfaces, capabilities, seccomp, and kernel "
            "cgroup settings. Does not stress-test resource exhaustion "
            "or test sandbox escapes."
        ),
        "base_url": BASE,
        "health": health,
        "job": job,
        "probe": probe,
        "observed": observed,
        "checks": checks,
        "passed": passed,
    }
    output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")

    print(json.dumps(observed, indent=2))
    for name, ok in checks.items():
        print(f"{'PASS' if ok else 'FAIL'}: {name}")
    print(f"Saved: {output}")

    require(passed, "one or more sandbox checks failed; inspect the saved report")
    print("PASS: all sandbox configuration checks passed for this Python job.")


if __name__ == "__main__":
    main()
