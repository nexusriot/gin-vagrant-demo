package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nexusriot/gin-vagrant-demo/internal/config"
	"github.com/nexusriot/gin-vagrant-demo/internal/repository"
	"github.com/nexusriot/gin-vagrant-demo/internal/repository/memory"
	"github.com/nexusriot/gin-vagrant-demo/internal/repository/postgres"
	"github.com/nexusriot/gin-vagrant-demo/internal/server"
)

// Injected at build time via -ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local /health endpoint and exit (used by Docker HEALTHCHECK)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	if *healthcheck {
		os.Exit(runHealthcheck(cfg.Port))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var repo repository.ItemRepository
	if cfg.DatabaseURL != "" {
		pool, err := postgres.Connect(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("postgres connect failed", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		repo = postgres.NewItemRepository(pool)
		logger.Info("repository: postgres")
	} else {
		repo = memory.NewItemRepository()
		logger.Warn("repository: in-memory (data is not persisted; set DATABASE_URL to use postgres)")
	}

	r := server.New(cfg, repo, server.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
	}, logger)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	go func() {
		logger.Info("starting", "version", version, "commit", commit, "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", "error", err)
	}
	logger.Info("stopped")
}

func runHealthcheck(port int) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}
