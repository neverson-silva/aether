package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"

	alertsApp "aether/internal/modules/alerts/application"
	alertsInfra "aether/internal/modules/alerts/infra"
	appsInfra "aether/internal/modules/apps/infra"
	"aether/internal/modules/auth/infra"
	backupsEngine "aether/internal/modules/backups/adapters/engine"
	backupsApp "aether/internal/modules/backups/application"
	backupsInfra "aether/internal/modules/backups/infra"
	databasesApp "aether/internal/modules/databases/application"
	databasesInfra "aether/internal/modules/databases/infra"
	deployInfra "aether/internal/modules/deployments/infra"
	domainsApp "aether/internal/modules/domains/application"
	domainsInfra "aether/internal/modules/domains/infra"
	jobsApp "aether/internal/modules/jobs/application"
	jobsInfra "aether/internal/modules/jobs/infra"
	realtimeApp "aether/internal/modules/realtime/application"
	realtimeInfra "aether/internal/modules/realtime/infra"
	servicesInfra "aether/internal/modules/services/infra"
	settingsApp "aether/internal/modules/settings/application"
	settingsInfra "aether/internal/modules/settings/infra"
	snapshotsApp "aether/internal/modules/snapshots/application"
	snapshotsInfra "aether/internal/modules/snapshots/infra"
	sourcecontrolApp "aether/internal/modules/sourcecontrol/application"
	sourcecontrolDomain "aether/internal/modules/sourcecontrol/domain"
	sourcecontrolInfra "aether/internal/modules/sourcecontrol/infra"
	templatesApp "aether/internal/modules/templates/application"
	templatesInfra "aether/internal/modules/templates/infra"
	variablesApp "aether/internal/modules/variables/application"
	variablesInfra "aether/internal/modules/variables/infra"
	composeengine "aether/internal/platform/compose"
	"aether/internal/platform/config"
	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter"
	"aether/internal/platform/druntime/queue"
	platformscheduler "aether/internal/platform/druntime/scheduler"
	platformgit "aether/internal/platform/git"
	"aether/internal/platform/health"
	"aether/internal/platform/observability"
	"aether/internal/platform/outbox"
	githubscm "aether/internal/platform/scm/github"
	"aether/internal/platform/worker"
)

type composeGitClone struct {
	connections *sourcecontrolApp.Connections
}

func (c composeGitClone) Clone(ctx context.Context, source *sourcecontrolDomain.ServiceSource, destination string) (string, error) {
	connection, err := c.connections.Store.GetConnection(ctx, source.ConnectionID)
	if err != nil {
		return "", err
	}
	provider, err := c.connections.ProviderForInstallation(ctx, connection.InstallationID)
	if err != nil {
		return "", err
	}
	credential, err := provider.CreateCloneCredential(ctx, source.RepositoryID)
	if err != nil {
		return "", err
	}
	url := "https://github.com/" + source.RepositoryFullName + ".git"
	if err := platformgit.CloneWithCredential(ctx, url, source.Branch, destination, credential.Username, credential.Secret); err != nil {
		return "", err
	}
	return destination, nil
}

