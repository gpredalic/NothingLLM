#!/bin/sh
set -e

# The application reads configuration from the database after setup.
# Only POSTGRES_URL is required as an environment variable.
exec /nothingllm