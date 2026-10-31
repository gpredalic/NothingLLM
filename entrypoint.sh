#!/bin/sh
set -e

cat > /app/config.yaml <<EOF
server:
  addr: ":8080"

database:
  dsn: "${POSTGRES_URL:?POSTGRES_URL is required}"

qdrant:
  url: "${QDRANT_URL:?QDRANT_URL is required}"
  api_key: "${QDRANT_API_KEY:-}"
  sync_interval: "60s"

openai:
  api_key: "${OPENAI_API_KEY:-sk-placeholder}"
  embedding_model: "text-embedding-3-small"

auth:
  session_ttl: "24h"
  bcrypt_cost: 12

rate_limit:
  requests_per_minute: 100
  login_per_minute: 5
EOF

exec /nothingllm