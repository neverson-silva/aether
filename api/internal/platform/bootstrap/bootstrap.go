package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"

	alertsApp "aether/internal/modules/alerts/application"
	alertshttp "aether/internal/modules/alerts/http"
	alertsInfra "aether/internal/modules/alerts/infra"
	appsApp "aether/internal/modules/apps/application"
	appshttp "aether/internal/modules/apps/http"
	appsInfra "aether/internal/modules/apps/infra"
	authApp "aether/internal/modules/auth/application"
	authdomain "aether/internal/modules/auth/domain"
	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/auth/infra"
	container "aether/internal/modules/backups/adapters/container"
	_ "aether/internal/modules/backups/adapters/engine"
	backupsApp "aether/internal/modules/backups/application"
	backupshttp "aether/internal/modules/backups/http"
	backupsInfra "aether/internal/modules/backups/infra"
	clustersApp "aether/internal/modules/clusters/application"
	clustershttp "aether/internal/modules/clusters/http"
	clustersInfra "aether/internal/modules/clusters/infra"
	databasesApp "aether/internal/modules/databases/application"
	databaseshttp "aether/internal/modules/databases/http"
	databasesInfra "aether/internal/modules/databases/infra"
	deployApp "aether/internal/modules/deployments/application"
	deployhttp "aether/internal/modules/deployments/http"
	deployInfra "aether/internal/modules/deployments/infra"
	domainsApp "aether/internal/modules/domains/application"
	domainshttp "aether/internal/modules/domains/http"
	domainsInfra "aether/internal/modules/domains/infra"
	gitopsApp "aether/internal/modules/gitops/application"
	gitopshttp "aether/internal/modules/gitops/http"
	gitopsInfra "aether/internal/modules/gitops/infra"
	hostApp "aether/internal/modules/host/application"
	hosthttp "aether/internal/modules/host/http"
	jobsApp "aether/internal/modules/jobs/application"
	jobshttp "aether/internal/modules/jobs/http"
	jobsInfra "aether/internal/modules/jobs/infra"
	mirrorsApp "aether/internal/modules/mirrors/application"
	mirrorshttp "aether/internal/modules/mirrors/http"
	mirrorsInfra "aether/internal/modules/mirrors/infra"
	monitoringApp "aether/internal/modules/monitoring/application"
	monitoringhttp "aether/internal/modules/monitoring/http"
	monitoringInfra "aether/internal/modules/monitoring/infra"
	orgsApp "aether/internal/modules/orgs/application"
	orgshttp "aether/internal/modules/orgs/http"
	orgsInfra "aether/internal/modules/orgs/infra"
	pipelinesApp "aether/internal/modules/pipelines/application"
	pipelineshttp "aether/internal/modules/pipelines/http"
	pipelinesInfra "aether/internal/modules/pipelines/infra"
	realtimeApp "aether/internal/modules/realtime/application"
	realtimeDomain "aether/internal/modules/realtime/domain"
	realtimehttp "aether/internal/modules/realtime/http"
	realtimeInfra "aether/internal/modules/realtime/infra"
	settingsApp "aether/internal/modules/settings/application"
	settingshttp "aether/internal/modules/settings/http"
	settingsInfra "aether/internal/modules/settings/infra"
	snapshotsApp "aether/internal/modules/snapshots/application"
	snapshotshttp "aether/internal/modules/snapshots/http"
	snapshotsInfra "aether/internal/modules/snapshots/infra"
	sourcecontrolApp "aether/internal/modules/sourcecontrol/application"
	sourcecontrolhttp "aether/internal/modules/sourcecontrol/http"
	sourcecontrolInfra "aether/internal/modules/sourcecontrol/infra"
	specsApp "aether/internal/modules/specs/application"
	specshttp "aether/internal/modules/specs/http"
	statsApp "aether/internal/modules/stats/application"
	statshttp "aether/internal/modules/stats/http"
	templatesApp "aether/internal/modules/templates/application"
	templateshttp "aether/internal/modules/templates/http"
	templatesInfra "aether/internal/modules/templates/infra"
	variablesApp "aether/internal/modules/variables/application"
	variableshttp "aether/internal/modules/variables/http"
	variablesInfra "aether/internal/modules/variables/infra"
	volumesApp "aether/internal/modules/volumes/application"
	volumeshttp "aether/internal/modules/volumes/http"
	volumesInfra "aether/internal/modules/volumes/infra"
	webhooksApp "aether/internal/modules/webhooks/application"
	webhookshttp "aether/internal/modules/webhooks/http"
	webhooksInfra "aether/internal/modules/webhooks/infra"
	apihttp "aether/internal/platform/api"
	"aether/internal/platform/config"
	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter"
	"aether/internal/platform/hostinfo"
	"aether/internal/platform/outbox"
	githubscm "aether/internal/platform/scm/github"
	"aether/internal/platform/storage"
	"aether/internal/platform/worker"
)

