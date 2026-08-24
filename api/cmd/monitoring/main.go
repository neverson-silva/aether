package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	hostApp "aether/internal/modules/host/application"
	monitoringApp "aether/internal/modules/monitoring/application"
	monitoringInfra "aether/internal/modules/monitoring/infra"
	"aether/internal/platform/config"
	"aether/internal/platform/database"
	"aether/internal/platform/health"
	"aether/internal/platform/hostinfo"
	"aether/internal/platform/observability"
	"aether/internal/platform/worker"
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
	healthServer := health.NewServer(cfg.MonitoringHealthAddr, status)
	healthServer.SetMetrics(metrics)
	go func() {
		if err := healthServer.Run(ctx); err != nil {
			slog.Error("monitoring health server stopped", "err", err)
			stop()
		}
	}()
	host := &hostApp.Host{LogsDir: cfg.LogsDir, AgentFile: filepath.Join(cfg.StateDir, "host-stats.json"), PublicIP: hostinfo.PublicIP(), FreeDomainBase: cfg.FreeDomainBase}
	publisher, err := monitoringInfra.NewPublisherWithAuth(cfg.NATSURL, cfg.NATSName, cfg.NATSUser, cfg.NATSPassword)
	if err != nil {
		slog.Error("connect NATS", "err", err)
		os.Exit(1)
	}
	defer publisher.Close()
	publisher.Metrics = metrics
	collector := monitoringApp.NewMonitoring(worker.NewPodmanRuntime(), host, slog.Default(), monitoringInfra.NewStore(pool))
	collector.Metrics = metrics
	collector.Publish = publisher.Publish
	collector.Collect(ctx)
	status.SetReady(true)
	collector.Run(ctx, 2*time.Second)
}
