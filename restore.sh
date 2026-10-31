#!/bin/bash
set -euo pipefail

# NothingLLM Restore Script
# Usage: ./restore.sh <backup_dir>
#
# Restores PostgreSQL and Qdrant from a backup created by backup.sh

BACKUP_DIR="${1:?Usage: ./restore.sh <backup_dir>}"

# Configuration
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-nlm}"
PG_DB="${PG_DB:-nothingllm}"
PG_PASSWORD="${PG_PASSWORD:-nlm}"
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
QDRANT_API_KEY="${QDRANT_API_KEY:-}"

if [ ! -d "${BACKUP_DIR}" ]; then
    echo "ERROR: Backup directory not found: ${BACKUP_DIR}"
    exit 1
fi

echo "=== NothingLLM Restore ==="
echo "Source: ${BACKUP_DIR}"

# Confirm
echo "This will DROP and recreate all data in '${PG_DB}'."
read -p "Continue? (yes/no): " CONFIRM
if [ "${CONFIRM}" != "yes" ]; then
    echo "Aborted."
    exit 0
fi

# 1. Restore PostgreSQL
PG_DUMP="${BACKUP_DIR}/postgres.dump"
if [ -f "${PG_DUMP}" ]; then
    echo "[1/2] Restoring PostgreSQL..."
    export PGPASSWORD="${PG_PASSWORD}"
    # Drop and recreate database
    psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d postgres \
        -c "DROP DATABASE IF EXISTS ${PG_DB};"
    psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d postgres \
        -c "CREATE DATABASE ${PG_DB};"
    pg_restore -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "${PG_DB}" \
        --no-owner --clean --if-exists "${PG_DUMP}"
    echo "  PostgreSQL restored."
else
    echo "  WARNING: postgres.dump not found. Skipping DB restore."
fi

# 2. Restore Qdrant snapshot
QDRANT_SNAP="${BACKUP_DIR}/qdrant_snapshot.tar"
if [ -f "${QDRANT_SNAP}" ]; then
    echo "[2/2] Restoring Qdrant snapshot..."
    SNAP_NAME="restore_$(date +%s).tar"

    # Upload snapshot to Qdrant
    curl -s -X POST "${QDRANT_URL}/snapshots/upload" \
        ${QDRANT_API_KEY:+-H "api-key: ${QDRANT_API_KEY}"} \
        -F "file=@${QDRANT_SNAP}" -F "filename=${SNAP_NAME}" 2>/dev/null || true

    # Attempt to restore from uploaded snapshot
    curl -s -X POST "${QDRANT_URL}/snapshots/upload" \
        ${QDRANT_API_KEY:+-H "api-key: ${QDRANT_API_KEY}"} \
        -d "{\"location\": \"${SNAP_NAME}\", \"delete_after_restore\": true}" 2>/dev/null || true

    echo "  Qdrant snapshot restore initiated (check Qdrant logs for status)."
else
    echo "  WARNING: qdrant_snapshot.tar not found. Skipping Qdrant restore."
fi

echo "=== Restore complete ==="
echo "Verify: curl ${QDRANT_URL}/collections && psql -d ${PG_DB} -c 'SELECT COUNT(*) FROM users;'"