package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aether/internal/platform/bootstrap"
	"aether/internal/platform/config"
	"aether/internal/platform/database"
	"aether/internal/platform/health"
	"aether/internal/platform/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDirs(); err != nil {
		slog.Error("prepare directories", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, database.Config{Host: cfg.DatabaseHost, Port: cfg.DatabasePort, Name: cfg.DatabaseName, User: cfg.DatabaseUser, Password: cfg.DatabasePassword, SSLMode: cfg.DatabaseSSLMode, PoolMax: cfg.DatabasePoolMax, ConnectTimeout: cfg.DatabaseConnectTimeout})
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	status := &health.Status{}
	metrics := observability.NewMetrics()
	healthServer := health.NewServer(cfg.WorkerHealthAddr, status)
	healthServer.SetMetrics(metrics)
	go func() {
		if err := healthServer.Run(ctx); err != nil {
			slog.Error("worker health server stopped", "err", err)
			stop()
		}
	}()
	if err := bootstrap.RunWorker(ctx, cfg, masterKey(resolveSecret(cfg.KeysDir)), pool, status, metrics); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}

func resolveSecret(keysDir string) string {
	if value := os.Getenv("AETHER_API_SECRET"); value != "" {
		return value
	}
	path := filepath.Join(keysDir, "master.key")
	if raw, err := os.ReadFile(path); err == nil && len(raw) >= 32 {
		return string(raw)
	}
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err == nil {
		_ = os.MkdirAll(keysDir, 0o700)
		_ = os.WriteFile(path, raw, 0o600)
		return string(raw)
	}
	return "dev-secret-please-override"
}

func masterKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
