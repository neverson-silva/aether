package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aether/internal/platform/bootstrap"
	"aether/internal/platform/config"
	"aether/internal/platform/database"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "apply database migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDirs(); err != nil {
		slog.Error("prepare directories", "err", err)
		os.Exit(1)
	}

	secret := resolveSecret(cfg.KeysDir)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, database.Config{
		Host: cfg.DatabaseHost, Port: cfg.DatabasePort, Name: cfg.DatabaseName,
		User: cfg.DatabaseUser, Password: cfg.DatabasePassword, SSLMode: cfg.DatabaseSSLMode,
		PoolMax: cfg.DatabasePoolMax, ConnectTimeout: cfg.DatabaseConnectTimeout,
	})
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, "api/db/migrations"); err != nil {
		slog.Error("apply migrations", "err", err)
		os.Exit(1)
	}
	if *migrateOnly {
		slog.Info("database migrations applied")
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	bootstrap.Run(ctx, stop, cfg, secret, masterKey(secret), pool, logger)
}

func resolveSecret(keysDir string) string {
	if v := os.Getenv("AETHER_API_SECRET"); v != "" {
		return v
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
