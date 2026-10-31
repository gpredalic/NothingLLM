#!/bin/bash
set -euo pipefail

# NothingLLM Backup Script
# Usage: ./backup.sh [output_dir]

OUTPUT_DIR="${1:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${OUTPUT_DIR}/nothingllm_${TIMESTAMP}"

# Configuration (override via env vars)
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-nlm}"
PG_DB="${PG_DB:-nothingllm}"
PG_PASSWORD="${PG_PASSWORD:-nlm}"
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
QDRANT_API_KEY="${QDRANT_API_KEY:-}"

mkdir -p "${BACKUP_DIR}"

echo "=== NothingLLM Backup: ${TIMESTAMP} ==="
echo "Output: ${BACKUP_DIR}"

# 1. PostgreSQL dump
echo "[1/3] Backing up PostgreSQL..."
export PGPASSWORD="${PG_PASSWORD}"
pg_dump -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "${PG_DB}" \
    --format=custom --compress=9 --file="${BACKUP_DIR}/postgres.dump"
echo "  -> postgres.dump ($(du -h "${BACKUP_DIR}/postgres.dump" | cut -f1))"

# 2. Qdrant snapshot
echo "[2/3] Creating Qdrant snapshot..."
SNAPSHOT_RESP=$(curl -s -X POST "${QDRANT_URL}/snapshots" \
    ${QDRANT_API_KEY:+-H "api-key: ${QDRANT_API_KEY}"} \
    -d '{"wait": true}' 2>/dev/null) || true

if [ -n "${SNAPSHOT_RESP}" ] && echo "${SNAPSHOT_RESP}" | grep -q '"snapshot_name"'; then
    SNAPSHOT_NAME=$(echo "${SNAPSHOT_RESP}" | grep -oP '"snapshot_name"\s*:\s*"\K[^"]+')
    echo "  Snapshot: ${SNAPSHOT_NAME}"
    curl -s -o "${BACKUP_DIR}/qdrant_snapshot.tar" \
        "${QDRANT_URL}/snapshots/${SNAPSHOT_NAME}" \
        ${QDRANT_API_KEY:+-H "api-key: ${QDRANT_API_KEY}"}
    echo "  -> qdrant_snapshot.tar ($(du -h "${BACKUP_DIR}/qdrant_snapshot.tar" | cut -f1))"
    # Cleanup snapshot on Qdrant after download
    curl -s -X DELETE "${QDRANT_URL}/snapshots/${SNAPSHOT_NAME}" \
        ${QDRANT_API_KEY:+-H "api-key: ${QDRANT_API_KEY}"} 2>/dev/null || true
else
    echo "  WARNING: Qdrant snapshot failed (service may be down). Skipping."
    echo "qdrant_unavailable" > "${BACKUP_DIR}/qdrant_status.txt"
fi

# 3. Config export (non-secret)
echo "[3/3] Exporting configuration summary..."
cat > "${BACKUP_DIR}/manifest.json" <<EOF
{
  "backup_time": "$(date -Iseconds)",
  "postgres": {"host": "${PG_HOST}", "port": "${PG_PORT}", "db": "${PG_DB}"},
  "qdrant": {"url": "${QDRANT_URL}"},
  "files": ["postgres.dump", "qdrant_snapshot.tar"]
}
EOF

echo "=== Backup complete: ${BACKUP_DIR} ==="

# Retention: keep last 7 backups
LATEST=$(ls -td "${OUTPUT_DIR}"/nothingllm_* 2>/dev/null | tail -n +8)
if [ -n "${LATEST}" ]; then
    echo "Cleaning up old backups:"
    echo "${LATEST}" | xargs -I{} rm -rf {}
fi