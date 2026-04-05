#!/usr/bin/env sh
set -eu
migrate -path migrations -database "postgres://openpup:dev_password@localhost:5432/agora?sslmode=disable" "$@"