func Run(ctx context.Context, stop context.CancelFunc, cfg *config.Config, secret string, secretKey []byte, pool *pgxpool.Pool, logger *slog.Logger) {
	store := infra.NewStore(pool)
	svc := &authApp.Auth{
		Users: store, Orgs: store, Members: store, Keys: store, AuditLog: store,
		Tokens: infra.NewSigner(secret), Hash: infra.NewHasher(),
		TokenTTL: 10 * time.Minute,
	}
	handler := authhttp.New(svc, cfg.CookieSecure)

	appsStore := appsInfra.NewStore(pool)
	appsSecrets, err := appsInfra.NewSecretCipher(secretKey)
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

	domainsStore := domainsInfra.NewStore(pool)
	databasesStore := databasesInfra.NewStore(pool)
	appsSvc.Databases = databasesStore
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
	ensureIngress(ctx, cfg)

	jobsStore := jobsInfra.NewStore(pool)
	jobsSvc := &jobsApp.Jobs{Store: jobsStore, Apps: appsStore, Runtime: jobsApp.NewRuntime()}
	jobsHandler := jobshttp.New(jobsSvc)

	dbCipher, err := databasesInfra.NewPasswordCipher(secretKey)
	if err != nil {
		slog.Error("cipher de senhas", "err", err)
		os.Exit(1)
	}
	databasesSvc := &databasesApp.Databases{Store: databasesStore, Apps: appsStore, Passwords: dbCipher, Runtime: deployWorkerRuntime, Network: cfg.IngressNetwork, LogsDir: cfg.LogsDir, Deployments: deployStore}
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
		Apps: appsStore, Passwords: dbCipher, Deployer: webhookDeployer{svc: deploySvc}, Logger: logger,
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

	runtimeConfig := druntime.Config{
		Backend:  cfg.RuntimeBackend,
		NATSURL:  cfg.NATSURL,
		NATSName: cfg.NATSName,
		NATSUser: cfg.NATSUser, NATSPassword: cfg.NATSPassword,
	}
	var sharedNATS *natsgo.Conn
	if cfg.RuntimeBackend == "" || cfg.RuntimeBackend == "nats" {
		options := []natsgo.Option{natsgo.Name(cfg.NATSName + "-api")}
		if cfg.NATSUser != "" || cfg.NATSPassword != "" {
			options = append(options, natsgo.UserInfo(cfg.NATSUser, cfg.NATSPassword))
		}
		sharedNATS, err = natsgo.Connect(cfg.NATSURL, options...)
		if err != nil {
			slog.Error("connect shared NATS", "err", err)
			os.Exit(1)
		}
		defer sharedNATS.Drain()
	}
	var rtRuntime *druntime.Runtime
	if sharedNATS != nil {
		rtRuntime, err = adapter.NewWithConn(context.Background(), runtimeConfig, sharedNATS)
	} else {
		rtRuntime, err = adapter.New(context.Background(), runtimeConfig)
	}
	if err != nil {
		slog.Error("runtime realtime", "err", err)
		os.Exit(1)
	}
	var eventLog realtimeApp.EventLog
	if sharedNATS != nil {
		eventLog, err = realtimeInfra.NewNATSEventLogWithConn(sharedNATS)
		if err != nil {
			slog.Error("event log realtime", "err", err)
			os.Exit(1)
		}
	}
	realtimeSvc := &realtimeApp.Realtime{
		Presence: rtRuntime.Presence, PubSub: rtRuntime.PubSub,
		Queue: rtRuntime.Queue,
		Apps:  appsStore, Deployments: deployStore, Ports: deployWorkerRuntime,
		Log: eventLog, Notifications: notificationsSvc,
	}
	databasesStudio.Cache = rtRuntime.Cache
	databasesStudio.CatalogTTL = time.Duration(cfg.StudioCacheTTLSeconds) * time.Second
	deploySvc.Queue = rtRuntime.Queue
	deploySvc.Outbox = outbox.NewStore(pool)
	deploySvc.Notifier = realtimeSvc

	dbBackupsStore := backupsInfra.NewDatabaseStore(pool)
	dbBackupsSvc := &backupsApp.DatabaseBackups{
		Store:        dbBackupsStore,
		Databases:    databasesSvc,
		Passwords:    dbCipher,
		Destinations: settingsDestProvider{s: settingsSvc},
		Exec:         container.NewPodman(),
		Queue:        rtRuntime.Queue,
		Scheduler:    rtRuntime.Scheduler,
		Locks:        rtRuntime.Locks,
		Audit:        auditRecorder{store: svc.AuditLog},
		Notifier:     realtimeSvc,
		Outbox:       outbox.NewStore(pool),
		UploadRoot:   filepath.Join(cfg.StateDir, "restores"), MaxUploadBytes: cfg.RestoreMaxUploadBytes,
		Timeout: 45 * time.Minute,
	}
	dbBackupsHandler := backupshttp.NewDatabaseBackupHandler(dbBackupsSvc)
	backupsSvc.Async = dbBackupsSvc
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
		Presence:  rtRuntime.Presence,
	})
	realtimeHandler := realtimehttp.New(realtimeSvc, realtimeHub)

	monitoringHistory := monitoringApp.NewMonitoring(nil, nil, slog.Default(), monitoringInfra.NewStore(pool))
	var monitoringReader *monitoringInfra.Reader
	if sharedNATS != nil {
		monitoringReader, err = monitoringInfra.NewReaderWithConn(ctx, sharedNATS, monitoringHistory)
	} else {
		monitoringReader, err = monitoringInfra.NewReaderWithAuth(ctx, cfg.NATSURL, cfg.NATSName, cfg.NATSUser, cfg.NATSPassword, monitoringHistory)
	}
	if err != nil {
		slog.Error("monitoring realtime", "err", err)
		os.Exit(1)
	}
	monitoringHandler := monitoringhttp.New(monitoringReader)

	sourceStore := sourcecontrolInfra.NewStore(pool)
	var githubProvider *githubscm.Provider
	if cfg.GitHubAppID != 0 || cfg.GitHubPrivateKey != "" || cfg.GitHubWebhookSecret != "" {
		githubProvider, err = githubscm.New(cfg.GitHubAppID, cfg.GitHubPrivateKey, cfg.GitHubWebhookSecret, cfg.GitHubAPIURL)
		if err != nil {
			slog.Error("configure github app", "err", err)
			os.Exit(1)
		}
	}
	sourceConnections := &sourcecontrolApp.Connections{
		Store: sourceStore, Provider: githubProvider, Cipher: appsSecrets,
		PublicURL: cfg.PublicURL, APIURL: cfg.GitHubAPIURL,
	}
	sourceService := &sourcecontrolApp.Service{
		Sources: sourceStore, Deliveries: sourceStore, Deploy: webhookDeployer{svc: deploySvc},
		Templates: sourceConnections, Apps: appsSvc,
	}
	sourceHandler := sourcecontrolhttp.New(sourceService, sourceConnections, githubProvider, cfg.GitHubAppSlug)

	router := apihttp.New(apihttp.Options{
		Logger:          logger,
		CORSOrigins:     []string{"*"},
		RequestTimeout:  60 * time.Second,
		AuthRateLimiter: apihttp.NewRateLimiter(2, 5),
	}, handler, appsHandler, deployHandler, domainsHandler, jobsHandler, databasesHandler, backupsHandler, templatesHandler, gitopsHandler, alertsHandler, snapshotsHandler, clustersHandler, pipelinesHandler, settingsHandler, webhooksHandler, mirrorsHandler, volumesHandler, orgsHandler, variablesHandler, hostHandler, specsHandler, statsHandler, realtimeHandler, monitoringHandler)
	router.WithDatabaseBackups(dbBackupsHandler)
	router.WithSourceControl(sourceHandler)
	router.SetReadiness(func(ctx context.Context) error {
		if err := pool.Ping(ctx); err != nil {
			return err
		}
		if runtimeReady := rtRuntime.Queue; runtimeReady != nil {
			_, err := runtimeReady.Len(ctx, "deployments")
			return err
		}
		return nil
	})
	router.ServeFrontend("frontend/web/dist")

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
	_ = rtRuntime.Close(shutdownCtx)
	slog.Info("api encerrada")
}

type webhookDeployer struct {
	svc *deployApp.Deployments
}

func (d webhookDeployer) Deploy(ctx context.Context, appID, orgID uuid.UUID, trigger, commit string) (any, error) {
	return d.svc.Deploy(ctx, appID, orgID, deployApp.DeployOpts{Trigger: trigger, CommitSHA: commit})
}

type settingsDestProvider struct {
	s *settingsApp.Settings
}

func (d settingsDestProvider) GetProvider(ctx context.Context, destID, orgID uuid.UUID) (storage.Provider, error) {
	dest, err := d.s.GetS3(ctx, destID, orgID)
	if err != nil {
		return nil, err
	}
	if dest.IsOAuth() {
		return d.s.GoogleProvider(dest)
	}
	return d.s.S3Provider(dest)
}

type auditRecorder struct {
	store authdomain.AuditStore
}

func (a auditRecorder) Record(ctx context.Context, orgID uuid.UUID, action, resourceType, resourceID, details string) {
	_ = a.store.Record(ctx, authdomain.AuditEvent{
		OrgID: orgID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: details,
	})
}
