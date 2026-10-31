#!/bin/bash
set -euo pipefail

# NothingLLM Docker Test Script
# Builds, starts, smoke-tests, and shuts down the full stack.
# Usage: ./scripts/test-docker.sh

cd "$(dirname "$0")/.."

API_URL="http://localhost:8080"
WEB_URL="http://localhost:3000"
TIMEOUT=60

cleanup() {
    echo ""
    echo "=== Shutting down ==="
    docker compose down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Docker Compose Test ==="

# 1. Validate config
echo "[1/5] Validating docker compose config..."
docker compose config --quiet
echo "  Config valid."

# 2. Build
echo "[2/5] Building images..."
docker compose build
echo "  Build complete."

# 3. Start
echo "[3/5] Starting stack..."
docker compose up -d
echo "  Stack starting..."

# 4. Wait for health
echo "[4/5] Waiting for health checks (${TIMEOUT}s timeout)..."
for i in $(seq 1 ${TIMEOUT}); do
    if curl -sf "${API_URL}/ready" > /dev/null 2>&1; then
        echo "  API ready after ${i}s."
        break
    fi
    if [ "$i" -eq "${TIMEOUT}" ]; then
        echo "  FAILURE: API not ready after ${TIMEOUT}s"
        docker compose logs api 2>&1 | tail -20
        exit 1
    fi
    sleep 1
done

# 5. Smoke tests
echo "[5/5] Running smoke tests..."

# /health
RESP=$(curl -sf "${API_URL}/health")
echo "${RESP}" | grep -q '"status":"ok"'
echo "  /health: OK"

# /ready
RESP=$(curl -sf "${API_URL}/ready")
echo "${RESP}" | grep -q '"status":"ready"'
echo "  /ready: OK"

# /version
RESP=$(curl -sf "${API_URL}/version")
echo "${RESP}" | grep -q '"version"'
echo "${RESP}" | grep -q '"git_commit"'
echo "${RESP}" | grep -q '"build_date"'
echo "  /version: OK"

# /metrics
RESP=$(curl -sf "${API_URL}/metrics")
echo "${RESP}" | grep -q '"users"'
echo "${RESP}" | grep -q '"workspaces"'
echo "${RESP}" | grep -q '"collections"'
echo "${RESP}" | grep -q '"active_sessions"'
echo "  /metrics: OK"

echo ""
echo "=== All Docker tests passed ==="