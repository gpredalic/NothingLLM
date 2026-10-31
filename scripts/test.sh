#!/bin/bash
set -euo pipefail

# NothingLLM Test Script
# Runs all Go tests and fails on first error.
# Usage: ./scripts/test.sh

cd "$(dirname "$0")/.."

echo "=== Running Go Tests ==="
go test ./... -count=1

echo ""
echo "=== All tests passed ==="