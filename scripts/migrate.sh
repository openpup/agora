#!/usr/bin/env sh
set -eu

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-openpup}"
DB_PASSWORD="${DB_PASSWORD:-dev_password}"
DB_NAME="${DB_NAME:-agora}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

DATABASE_URL="${DATABASE_URL:-postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}}"

migrate -path migrations -database "$DATABASE_URL" "$@"
