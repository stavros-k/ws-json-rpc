#!/bin/bash
set -e
go build -o ./data/server ./backend/cmd/server
LOG_LEVEL=debug GENERATE=true ./data/server

sqlc vet
sqlc compile
sqlc generate

DOCS_BASE_PATH=/docs npm run docs:build

npm run fmt

go build -o ./data/server ./backend/cmd/server
