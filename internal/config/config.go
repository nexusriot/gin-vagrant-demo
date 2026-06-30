package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client is a service account that can exchange credentials for a JWT.
type Client struct {
	ID     string
	Secret string
}

type Config struct {
	Port    int
	GinMode string

	DatabaseURL string
	RedisURL    string

	JWTSecret  string
	APIClients []Client

	AllowOrigins []string

	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration

	RateLimitPerSec float64
	RateLimitBurst  int
	MaxBodyBytes    int64
}

func Load() (*Config, error) {
	secret := envStr("JWT_SECRET", "change-me-in-production")
	if secret == "change-me-in-production" {
		fmt.Fprintln(os.Stderr, "WARNING: JWT_SECRET is not set — using insecure default; set JWT_SECRET in production")
	}

	return &Config{
		Port:    envInt("PORT", 8080),
		GinMode: envStr("GIN_MODE", "release"),

		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),

		JWTSecret:  secret,
		APIClients: parseClients(os.Getenv("API_CLIENTS")),

		AllowOrigins: strings.Split(envStr("ALLOW_ORIGINS", "*"), ","),

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,

		RateLimitPerSec: 10,
		RateLimitBurst:  30,
		MaxBodyBytes:    1 << 20, // 1 MB
	}, nil
}

// parseClients parses "id1:secret1,id2:secret2" → []Client.
// Falls back to a single demo client when the env var is unset.
func parseClients(raw string) []Client {
	if raw == "" {
		return []Client{{ID: "demo", Secret: "demo-secret"}}
	}
	var clients []Client
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 && parts[0] != "" {
			clients = append(clients, Client{ID: parts[0], Secret: parts[1]})
		}
	}
	return clients
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
