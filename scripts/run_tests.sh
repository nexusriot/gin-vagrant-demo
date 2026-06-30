#!/usr/bin/env bash
set -euo pipefail

cd /vagrant

echo "[INFO] go vet ./..."
go vet ./...

echo "[INFO] go test -race ./..."
go test -race ./...

if command -v govulncheck &>/dev/null; then
  echo "[INFO] govulncheck ./..."
  govulncheck ./...
else
  echo "[WARN] govulncheck not found; skipping (install: go install golang.org/x/vuln/cmd/govulncheck@latest)"
fi
