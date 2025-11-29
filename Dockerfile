# --- Builder stage ---
FROM golang:1.22 AS builder
WORKDIR /app

# Copy go.mod first (for better caching if you ever build on host)
COPY go.mod ./

# Copy the rest of the source
COPY . .

# Resolve deps and generate go.sum if missing
RUN go mod tidy

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/gin-server ./cmd/gin-demo/main.go

# --- Runtime stage ---
FROM debian:stable-slim
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gin-server /app/gin-server

EXPOSE 8080
ENV PORT=8080

# This is what container will run (NOT bash)
CMD ["/app/gin-server"]

