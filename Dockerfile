# --- Builder stage ---
FROM golang:1.22 AS builder
WORKDIR /app

# Build args for version metadata (override with --build-arg).
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

# Copy module files first so dependency download is cached independently of source.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source.
COPY . .

# Build a static binary with version info injected via ldflags.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT} \
      -X main.buildTime=${BUILD_TIME}" \
    -o /app/gin-server ./cmd/gin-demo

# --- Runtime stage ---
# Distroless: no shell, no package manager, runs as nonroot by default.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /app/gin-server /app/gin-server

EXPOSE 8080
ENV PORT=8080

# The binary probes its own /health endpoint, so no curl/wget is needed in the image.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/gin-server", "-healthcheck"]

ENTRYPOINT ["/app/gin-server"]
