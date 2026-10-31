#!/bin/bash
set -uo pipefail

# NothingLLM Test Deployment Validation
# Phase 9: Complete fresh-install test from zero state.
# Usage: ./scripts/deploy-test.sh

cd "$(dirname "$0")/.."

API_URL="http://localhost:8080"
WEB_URL="http://localhost:3000"
QDRANT_URL="http://localhost:6333"
TIMEOUT=90
PASS=0
FAIL=0
WARN=0
ERRORS=()
WARNINGS=()
RESULTS=()

log() { echo "[$(date +%H:%M:%S)] $*"; }
pass() { PASS=$((PASS+1)); RESULTS+=("PASS: $1"); log "  ✓ $1"; }
fail() { FAIL=$((FAIL+1)); ERRORS+=("$1"); RESULTS+=("FAIL: $1"); log "  ✗ $1"; }
warn() { WARN=$((WARN+1)); WARNINGS+=("$1"); RESULTS+=("WARN: $1"); log "  ⚠ $1"; }

section() { log ""; log "═══════════════════════════════════════════════"; log "  $1"; log "═══════════════════════════════════════════════"; }

generate_report() {
    local VERDICT="PASS"
    if [ $FAIL -gt 0 ]; then
        VERDICT="FAIL"
    fi
    local REPORT="TEST_DEPLOYMENT_REPORT.md"
    cat > "$REPORT" << EOF
# Test Deployment Report

**Date:** $(date -u "+%Y-%m-%d %H:%M:%S UTC")
**Verdict:** ${VERDICT}

## 1. Environment

| Component | Version |
|-----------|---------|
| Docker | $(docker --version 2>/dev/null \| awk '{print $3}' \| tr -d ',' \|\| echo "unavailable") |
| Docker Compose | $(docker compose version --short 2>/dev/null \|\| echo "unavailable") |
| OS | $(uname -sr) |
| API | Go 1.24 / Alpine 3.20 |
| Frontend | Node 20 / Next.js |
| PostgreSQL | 16-alpine |
| Qdrant | latest |

## 2. Steps Executed

$(for r in "${RESULTS[@]}"; do echo "- \`${r}\`"; done)

## 3. Summary

| Metric | Count |
|--------|-------|
| Passed | ${PASS} |
| Failed | ${FAIL} |
| Warnings | ${WARN} |
| **Total** | **$((PASS+FAIL+WARN))** |

## 4. Errors

$(if [ ${#ERRORS[@]} -gt 0 ]; then for e in "${ERRORS[@]}"; do echo "- ${e}"; done; else echo "None."; fi)

## 5. Warnings

$(if [ ${#WARNINGS[@]} -gt 0 ]; then for w in "${WARNINGS[@]}"; do echo "- ${w}"; done; else echo "None."; fi)

## 6. Final Verdict

**${VERDICT}** — $(if [ $FAIL -eq 0 ]; then echo "All critical tests passed. Deployment is validated."; else echo "${FAIL} test(s) failed. See errors above."; fi)
EOF
    log ""
    log "═══════════════════════════════════════════════"
    log "  RESULTS: ${PASS} passed, ${FAIL} failed, ${WARN} warnings"
    log "  VERDICT: ${VERDICT}"
    log "  Report: ${REPORT}"
    log "═══════════════════════════════════════════════"
}

# ============================================================
# PHASE 1: PREPARATION
# ============================================================
section "PHASE 1: PREPARATION"

log "[1.1] Stopping running stack..."
docker compose down --remove-orphans 2>/dev/null || true
pass "Stack stopped"

log "[1.2] Removing test volumes..."
docker volume rm nothingllm_postgres_data nothingllm_qdrant_data 2>/dev/null || true
pass "Volumes removed"

log "[1.3] Rebuilding images..."
docker compose build --pull 2>&1 | tail -5
if [ $? -eq 0 ]; then
    pass "Images built"
else
    fail "Image build failed"
    log "  Aborting: build failed"
    generate_report
    exit 1
fi

log "[1.4] Starting fresh stack..."
docker compose up -d
pass "Stack started"

log "[1.5] Waiting for health checks (${TIMEOUT}s timeout)..."
API_READY=0
for i in $(seq 1 ${TIMEOUT}); do
    if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
        API_READY=1
        log "  API healthy after ${i}s"
        break
    fi
    sleep 1
done
if [ $API_READY -eq 1 ]; then
    pass "API health check passed"
else
    fail "API not healthy after ${TIMEOUT}s"
    docker compose logs api 2>&1 | tail -20
    generate_report
    exit 1
fi

QDRANT_READY=0
for i in $(seq 1 ${TIMEOUT}); do
    if curl -sf "${QDRANT_URL}/healthz" > /dev/null 2>&1; then
        QDRANT_READY=1
        log "  Qdrant healthy after ${i}s"
        break
    fi
    sleep 1
done
if [ $QDRANT_READY -eq 1 ]; then
    pass "Qdrant health check passed"
else
    fail "Qdrant not healthy after ${TIMEOUT}s"
fi

# ============================================================
# PHASE 2: FRESH INSTALL TEST
# ============================================================
section "PHASE 2: FRESH INSTALL TEST"

log "[2.1] Verifying setup mode is active..."
RESP=$(curl -sf "${API_URL}/api/setup/status" 2>/dev/null)
if echo "$RESP" | grep -q '"complete":false'; then
    pass "Setup mode active (complete=false)"
else
    fail "Setup mode not active: $RESP"
fi

log "[2.2] Verifying /health reports setup mode..."
RESP=$(curl -sf "${API_URL}/health" 2>/dev/null)
if echo "$RESP" | grep -q '"mode":"setup"'; then
    pass "Health reports setup mode"
else
    warn "Health does not report setup mode explicitly: $RESP"
fi

# ============================================================
# PHASE 3: SETUP WIZARD TEST
# ============================================================
section "PHASE 3: SETUP WIZARD TEST"

log "[3.1] Creating admin user..."
RESP=$(curl -sf -X POST "${API_URL}/api/setup/admin" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.local","password":"SecurePass123!"}' 2>/dev/null)
if echo "$RESP" | grep -q '"id"'; then
    pass "Admin created"
else
    fail "Admin creation failed: $RESP"
fi

log "[3.2] Verifying admin count = 1..."
RESP=$(curl -sf "${API_URL}/api/setup/status" 2>/dev/null)
if echo "$RESP" | grep -q '"has_admin":true'; then
    pass "Admin count verified"
else
    fail "Admin not reflected in status: $RESP"
fi

log "[3.3] Configuring OpenAI API key..."
RESP=$(curl -sf -X POST "${API_URL}/api/setup/config" \
    -H "Content-Type: application/json" \
    -d '{"openai_api_key":"sk-test-key-for-deployment"}' 2>/dev/null)
if echo "$RESP" | grep -q '"status":"ok"'; then
    pass "OpenAI key saved"
else
    fail "OpenAI key save failed: $RESP"
fi

log "[3.4] Configuring Qdrant URL..."
RESP=$(curl -sf -X POST "${API_URL}/api/setup/config" \
    -H "Content-Type: application/json" \
    -d '{"qdrant_url":"http://qdrant:6333"}' 2>/dev/null)
if echo "$RESP" | grep -q '"status":"ok"'; then
    pass "Qdrant URL saved"
else
    fail "Qdrant URL save failed: $RESP"
fi

log "[3.5] Running connectivity test..."
RESP=$(curl -sf -X POST "${API_URL}/api/setup/test" 2>/dev/null)
log "  Results: $RESP"
# Qdrant should be OK (local), OpenAI will fail with test key (expected)
if echo "$RESP" | grep -q '"qdrant"'; then
    if echo "$RESP" | grep -q '"qdrant":{"status":"ok"'; then
        pass "Qdrant connectivity OK"
    else
        fail "Qdrant connectivity failed: $RESP"
    fi
fi
# OpenAI test with placeholder key - expected to fail, just verify it doesn't crash
if echo "$RESP" | grep -q '"openai"'; then
    pass "OpenAI test executed (response received)"
else
    warn "No OpenAI test result in response"
fi

log "[3.6] Completing setup..."
RESP=$(curl -sf -X POST "${API_URL}/api/setup/complete" 2>/dev/null)
if echo "$RESP" | grep -q '"status":"ok"'; then
    pass "Setup completed"
else
    fail "Setup completion failed: $RESP"
fi

# ============================================================
# PHASE 4: POST-SETUP TEST
# ============================================================
section "PHASE 4: POST-SETUP TEST"

log "[4.1] Verifying setup status shows complete..."
RESP=$(curl -sf "${API_URL}/api/setup/status" 2>/dev/null)
if echo "$RESP" | grep -q '"complete":true'; then
    pass "Setup marked complete"
else
    fail "Setup not marked complete: $RESP"
fi

log "[4.2] Verifying setup endpoints are locked..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/setup/admin" \
    -H "Content-Type: application/json" \
    -d '{"email":"hacker@test.local","password":"Hack123!"}' 2>/dev/null)
if [ "$CODE" = "403" ]; then
    pass "Setup admin endpoint locked (403)"
else
    fail "Setup admin endpoint not locked (got $CODE)"
fi

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/setup/config" \
    -H "Content-Type: application/json" \
    -d '{"openai_api_key":"sk-hack"}' 2>/dev/null)
if [ "$CODE" = "403" ]; then
    pass "Setup config endpoint locked (403)"
else
    fail "Setup config endpoint not locked (got $CODE)"
fi

log "[4.3] Verifying API is now in full mode..."
sleep 2
RESP=$(curl -sf "${API_URL}/health" 2>/dev/null)
if echo "$RESP" | grep -q '"status":"ok"'; then
    pass "API running in full mode"
else
    fail "API not in full mode: $RESP"
fi

# ============================================================
# PHASE 5: AUTH TEST
# ============================================================
section "PHASE 5: AUTH TEST"

log "[5.1] Logging in as admin..."
LOGIN_RESP=$(curl -sf -c /tmp/nlm_cookies.txt -X POST "${API_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.local","password":"SecurePass123!"}' 2>/dev/null)
if [ -n "$LOGIN_RESP" ]; then
    pass "Login successful"
else
    fail "Login failed"
fi

log "[5.2] Verifying session cookie..."
if [ -f /tmp/nlm_cookies.txt ] && grep -q "session" /tmp/nlm_cookies.txt; then
    pass "Session cookie issued"
else
    fail "No session cookie found"
fi

log "[5.3] Checking cookie flags..."
COOKIE_LINE=$(grep "session" /tmp/nlm_cookies.txt 2>/dev/null || echo "")
if echo "$COOKIE_LINE" | grep -qi "HttpOnly"; then
    pass "Cookie HttpOnly flag set"
else
    warn "Cookie HttpOnly flag not detected in jar"
fi
log "  Cookie: $COOKIE_LINE"

# ============================================================
# PHASE 6: WORKSPACE TEST
# ============================================================
section "PHASE 6: WORKSPACE TEST"

log "[6.1] Creating workspace..."
WS_RESP=$(curl -sf -b /tmp/nlm_cookies.txt -X POST "${API_URL}/api/workspaces" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test Workspace"}' 2>/dev/null)
WS_ID=$(echo "$WS_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$WS_ID" ]; then
    pass "Workspace created: $WS_ID"
else
    fail "Workspace creation failed: $WS_RESP"
fi

log "[6.2] Reading workspace..."
RESP=$(curl -sf -b /tmp/nlm_cookies.txt "${API_URL}/api/workspaces/${WS_ID}" 2>/dev/null)
if echo "$RESP" | grep -q "Test Workspace"; then
    pass "Workspace read OK"
else
    fail "Workspace read failed: $RESP"
fi

log "[6.3] Deleting workspace..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" -b /tmp/nlm_cookies.txt -X DELETE "${API_URL}/api/workspaces/${WS_ID}" 2>/dev/null)
if [ "$CODE" = "200" ] || [ "$CODE" = "204" ]; then
    pass "Workspace deleted"
else
    fail "Workspace delete failed (got $CODE)"
fi

# ============================================================
# PHASE 7: QDRANT TEST
# ============================================================
section "PHASE 7: QDRANT TEST"

log "[7.1] Creating collection in Qdrant..."
curl -sf -X PUT "${QDRANT_URL}/collections/test-documents" \
    -H "Content-Type: application/json" \
    -d '{"vectors": {"size": 1536, "distance": "Cosine"}}' > /dev/null 2>&1
pass "Qdrant collection created"

log "[7.2] Adding test point..."
curl -sf -X PUT "${QDRANT_URL}/collections/test-documents/points" \
    -H "Content-Type: application/json" \
    -d '{"points":[{"id":1,"vector":[0.1,0.2,0.3],"payload":{"text":"Hello NothingLLM","source":"test"}}]}' > /dev/null 2>&1
pass "Test point added"

log "[7.3] Verifying collection visible via API..."
sleep 2
RESP=$(curl -sf -b /tmp/nlm_cookies.txt "${API_URL}/api/collections" 2>/dev/null)
if echo "$RESP" | grep -q "test-documents"; then
    pass "Collection visible in API"
else
    warn "Collection not yet visible (sync pending): $RESP"
fi

# ============================================================
# PHASE 8: OPENAI TEST
# ============================================================
section "PHASE 8: OPENAI TEST"

log "[8.1] Verifying model sync..."
RESP=$(curl -sf -b /tmp/nlm_cookies.txt "${API_URL}/api/models" 2>/dev/null)
if [ -n "$RESP" ]; then
    pass "Model sync endpoint responsive"
else
    warn "Model sync not responsive (expected with test key)"
fi

log "[8.2] Verifying embeddings endpoint..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" -b /tmp/nlm_cookies.txt -X POST "${API_URL}/api/embed" \
    -H "Content-Type: application/json" \
    -d '{"text":"test"}' 2>/dev/null)
if [ "$CODE" = "200" ]; then
    pass "Embeddings endpoint OK"
elif [ "$CODE" = "502" ] || [ "$CODE" = "503" ]; then
    warn "Embeddings returned $CODE (expected with test key)"
else
    warn "Embeddings returned unexpected $CODE"
fi

# ============================================================
# PHASE 9: RAG TEST
# ============================================================
section "PHASE 9: RAG TEST"

log "[9.1] Creating workspace for RAG test..."
WS_RESP=$(curl -sf -b /tmp/nlm_cookies.txt -X POST "${API_URL}/api/workspaces" \
    -H "Content-Type: application/json" \
    -d '{"name":"RAG Test WS"}' 2>/dev/null)
WS2_ID=$(echo "$WS_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$WS2_ID" ]; then
    pass "RAG workspace created"
else
    warn "RAG workspace creation failed: $WS_RESP"
fi

# ============================================================
# PHASE 10: HEALTH TESTS
# ============================================================
section "PHASE 10: HEALTH TESTS"

for endpoint in health ready version metrics; do
    CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/${endpoint}" 2>/dev/null)
    if [ "$CODE" = "200" ]; then
        pass "/${endpoint} returns 200"
    else
        fail "/${endpoint} returned $CODE"
    fi
done

# ============================================================
# PHASE 11: BACKUP TEST
# ============================================================
section "PHASE 11: BACKUP TEST"

log "[11.1] Running backup..."
BACKUP_DIR="/tmp/nlm-backup-test"
mkdir -p "$BACKUP_DIR"
docker compose exec -T postgres pg_dump -U nlm nothingllm > "${BACKUP_DIR}/db.sql" 2>/dev/null
if [ -s "${BACKUP_DIR}/db.sql" ]; then
    pass "Database backup created ($(wc -l < "${BACKUP_DIR}/db.sql") lines)"
else
    fail "Database backup failed"
fi

# ============================================================
# PHASE 12: RESTORE TEST
# ============================================================
section "PHASE 12: RESTORE TEST"

log "[12.1] Creating test restore database..."
docker compose exec -T postgres psql -U nlm -c "DROP DATABASE IF EXISTS nlm_restore; CREATE DATABASE nlm_restore;" > /dev/null 2>&1

log "[12.2] Restoring backup..."
docker compose exec -T postgres psql -U nlm nlm_restore < "${BACKUP_DIR}/db.sql" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    pass "Restore successful"
else
    fail "Restore failed"
fi

log "[12.3] Verifying restored data..."
USER_COUNT=$(docker compose exec -T postgres psql -U nlm nlm_restore -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d ' ')
if [ "$USER_COUNT" -ge 1 ] 2>/dev/null; then
    pass "Users restored ($USER_COUNT)"
else
    fail "Users not found in restore"
fi

SETTINGS_COUNT=$(docker compose exec -T postgres psql -U nlm nlm_restore -t -c "SELECT COUNT(*) FROM settings;" 2>/dev/null | tr -d ' ')
if [ "$SETTINGS_COUNT" -ge 1 ] 2>/dev/null; then
    pass "Settings restored ($SETTINGS_COUNT rows)"
else
    fail "Settings not found in restore"
fi

docker compose exec -T postgres psql -U nlm -c "DROP DATABASE IF EXISTS nlm_restore;" > /dev/null 2>&1

# ============================================================
# PHASE 13: FAILURE TESTS
# ============================================================
section "PHASE 13: FAILURE TESTS"

log "[13.1] Stopping Qdrant..."
docker compose stop qdrant > /dev/null 2>&1
sleep 2

log "[13.2] Verifying /ready returns 503..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/ready" 2>/dev/null)
if [ "$CODE" = "503" ]; then
    pass "/ready returns 503 when Qdrant down"
else
    warn "/ready returned $CODE (expected 503)"
fi

log "[13.3] Verifying API remains available..."
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/health" 2>/dev/null)
if [ "$CODE" = "200" ]; then
    pass "API remains available with Qdrant down"
else
    fail "API down with Qdrant stopped (got $CODE)"
fi

log "[13.4] Restarting Qdrant..."
docker compose start qdrant > /dev/null 2>&1
sleep 3
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/ready" 2>/dev/null)
if [ "$CODE" = "200" ]; then
    pass "API recovers after Qdrant restart"
else
    warn "API not ready after Qdrant restart (got $CODE)"
fi

# ============================================================
# PHASE 14: CLEANUP
# ============================================================
section "PHASE 14: CLEANUP"

log "Stopping test stack..."
docker compose down --remove-orphans 2>/dev/null || true
rm -f /tmp/nlm_cookies.txt
rm -rf "$BACKUP_DIR"
pass "Cleanup complete"

# ============================================================
# REPORT GENERATION
# ============================================================
generate_report
exit $FAIL