#!/bin/bash
# Demo script to submit multiple jobs concurrently to show workers processing in parallel.

URL="http://localhost:8080/api/jobs"

echo "Submitting jobs..."

curl -s -X POST $URL -H "Content-Type: application/json" -d '{"language":"python","code":"import time\nprint(\"Job 1 starting\")\ntime.sleep(2)\nprint(\"Job 1 done\")"}' &
curl -s -X POST $URL -H "Content-Type: application/json" -d '{"language":"python","code":"import time\nprint(\"Job 2 starting\")\ntime.sleep(2)\nprint(\"Job 2 done\")"}' &
curl -s -X POST $URL -H "Content-Type: application/json" -d '{"language":"python","code":"import time\nprint(\"Job 3 starting\")\ntime.sleep(2)\nprint(\"Job 3 done\")"}' &
curl -s -X POST $URL -H "Content-Type: application/json" -d '{"language":"python","code":"import time\nprint(\"Job 4 starting\")\ntime.sleep(2)\nprint(\"Job 4 done\")"}' &
curl -s -X POST $URL -H "Content-Type: application/json" -d '{"language":"python","code":"import time\nprint(\"Job 5 starting\")\ntime.sleep(2)\nprint(\"Job 5 done\")"}' &

wait
echo "All 5 jobs submitted! Watch the server logs to see workers pick them up concurrently."
