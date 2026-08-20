package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	alertsApp "aether/internal/alerts/application"
	alertshttp "aether/internal/alerts/http"
	alertsInfra "aether/internal/alerts/infra"
	apihttp "aether/internal/api"
	appsApp "aether/internal/apps/application"
	appshttp "aether/internal/apps/http"
	appsInfra "aether/internal/apps/infra"
	authApp "aether/internal/auth/application"
	authhttp "aether/internal/auth/http"
	"aether/internal/auth/infra"
	backupsApp "aether/internal/backups/application"
	backupshttp "aether/internal/backups/http"
	backupsInfra "aether/internal/backups/infra"
	clustersApp "aether/internal/clusters/application"
	clustershttp "aether/internal/clusters/http"
	clustersInfra "aether/internal/clusters/infra"
	"aether/internal/config"
	"aether/internal/database"
	databasesApp "aether/internal/databases/application"
	databaseshttp "aether/internal/databases/http"
	databasesInfra "aether/internal/databases/infra"
	deployApp "aether/internal/deployments/application"
	deployhttp "aether/internal/deployments/http"
	deployInfra "aether/internal/deployments/infra"
	domainsApp "aether/internal/domains/application"
	domainshttp "aether/internal/domains/http"
	domainsInfra "aether/internal/domains/infra"
	"aether/internal/druntime"
	"aether/internal/druntime/adapter"
	gitopsApp "aether/internal/gitops/application"
	gitopshttp "aether/internal/gitops/http"
	gitopsInfra "aether/internal/gitops/infra"
	hostApp "aether/internal/host/application"
	"aether/internal/hostinfo"
	hosthttp "aether/internal/host/http"
	jobsApp "aether/internal/jobs/application"
	jobshttp "aether/internal/jobs/http"
	jobsInfra "aether/internal/jobs/infra"
	mirrorsApp "aether/internal/mirrors/application"
	mirrorshttp "aether/internal/mirrors/http"
	mirrorsInfra "aether/internal/mirrors/infra"
	monitoringApp "aether/internal/monitoring/application"
	monitoringhttp "aether/internal/monitoring/http"
	monitoringInfra "aether/internal/monitoring/infra"
	orgsApp "aether/internal/orgs/application"
	orgshttp "aether/internal/orgs/http"
	orgsInfra "aether/internal/orgs/infra"
	pipelinesApp "aether/internal/pipelines/application"
	pipelineshttp "aether/internal/pipelines/http"
	pipelinesInfra "aether/internal/pipelines/infra"
	realtimeApp "aether/internal/realtime/application"
	realtimeDomain "aether/internal/realtime/domain"
	realtimehttp "aether/internal/realtime/http"
	realtimeInfra "aether/internal/realtime/infra"
	settingsApp "aether/internal/settings/application"
	settingshttp "aether/internal/settings/http"
	settingsInfra "aether/internal/settings/infra"
	snapshotsApp "aether/internal/snapshots/application"
	snapshotshttp "aether/internal/snapshots/http"
	snapshotsInfra "aether/internal/snapshots/infra"
	specsApp "aether/internal/specs/application"
	specshttp "aether/internal/specs/http"
	statsApp "aether/internal/stats/application"
	statshttp "aether/internal/stats/http"
	templatesApp "aether/internal/templates/application"
	templateshttp "aether/internal/templates/http"
	templatesInfra "aether/internal/templates/infra"
	variablesApp "aether/internal/variables/application"
	variableshttp "aether/internal/variables/http"
	variablesInfra "aether/internal/variables/infra"
	volumesApp "aether/internal/volumes/application"
	volumeshttp "aether/internal/volumes/http"
	volumesInfra "aether/internal/volumes/infra"
	webhooksApp "aether/internal/webhooks/application"
	webhookshttp "aether/internal/webhooks/http"
	webhooksInfra "aether/internal/webhooks/infra"
	"aether/internal/worker"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "apply database migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("carregar config", "err", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDirs(); err != nil {
		slog.Error("preparar diretórios", "err", err)
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
		slog.Error("conectar banco", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if *migrateOnly {
		if err := database.Migrate(ctx, pool, "db/migrations"); err != nil {
			slog.Error("aplicar migrations", "err", err)
			os.Exit(1)
		}
		slog.Info("database migrations applied")
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := database.Migrate(ctx, pool, "db/migrations"); err != nil {
		slog.Error("aplicar migrations", "err", err)
		os.Exit(1)
	}

	store := infra.NewStore(pool)
	svc := &authApp.Auth{
		Users: store, Orgs: store, Members: store, Keys: store, AuditLog: store,
		Tokens: infra.NewSigner(secret), Hash: infra.NewHasher(),
		TokenTTL: 7 * 24 * time.Hour,
	}
	handler := authhttp.New(svc)

	appsStore := appsInfra.NewStore(pool)
	appsSecrets, err := appsInfra.NewSecretCipher(masterKey(secret))
	if err != nil {
		slog.Error("cipher de secrets do app", "err", err)
		os.Exit(1)
	}
	appsStore.Cipher = appsSecrets

	deployStore := deployInfra.NewStore(pool)
	deploySvc := &deployApp.Deployments{Store: deployStore, Apps: appsStore}
	deployWorkerRuntime := worker.NewPodmanRuntime()
	appOps := &deployApp.AppOps{Deployments: deploySvc, Runtime: deployWorkerRuntime}

	appsSvc := &appsApp.Apps{Store: appsStore, Secrets: appsSecrets, Containers: appOps}
	appsSvc.LatestDeployments = func(ctx context.Context, appIDs []uuid.UUID) (map[uuid.UUID]string, error) {
		latest, err := deployStore.LatestByApps(ctx, appIDs)
		if err != nil {
			return nil, err
		}
		out := make(map[uuid.UUID]string, len(latest))
		for id, dep := range latest {
			out[id] = string(dep.Status)
		}
		return out, nil
	}
	appsHandler := appshttp.New(appsSvc)
	deployHandler := deployhttp.New(deploySvc, appsStore, appOps, cfg.LogsDir, deployWorkerRuntime)

	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	deployWorker := &worker.Worker{
		Store:          deployStore,
		Apps:           appsStore,
		Runtime:        deployWorkerRuntime,
		LogsDir:        cfg.LogsDir,
		BuildsDir:      cfg.BuildsDir,
		UploadsDir:     cfg.UploadsDir,
		IngressNetwork: cfg.IngressNetwork,
		CnbBuilder:     cfg.CnbBuilder,
		Logger:         slog.Default(),
	}
	domainsStore := domainsInfra.NewStore(pool)
	databasesStore := databasesInfra.NewStore(pool)
	domainsSvc := &domainsApp.Domains{
		Store: domainsStore, Apps: appsStore, DBs: databasesStore,
		Provisioner: &domainsApp.Provisioner{
			TraefikDir:         filepath.Join(cfg.StateDir, "traefik"),
			FreeDomainBase:     cfg.FreeDomainBase,
			FreeDomainProvider: cfg.FreeDomainProvider,
			TraefikBin:         cfg.TraefikBin,
		},
	}
	domainsHandler := domainshttp.New(domainsSvc)
	autoFreeDomain := func(ctx context.Context, id uuid.UUID, serviceType, name string, orgID uuid.UUID) {
		if !domainsSvc.Provisioner.IsPublicBase() {
			return
		}
		_, _ = domainsSvc.GenerateFreeDomain(ctx, id, orgID, serviceType, true)
	}
	appsSvc.OnCreated = autoFreeDomain
	ensureIngress(workerCtx, cfg)
	provisionWorker := &domainsApp.ProvisionWorker{Store: domainsStore, Provisioner: domainsSvc.Provisioner}
	go provisionWorker.Run(workerCtx)

	jobsStore := jobsInfra.NewStore(pool)
	jobsSvc := &jobsApp.Jobs{Store: jobsStore, Apps: appsStore, Runtime: jobsApp.NewRuntime()}
	jobsHandler := jobshttp.New(jobsSvc)

	dbCipher, err := databasesInfra.NewPasswordCipher(masterKey(secret))
	if err != nil {
		slog.Error("cipher de senhas", "err", err)
		os.Exit(1)
	}
	databasesSvc := &databasesApp.Databases{Store: databasesStore, Apps: appsStore, Passwords: dbCipher, Runtime: deployWorkerRuntime, Network: cfg.IngressNetwork, LogsDir: cfg.LogsDir, Deployments: deployStore}
	databasesSvc.OnCreated = autoFreeDomain
	databasesStudio := &databasesApp.Studio{Databases: databasesSvc, Timeout: 30 * time.Second, MaxRows: 1000}
	databasesHandler := databaseshttp.New(databasesSvc, databasesStudio)

	backupsStore := backupsInfra.NewStore(pool)
	backupsSvc := &backupsApp.Backups{Store: backupsStore, Databases: databasesStore}
	backupsHandler := backupshttp.New(backupsSvc)

	templatesStore := templatesInfra.NewStore(pool)
	templatesSvc := &templatesApp.Templates{Store: templatesStore, Apps: appsStore}
	composeSvc := &templatesApp.Compose{Store: templatesStore, Apps: appsStore, Deployments: deployStore, DataDir: cfg.DataDir}
	templatesHandler := templateshttp.New(templatesSvc, composeSvc)

	gitopsStore := gitopsInfra.NewStore(pool)
	gitopsSvc := &gitopsApp.GitOps{Store: gitopsStore}
	gitopsHandler := gitopshttp.New(gitopsSvc)

	alertsStore := alertsInfra.NewStore(pool)
	alertsSvc := &alertsApp.Alerts{Store: alertsStore}
	notificationsSvc := &alertsApp.Notifications{Store: alertsStore}
	channelsSvc := &alertsApp.Channels{Store: alertsStore, Keys: dbCipher}
	alertsHandler := alertshttp.New(alertsSvc, notificationsSvc, channelsSvc)

	snapshotsStore := snapshotsInfra.NewStore(pool)
	snapshotsSvc := &snapshotsApp.Snapshots{Store: snapshotsStore}
	snapshotsHandler := snapshotshttp.New(snapshotsSvc)

	clustersStore := clustersInfra.NewStore(pool)
	clustersSvc := &clustersApp.Clusters{Store: clustersStore}
	clustersHandler := clustershttp.New(clustersSvc)

	pipelinesStore := pipelinesInfra.NewStore(pool)
	pipelinesSvc := &pipelinesApp.Pipelines{Store: pipelinesStore, Apps: appsStore, StageRunner: pipelinesApp.PodmanStageRunner{}}
	pipelinesHandler := pipelineshttp.New(pipelinesSvc)

	settingsStore := settingsInfra.NewStore(pool)
	svc.SSO = settingsStore
	settingsSvc := &settingsApp.Settings{
		Store: settingsStore, Passwords: dbCipher,
		OIDC:              settingsInfra.NewOIDCDiscoverer(cfg.PublicURL),
		GoogleRedirectURI: cfg.GoogleOAuthRedirectURI,
		PublicURL:         cfg.PublicURL,
	}
	settingsHandler := settingshttp.New(settingsSvc).WithSSOLogin(func(ctx context.Context, email, name string) (any, string, error) {
		user, token, err := svc.SSOLogin(ctx, email, name)
		if err != nil {
			return nil, "", err
		}
		return user, token, nil
	})

	webhooksStore := webhooksInfra.NewStore(pool)
	webhooksSvc := &webhooksApp.Webhooks{Store: webhooksStore, Passwords: dbCipher}
	webhookProviders := &webhooksApp.ProviderHooks{
		Apps: appsStore, Passwords: dbCipher, Deployer: webhookDeployer{svc: deploySvc},
	}
	webhooksHandler := webhookshttp.New(webhooksSvc, webhookProviders)

	mirrorsStore := mirrorsInfra.NewStore(pool)
	mirrorsSvc := &mirrorsApp.Mirrors{Store: mirrorsStore}
	mirrorsHandler := mirrorshttp.New(mirrorsSvc)

	volumesStore := volumesInfra.NewStore(pool)
	volumesSvc := &volumesApp.Volumes{Store: volumesStore, Apps: appsStore, Destinations: settingsStore}
	volumesHandler := volumeshttp.New(volumesSvc)

	orgsStore := orgsInfra.NewStore(pool)
	orgsSvc := &orgsApp.Organizations{Store: orgsStore, Apps: appsStore, Databases: databasesStore, Domains: domainsStore}
	orgsHandler := orgshttp.New(orgsSvc)

	variablesStore := variablesInfra.NewStore(pool)
	variablesStore.Cipher = appsSecrets
	variablesSvc := &variablesApp.Variables{Store: variablesStore, Apps: appsStore}
	variablesHandler := variableshttp.New(variablesSvc)
	deploySvc.Resolver = &variablesApp.Resolver{Vars: variablesStore, Apps: appsStore, Cipher: appsSecrets}
	appsHandler.WithResolver(deploySvc.Resolver)
	composeSvc.ProjectVars = variablesStore
	jobsSvc.Resolver = deploySvc.Resolver

	hostSvc := &hostApp.Host{LogsDir: cfg.LogsDir, AgentFile: filepath.Join(cfg.StateDir, "host-stats.json"), PublicIP: hostinfo.PublicIP(), FreeDomainBase: domainsSvc.Provisioner.EffectiveBase()}
	hostHandler := hosthttp.New(hostSvc)

	specsSvc := &specsApp.Specs{Apps: appsStore, Deployments: deployStore, Runtime: deployWorkerRuntime}
	specsAnalyzer := &specsApp.Analyzer{UploadsDir: cfg.UploadsDir}
	specsHandler := specshttp.New(specsSvc, specsAnalyzer, appsStore)

	statsSvc := &statsApp.Stats{Apps: appsStore, Deployments: deployStore, Databases: databasesStore, Runtime: deployWorkerRuntime}
	statsHandler := statshttp.New(statsSvc)

	rtRuntime, err := adapter.New(context.Background(), druntime.Config{
		Backend:       cfg.RuntimeBackend,
		RedisAddr:     cfg.RedisAddr,
		RedisPassword: cfg.RedisPassword,
		RedisUsername: cfg.RedisUsername,
		RedisDB:       cfg.RedisDB,
	})
	if err != nil {
		slog.Error("runtime realtime", "err", err)
		os.Exit(1)
	}
	var eventLog realtimeApp.EventLog
	if cfg.RuntimeBackend == "redis" {
		eventLog, err = realtimeInfra.NewRedisEventLog(druntime.Config{
			RedisAddr:     cfg.RedisAddr,
			RedisPassword: cfg.RedisPassword,
			RedisUsername: cfg.RedisUsername,
			RedisDB:       cfg.RedisDB,
		})
		if err != nil {
			slog.Error("event log realtime", "err", err)
			os.Exit(1)
		}
	}
	realtimeSvc := &realtimeApp.Realtime{
		Presence: rtRuntime.Presence, PubSub: rtRuntime.PubSub,
		Apps: appsStore, Deployments: deployStore, Ports: deployWorkerRuntime,
		Log: eventLog, Notifications: notificationsSvc,
	}
	databasesStudio.Cache = rtRuntime.Cache
	databasesStudio.CatalogTTL = time.Duration(cfg.StudioCacheTTLSeconds) * time.Second
	deployWorker.Notifier = realtimeSvc
	deployWorker.LogNotifier = realtimeSvc
	deploySvc.Queue = rtRuntime.Queue
	deploySvc.Notifier = realtimeSvc
	deployWorker.Queue = rtRuntime.Queue
	go deployWorker.Run(workerCtx, 3*time.Second)
	deployWatcher := &worker.Watcher{Store: deployStore, Runtime: deployWorkerRuntime, Notifier: realtimeSvc, Logger: slog.Default()}
	go deployWatcher.Run(workerCtx, 10*time.Second)
	realtimeHub := realtimeInfra.NewHub(realtimeInfra.HubOptions{
		SubscribeOrg: func(ctx context.Context, orgID uuid.UUID, handler func(realtimeDomain.Event)) (func(), error) {
			sub, err := realtimeSvc.SubscribeEvents(ctx, orgID, handler)
			if err != nil {
				return nil, err
			}
			return func() { _ = sub.Unsubscribe() }, nil
		},
		Replay:    realtimeSvc.ReplayEvents,
		Authorize: realtimeSvc.Authorize,
	})
	realtimeHandler := realtimehttp.New(realtimeSvc, realtimeHub)

	monitoringSvc := monitoringApp.NewMonitoring(deployWorkerRuntime, hostSvc, slog.Default(), monitoringInfra.NewStore(pool))
	go monitoringSvc.Run(workerCtx, 2*time.Second)
	monitoringHandler := monitoringhttp.New(monitoringSvc)

	router := apihttp.New(apihttp.Options{
		Logger:          logger,
		CORSOrigins:     []string{"*"},
		RequestTimeout:  60 * time.Second,
		AuthRateLimiter: apihttp.NewRateLimiter(2, 5),
	}, handler, appsHandler, deployHandler, domainsHandler, jobsHandler, databasesHandler, backupsHandler, templatesHandler, gitopsHandler, alertsHandler, snapshotsHandler, clustersHandler, pipelinesHandler, settingsHandler, webhooksHandler, mirrorsHandler, volumesHandler, orgsHandler, variablesHandler, hostHandler, specsHandler, statsHandler, realtimeHandler, monitoringHandler)
	router.SetReadiness(func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	router.ServeFrontend("web/dist")

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("api nova no ar", "addr", cfg.APIAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	stopWorker()
	_ = rtRuntime.Close(shutdownCtx)
	slog.Info("api encerrada")
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

type webhookDeployer struct {
	svc *deployApp.Deployments
}

func (d webhookDeployer) Deploy(ctx context.Context, appID, orgID uuid.UUID, trigger, commit string) (any, error) {
	return d.svc.Deploy(ctx, appID, orgID, deployApp.DeployOpts{Trigger: trigger, CommitSHA: commit})
}
