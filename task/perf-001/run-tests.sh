#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-perf-001}"

echo "Running tests for task ${TASK_ID}..."

# Run go tests (adjust -v if you need more verbosity)
go test ./... -run Test -v