func RunWorker(ctx context.Context, cfg *config.Config, secretKey []byte, pool *pgxpool.Pool, status *health.Status, metrics *observability.Metrics) error {
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	appsStore := appsInfra.NewStore(pool)
	appsSecrets, err := appsInfra.NewSecretCipher(secretKey)
	if err != nil {
		return err
	}
	appsStore.Cipher = appsSecrets
	variablesStore := variablesInfra.NewStore(pool)
	variablesStore.Cipher = appsSecrets
	cronResolver := &variablesApp.Resolver{Vars: variablesStore, Apps: appsStore, Cipher: appsSecrets}
	deployStore := deployInfra.NewStore(pool)
	deployRuntime, err := worker.NewDockerRuntime(cfg.DockerHost)
	if err != nil {
		return err
	}
	defer deployRuntime.Close()
	imageRuntime, err := worker.NewDockerRuntime(cfg.BuildDockerHost)
	if err != nil {
		return err
	}
	defer imageRuntime.Close()
	backupsEngine.Register(deployRuntime)
	if err := ensureIngress(workerCtx, cfg, deployRuntime); err != nil {
		return fmt.Errorf("bootstrap ingress: %w", err)
	}
	domainsStore := domainsInfra.NewStore(pool)
	databasesStore := databasesInfra.NewStore(pool)
	domainsSvc := &domainsApp.Domains{
		Store: domainsStore, Apps: appsStore, DBs: databasesStore,
		Provisioner: &domainsApp.Provisioner{
			TraefikDir:         filepath.Join(cfg.StateDir, "traefik"),
			FreeDomainBase:     cfg.FreeDomainBase,
			FreeDomainProvider: cfg.FreeDomainProvider,
			Runtime:            deployRuntime,
		},
	}

	runtimeConfig := druntime.Config{Backend: cfg.RuntimeBackend, NATSURL: cfg.NATSURL, NATSName: cfg.NATSName, NATSUser: cfg.NATSUser, NATSPassword: cfg.NATSPassword}
	var sharedNATS *natsgo.Conn
	if cfg.RuntimeBackend == "" || cfg.RuntimeBackend == "nats" {
		options := []natsgo.Option{natsgo.Name(cfg.NATSName + "-worker")}
		if cfg.NATSUser != "" || cfg.NATSPassword != "" {
			options = append(options, natsgo.UserInfo(cfg.NATSUser, cfg.NATSPassword))
		}
		sharedNATS, err = natsgo.Connect(cfg.NATSURL, options...)
		if err != nil {
			return err
		}
		defer sharedNATS.Drain()
	}
	var rtRuntime *druntime.Runtime
	if sharedNATS != nil {
		rtRuntime, err = adapter.NewWithConn(workerCtx, runtimeConfig, sharedNATS)
	} else {
		rtRuntime, err = adapter.New(workerCtx, runtimeConfig)
	}
	if err != nil {
		return err
	}
	defer rtRuntime.Close(context.Background())

	store := infra.NewStore(pool)
	alertsStore := alertsInfra.NewStore(pool)
	notifications := &alertsApp.Notifications{Store: alertsStore}
	var eventLog *realtimeInfra.NATSEventLog
	if sharedNATS != nil {
		eventLog, err = realtimeInfra.NewNATSEventLogWithConn(sharedNATS)
	} else {
		eventLog, err = realtimeInfra.NewNATSEventLogWithAuth(cfg.NATSURL, cfg.NATSName, cfg.NATSUser, cfg.NATSPassword)
	}
	if err != nil {
		return err
	}
	defer eventLog.Close()
	realtimeSvc := &realtimeApp.Realtime{
		Presence: rtRuntime.Presence, PubSub: rtRuntime.PubSub,
		Apps: appsStore, Deployments: deployStore, Ports: deployRuntime, Runtime: deployRuntime,
		Log: eventLog, Notifications: notifications,
	}

	deployWorker := &worker.Worker{
		Store: deployStore, Apps: appsStore, Runtime: deployRuntime,
		LogsDir: cfg.LogsDir, BuildsDir: cfg.BuildsDir, UploadsDir: cfg.UploadsDir,
		IngressNetwork: cfg.IngressNetwork, CnbBuilder: cfg.CnbBuilder,
		DockerHost: cfg.DockerHost, BuildDockerHost: cfg.BuildDockerHost,
		Images: imageRuntime, Builder: imageRuntime, Registry: imageRuntime,
		Logger: slog.Default(), Queue: rtRuntime.Queue,
		Metrics:  metrics,
		Notifier: realtimeSvc, LogNotifier: realtimeSvc,
	}
	deployWatcher := &worker.Watcher{Store: deployStore, Runtime: deployRuntime, ServiceStore: servicesInfra.NewStore(pool), Notifier: realtimeSvc, Logger: slog.Default()}
	provisionWorker := &domainsApp.ProvisionWorker{Store: domainsStore, Provisioner: domainsSvc.Provisioner}

	dbCipher, err := databasesInfra.NewPasswordCipher(secretKey)
	if err != nil {
		return err
	}
	databasesSvc := &databasesApp.Databases{Store: databasesStore, Apps: appsStore, Passwords: dbCipher, Runtime: deployRuntime, Network: cfg.IngressNetwork, LogsDir: cfg.LogsDir, Deployments: deployStore}
	composeSvc := &templatesApp.Compose{Store: templatesInfra.NewStore(pool), Apps: appsStore, Deployments: deployStore, DataDir: cfg.DataDir, Runtime: imageRuntime, ProjectVars: variablesStore, ComposeRuntime: composeengine.NewDocker(cfg.BuildDockerHost)}
	sourceStore := sourcecontrolInfra.NewStore(pool)
	composeSvc.Source = sourceStore
	var githubProvider *githubscm.Provider
	if cfg.GitHubAppID != 0 || cfg.GitHubPrivateKey != "" || cfg.GitHubWebhookSecret != "" {
		var providerErr error
		githubProvider, providerErr = githubscm.New(cfg.GitHubAppID, cfg.GitHubPrivateKey, cfg.GitHubWebhookSecret, cfg.GitHubAPIURL)
		if providerErr != nil {
			return providerErr
		}
	}
	connections := &sourcecontrolApp.Connections{Store: sourceStore, Provider: githubProvider, Cipher: appsSecrets, APIURL: cfg.GitHubAPIURL}
	composeSvc.Clone = composeGitClone{connections: connections}
	composeSvc.ServiceIdentity = func(ctx context.Context, specID uuid.UUID) (uuid.UUID, error) {
		var serviceID uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT service_id FROM compose_apps WHERE id = $1 UNION ALL SELECT service_id FROM apps WHERE id = $1 LIMIT 1`, specID).Scan(&serviceID); err != nil {
			return uuid.Nil, err
		}
		return serviceID, nil
	}
	deployWorker.ServiceDeploy = func(ctx context.Context, kind string, serviceID, specID, orgID uuid.UUID) (string, error) {
		switch kind {
		case "compose":
			return "", composeSvc.Up(ctx, specID, orgID)
		case "database":
			db, err := databasesSvc.DeployForWorker(ctx, specID, orgID)
			if err != nil {
				return "", err
			}
			return db.ContainerID, nil
		default:
			return "", fmt.Errorf("unsupported service deployment kind %q", kind)
		}
	}
	deployWorker.ComposeDeploy = composeSvc
	settingsStore := settingsInfra.NewStore(pool)
	settingsSvc := &settingsApp.Settings{Store: settingsStore, Passwords: dbCipher, PublicURL: cfg.PublicURL, OIDC: settingsInfra.NewOIDCDiscoverer(cfg.PublicURL), GoogleRedirectURI: cfg.GoogleOAuthRedirectURI}
	dbBackupsStore := backupsInfra.NewDatabaseStore(pool)
	dbBackupsSvc := &backupsApp.DatabaseBackups{
		Store: dbBackupsStore, Databases: databasesSvc, Passwords: dbCipher,
		Destinations: settingsDestProvider{s: settingsSvc}, Exec: deployRuntime,
		Queue: rtRuntime.Queue, Scheduler: rtRuntime.Scheduler, Locks: rtRuntime.Locks, Audit: auditRecorder{store: store},
		Notifier: realtimeSvc, Timeout: 45 * time.Minute,
		Outbox:     outbox.NewStore(pool),
		Cache:      rtRuntime.Cache,
		UploadRoot: filepath.Join(cfg.StateDir, "restores"), MaxUploadBytes: cfg.RestoreMaxUploadBytes,
	}
	cronWorker := &jobsApp.CronWorker{Store: jobsInfra.NewStore(pool), Apps: appsStore, Runtime: jobsApp.NewRuntime(deployRuntime), Resolver: cronResolver, Queue: rtRuntime.Queue, Scheduler: rtRuntime.Scheduler, Locks: rtRuntime.Locks, Metrics: metrics, Concurrency: 2}
	snapshotOutputDir := os.Getenv("AETHER_SNAPSHOT_HOST_DIR")
	if snapshotOutputDir == "" {
		snapshotOutputDir = filepath.Join(cfg.StateDir, "snapshots")
	}
	snapshotWorker := &snapshotsApp.SnapshotWorker{Store: snapshotsInfra.NewStore(pool), Executor: snapshotsInfra.DockerExecutor{OutputDir: snapshotOutputDir, Runtime: deployRuntime}, OutputDir: snapshotOutputDir, Queue: rtRuntime.Queue, Scheduler: rtRuntime.Scheduler, Locks: rtRuntime.Locks, Metrics: metrics, Concurrency: 2}
	if err := checkWorkerConsumers(workerCtx, rtRuntime.Queue); err != nil {
		return fmt.Errorf("initialize job consumers: %w", err)
	}

	if err := deployWorker.RecoverInterrupted(workerCtx, time.Now().Add(-30*time.Minute)); err != nil {
		return fmt.Errorf("recover interrupted deployments: %w", err)
	}
	go (&outbox.Dispatcher{Store: outbox.NewStore(pool), Bus: rtRuntime.Events, Jobs: rtRuntime.Queue}).Run(workerCtx)
	go deployWatcher.Run(workerCtx, 10*time.Second)
	go provisionWorker.Run(workerCtx)
	backupWorker := &backupsApp.BackupWorker{Service: dbBackupsSvc, Metrics: metrics, Concurrency: 2}
	if err := backupWorker.RecoverInterrupted(workerCtx, time.Now().Add(-90*time.Minute)); err != nil {
		return fmt.Errorf("recover interrupted backups: %w", err)
	}
	go backupWorker.RunUploadCleanupLoop(workerCtx)
	go deployWorker.Run(workerCtx, 3*time.Second)
	go backupWorker.Run(workerCtx)
	go platformscheduler.Run(workerCtx, rtRuntime.Locks,
		platformscheduler.ReconcileTask{Name: "backups", Run: dbBackupsSvc.Reconcile},
		platformscheduler.ReconcileTask{Name: "cron", Run: cronWorker.Reconcile},
		platformscheduler.ReconcileTask{Name: "snapshots", Run: snapshotWorker.Reconcile},
	)
	go cronWorker.Run(workerCtx)
	go snapshotWorker.Run(workerCtx)
	go watchWorkerHealth(workerCtx, status, rtRuntime.Queue)
	if status != nil {
		status.SetReady(true)
	}
	<-workerCtx.Done()
	return nil
}

func watchWorkerHealth(ctx context.Context, status *health.Status, jobs queue.Queue) {
	if status == nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthy := true
			for _, item := range []struct{ stream, group string }{{"deployments", "workers"}, {"backups", "backup-workers"}, {"snapshots", "snapshot-workers"}, {"cron", "cron-workers"}} {
				if _, err := jobs.Pending(ctx, item.stream, item.group); err != nil {
					healthy = false
					break
				}
			}
			status.SetReady(healthy)
		}
	}
}

func checkWorkerConsumers(ctx context.Context, jobs queue.Queue) error {
	for _, item := range []struct{ stream, group, id string }{{"deployments", "workers", "readiness-deploy"}, {"backups", "backup-workers", "readiness-backup"}, {"snapshots", "snapshot-workers", "readiness-snapshot"}, {"cron", "cron-workers", "readiness-cron"}} {
		consumer, err := jobs.NewConsumer(ctx, item.stream, item.group, item.id)
		if err != nil {
			return err
		}
		if err := consumer.Close(); err != nil {
			return err
		}
	}
	return nil
}
