#!/usr/bin/env bash
set -euo pipefail

cd /vagrant

echo "[INFO] Building Docker image gin-demo:latest..."
docker build --no-cache -t gin-demo:latest .

echo "[INFO] Stopping old container (if any)..."
docker rm -f gin-demo 2>/dev/null || true

echo "[INFO] Running new gin-demo container..."
docker run -d \
  --name gin-demo \
  -p 8080:8080 \
  gin-demo:latest

echo "[INFO] Waiting 2 seconds..."
sleep 2

echo "[INFO] Container status:"
docker ps --filter "name=gin-demo"

echo "[INFO] Last 20 log lines:"
docker logs --tail 20 gin-demo || true

