# --- Builder stage ---
FROM golang:1.22 AS builder
WORKDIR /app

# Copy go.mod (and optionally go.sum) first for caching
COPY go.mod ./
# COPY go.sum ./  # uncomment if you have go.sum

# Copy the rest of the source
COPY . .

# Resolve dependencies
RUN go mod tidy

# Build static binary (package, not single file path)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/gin-server ./cmd/gin-demo

# --- Runtime stage ---
FROM debian:stable-slim
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gin-server /app/gin-server

EXPOSE 8080
ENV PORT=8080

CMD ["/app/gin-server"]
