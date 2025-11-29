
#!/usr/bin/env bash
set -euo pipefail

cd /vagrant

echo "[INFO] Running go test ./..."
go test ./...