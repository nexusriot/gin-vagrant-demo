#!/usr/bin/env bash
set -euo pipefail

cd /vagrant

# Derive version metadata from git when available.
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "[INFO] Building Docker image gin-demo:latest (version=${VERSION} commit=${COMMIT})..."
docker build --no-cache \
  --build-arg "VERSION=${VERSION}" \
  --build-arg "COMMIT=${COMMIT}" \
  --build-arg "BUILD_TIME=${BUILD_TIME}" \
  -t gin-demo:latest .

echo "[INFO] Stopping old container (if any)..."
docker rm -f gin-demo 2>/dev/null || true

echo "[INFO] Running new gin-demo container..."
docker run -d \
  --name gin-demo \
  --restart unless-stopped \
  -p 8080:8080 \
  gin-demo:latest

echo "[INFO] Waiting 2 seconds..."
sleep 2

echo "[INFO] Container status:"
docker ps --filter "name=gin-demo"

echo "[INFO] Last 20 log lines:"
docker logs --tail 20 gin-demo || true
