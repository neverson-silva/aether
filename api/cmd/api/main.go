package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	if !cfg.DevMode {
		if cfg.DatabaseURL == "" && cfg.DatabasePassword == "" {
			slog.Error("load config", "err", "DATABASE_PASSWORD or DATABASE_URL must be set outside development mode")
			os.Exit(1)
		}
		if (cfg.DatabaseSSLMode == "disable" || cfg.DatabaseSSLMode == "prefer") && cfg.DatabaseHost != "aether-postgres" {
			slog.Error("load config", "err", "DATABASE_SSL_MODE must require TLS outside development mode")
			os.Exit(1)
		}
		if cfg.RuntimeBackend == "nats" && (cfg.NATSUser == "" || cfg.NATSPassword == "") {
			slog.Error("load config", "err", "NATS credentials must be set outside development mode")
			os.Exit(1)
		}
		if strings.HasPrefix(cfg.BuildDockerHost, "unix://") {
			slog.Error("load config", "err", "AETHER_BUILD_DOCKER_HOST must use the Docker control endpoint outside development mode")
			os.Exit(1)
		}
	}
	if err := cfg.EnsureDirs(); err != nil {
		slog.Error("prepare directories", "err", err)
		os.Exit(1)
	}

	secret, err := resolveSecret(cfg.KeysDir)
	if err != nil {
		slog.Error("load application secret", "err", err)
		os.Exit(1)
	}
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

func resolveSecret(keysDir string) (string, error) {
	if v := os.Getenv("AETHER_API_SECRET"); v != "" {
		if len(v) < 32 {
			return "", fmt.Errorf("AETHER_API_SECRET must contain at least 32 bytes")
		}
		return v, nil
	}
	path := filepath.Join(keysDir, "master.key")
	if raw, err := os.ReadFile(path); err == nil && len(raw) == 32 {
		if info, err := os.Stat(path); err != nil {
			return "", err
		} else if info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("master key permissions must be 0600 or stricter")
		}
		return string(raw), nil
	}
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err == nil {
		if err := os.MkdirAll(keysDir, 0o700); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return "", err
		}
		return string(raw), nil
	} else {
		return "", err
	}
}

func masterKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
