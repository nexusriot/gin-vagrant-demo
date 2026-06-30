package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nexusriot/gin-vagrant-demo/internal/config"
	handlerv1 "github.com/nexusriot/gin-vagrant-demo/internal/handler/v1"
	"github.com/nexusriot/gin-vagrant-demo/internal/middleware"
	"github.com/nexusriot/gin-vagrant-demo/internal/repository"
	"github.com/nexusriot/gin-vagrant-demo/internal/repository/memory"
)

// BuildInfo carries version metadata injected at build time via ldflags.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// New creates the production router with the full middleware stack and all routes wired.
func New(cfg *config.Config, repo repository.ItemRepository, info BuildInfo, log *slog.Logger) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(gin.Recovery())
	r.Use(middleware.Secure(cfg))
	r.Use(middleware.RateLimit(cfg.RateLimitPerSec, cfg.RateLimitBurst))
	r.Use(middleware.MaxBodySize(cfg.MaxBodyBytes))

	// Infrastructure — no auth required.
	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := repo.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// /health kept as a backward-compatible alias for /livez (Docker HEALTHCHECK, vm.sh test).
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, info)
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello from gin-vagrant-demo"})
	})

	// API v1
	v1 := r.Group("/v1")
	{
		authH := &handlerv1.AuthHandler{Secret: cfg.JWTSecret, Clients: cfg.APIClients, Log: log}
		v1.POST("/auth/token", authH.Token)

		protected := v1.Group("")
		protected.Use(middleware.Auth(cfg.JWTSecret, log))
		itemH := &handlerv1.ItemHandler{Repo: repo, Log: log}
		protected.GET("/items", itemH.List)
		protected.POST("/items", itemH.Create)
		protected.GET("/items/:id", itemH.Get)
		protected.PUT("/items/:id", itemH.Update)
		protected.DELETE("/items/:id", itemH.Delete)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found", "path": c.Request.URL.Path})
	})

	return r
}

// NewRouter returns a router with in-memory defaults (convenience for tests and local dev).
func NewRouter() *gin.Engine {
	return NewRouterWithInfo(BuildInfo{Version: "dev", Commit: "none", BuildTime: "unknown"})
}

// NewRouterWithInfo returns a router with the given build info and in-memory defaults.
func NewRouterWithInfo(info BuildInfo) *gin.Engine {
	cfg := &config.Config{
		GinMode:         gin.TestMode,
		JWTSecret:       "test-secret",
		AllowOrigins:    []string{"*"},
		RateLimitPerSec: 10000,
		RateLimitBurst:  10000,
		MaxBodyBytes:    1 << 20,
		APIClients:      []config.Client{{ID: "demo", Secret: "demo-secret"}},
	}
	return New(cfg, memory.NewItemRepository(), info, slog.Default())
}
