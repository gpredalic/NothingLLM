#!/bin/bash
set -euo pipefail

# NothingLLM Load Test
# Tests login, chat, and search endpoints with concurrent requests.
# Usage: ./scripts/loadtest.sh [api_url] [concurrency] [duration_seconds]

API_URL="${1:-http://localhost:8080}"
CONCURRENCY="${2:-100}"
DURATION="${3:-30}"

echo "=== NothingLLM Load Test ==="
echo "Target: ${API_URL}"
echo "Concurrency: ${CONCURRENCY}"
echo "Duration: ${DURATION}s"
echo ""

# Create test user (ignore if exists)
echo "[Setup] Creating test user..."
curl -s -X POST "${API_URL}/api/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"loadtest@example.com","password":"LoadTest123!"}' > /dev/null 2>&1 || true

# Login and get session cookie
echo "[Setup] Authenticating..."
COOKIE_JAR=$(mktemp)
trap "rm -f ${COOKIE_JAR}" EXIT

LOGIN_RESP=$(curl -s -c "${COOKIE_JAR}" -X POST "${API_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"loadtest@example.com","password":"LoadTest123!"}')

if echo "${LOGIN_RESP}" | grep -q '"status":"ok"'; then
    echo "  Authenticated successfully."
else
    echo "  WARNING: Login failed: ${LOGIN_RESP}"
    echo "  Running unauthenticated load test on /health only."
fi

# Health endpoint load test
echo ""
echo "[Test 1] Health endpoint (${CONCURRENCY} concurrent)..."
AB_OUTPUT=$(ab -n $((CONCURRENCY * 10)) -c "${CONCURRENCY}" "${API_URL}/health" 2>&1) || true
echo "${AB_OUTPUT}" | grep -E "(Requests per second|Time per request|Failed|Complete)"

# Login endpoint load test (rate-limited, smaller)
echo ""
echo "[Test 2] Login endpoint (${CONCURRENCY} concurrent)..."
LOGIN_AB=$(ab -n "${CONCURRENCY}" -c "${CONCURRENCY}" \
    -p <(echo -n '{"email":"loadtest@example.com","password":"LoadTest123!"}') \
    -T "application/json" \
    -X POST \
    "${API_URL}/api/auth/login" 2>&1) || true
echo "${LOGIN_AB}" | grep -E "(Requests per second|Time per request|Failed|Complete|503|429)"

# Search endpoint load test (authenticated)
echo ""
echo "[Test 3] Search endpoint (${CONCURRENCY} concurrent)..."
if [ -f "${COOKIE_JAR}" ] && [ -s "${COOKIE_JAR}" ]; then
    SEARCH_AB=$(ab -n $((CONCURRENCY * 5)) -c "${CONCURRENCY}" \
        -p <(echo -n '{"query":"test"}') \
        -T "application/json" \
        -X POST \
        -A "$(grep -v '^#' "${COOKIE_JAR}" | awk '{print $7"="$8}' | head -1)" \
        "${API_URL}/api/workspaces" 2>&1) || true
    echo "${SEARCH_AB}" | grep -E "(Requests per second|Time per request|Failed|Complete)"
else
    echo "  Skipped (no session cookie)."
fi

echo ""
echo "=== Load Test Complete ==="
echo ""
echo "Bottleneck Analysis:"
echo "  - If login 429s > 0: rate limiter is working (expected)"
echo "  - If p95 > 200ms: consider connection pool tuning"
echo "  - If errors > 0: check DB pool size and OpenAI timeout"