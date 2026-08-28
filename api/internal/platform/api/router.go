package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	alertshttp "aether/internal/modules/alerts/http"
	appshttp "aether/internal/modules/apps/http"
	authhttp "aether/internal/modules/auth/http"
	backupshttp "aether/internal/modules/backups/http"
	clustershttp "aether/internal/modules/clusters/http"
	databaseshttp "aether/internal/modules/databases/http"
	deployhttp "aether/internal/modules/deployments/http"
	domainshttp "aether/internal/modules/domains/http"
	gitopshttp "aether/internal/modules/gitops/http"
	hosthttp "aether/internal/modules/host/http"
	jobshttp "aether/internal/modules/jobs/http"
	mirrorshttp "aether/internal/modules/mirrors/http"
	monitoringhttp "aether/internal/modules/monitoring/http"
	orgshttp "aether/internal/modules/orgs/http"
	pipelineshttp "aether/internal/modules/pipelines/http"
	realtimehttp "aether/internal/modules/realtime/http"
	settingshttp "aether/internal/modules/settings/http"
	snapshotshttp "aether/internal/modules/snapshots/http"
	sourcecontrolhttp "aether/internal/modules/sourcecontrol/http"
	specshttp "aether/internal/modules/specs/http"
	statshttp "aether/internal/modules/stats/http"
	templateshttp "aether/internal/modules/templates/http"
	variableshttp "aether/internal/modules/variables/http"
	volumeshttp "aether/internal/modules/volumes/http"
	webhookshttp "aether/internal/modules/webhooks/http"
)

type Options struct {
	Logger          *slog.Logger
	CORSOrigins     []string
	RequestTimeout  time.Duration
	AuthRateLimiter *RateLimiter
}

type Router struct {
	engine      *gin.Engine
	auth        *authhttp.Handler
	apps        *appshttp.Handler
	deployments *deployhttp.Handler
	domains     *domainshttp.Handler
	jobs        *jobshttp.Handler
	databases   *databaseshttp.Handler
	backups     *backupshttp.Handler
	dbBackups   *backupshttp.DBBackupHandler
	templates   *templateshttp.Handler
	gitops      *gitopshttp.Handler
	alerts      *alertshttp.Handler
	snapshots   *snapshotshttp.Handler
	clusters    *clustershttp.Handler
	pipelines   *pipelineshttp.Handler
	settings    *settingshttp.Handler
	webhooks    *webhookshttp.Handler
	mirrors     *mirrorshttp.Handler
	volumes     *volumeshttp.Handler
	orgs        *orgshttp.Handler
	variables   *variableshttp.Handler
	host        *hosthttp.Handler
	specs       *specshttp.Handler
	stats       *statshttp.Handler
	realtime    *realtimehttp.Handler
	monitoring  *monitoringhttp.Handler
	ready       func(context.Context) error
	authLimiter *RateLimiter
}

func (r *Router) WithDatabaseBackups(h *backupshttp.DBBackupHandler) *Router {
	r.dbBackups = h
	authed := r.engine.Group("/api/v1")
	authed.Use(r.auth.Middleware())
	r.registerDatabaseBackupRoutes(authed)
	return r
}

func (r *Router) WithSourceControl(h *sourcecontrolhttp.Handler) *Router {
	if h != nil {
		api := r.engine.Group("/api/v1")
		api.POST("/webhooks/github", h.GitHubPush)
		authed := api.Group("")
		authed.Use(r.auth.Middleware())
		authed.GET("/source-control/github/install-url", h.GitHubInstallURL)
		authed.POST("/source-control/github/manifest/start", h.StartGitHubManifest)
		authed.GET("/source-control/github/connections", h.ListConnections)
		authed.POST("/source-control/github/connections", h.ConnectGitHub)
		authed.DELETE("/source-control/github/connections/:connectionID", h.DisconnectGitHub)
		authed.GET("/source-control/github/repositories", h.ListRepositories)
		authed.GET("/source-control/github/repositories/:repositoryID/branches", h.ListBranches)
		authed.GET("/source-control/github/repositories/:repositoryID/file", h.GetRepositoryFile)
		authed.GET("/apps/:appID/source", h.GetServiceSource)
		authed.PUT("/apps/:appID/source", h.SaveServiceSource)
		authed.POST("/apps/:appID/source/import-template", h.ImportServiceTemplate)
		authed.DELETE("/apps/:appID/source", h.DeleteServiceSource)
		api.GET("/source-control/github/manifest/callback", h.CompleteGitHubManifest)
		api.GET("/source-control/github/install-callback", h.CompleteGitHubInstallation)
	}
	return r
}

func New(opts Options, auth *authhttp.Handler, apps *appshttp.Handler, deployments *deployhttp.Handler, domains *domainshttp.Handler, jobs *jobshttp.Handler, databases *databaseshttp.Handler, backups *backupshttp.Handler, templates *templateshttp.Handler, gitops *gitopshttp.Handler, alerts *alertshttp.Handler, snapshots *snapshotshttp.Handler, clusters *clustershttp.Handler, pipelines *pipelineshttp.Handler, settings *settingshttp.Handler, webhooks *webhookshttp.Handler, mirrors *mirrorshttp.Handler, volumes *volumeshttp.Handler, orgs *orgshttp.Handler, variables *variableshttp.Handler, host *hosthttp.Handler, specs *specshttp.Handler, stats *statshttp.Handler, realtime *realtimehttp.Handler, monitoring *monitoringhttp.Handler) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(RequestID())
	if opts.Logger != nil {
		engine.Use(RequestLogger(opts.Logger))
	}
	if len(opts.CORSOrigins) > 0 {
		engine.Use(CORS(opts.CORSOrigins))
	}
	if opts.RequestTimeout > 0 {
		engine.Use(Timeout(opts.RequestTimeout))
	}

	r := &Router{engine: engine, auth: auth, apps: apps, deployments: deployments, domains: domains, jobs: jobs, databases: databases, backups: backups, dbBackups: nil, templates: templates, gitops: gitops, alerts: alerts, snapshots: snapshots, clusters: clusters, pipelines: pipelines, settings: settings, webhooks: webhooks, mirrors: mirrors, volumes: volumes, orgs: orgs, variables: variables, host: host, specs: specs, stats: stats, realtime: realtime, monitoring: monitoring, authLimiter: opts.AuthRateLimiter}
	r.routes()
	return r
}

func (r *Router) SetReadiness(check func(context.Context) error) {
	r.ready = check
}

func (r *Router) handleReady(c *gin.Context) {
	if r.ready != nil {
		if err := r.ready(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (r *Router) routes() {
	api := r.engine.Group("/api/v1")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/ready", r.handleReady)

	authRoutes := api.Group("")
	if r.authLimiter != nil {
		authRoutes.Use(RateLimit(r.authLimiter))
	}
	authRoutes.POST("/auth/register", r.auth.Register)
	authRoutes.POST("/auth/login", r.auth.Login)
	authRoutes.POST("/auth/logout", r.auth.Logout)
	authRoutes.GET("/auth/status", r.auth.AuthStatus)
	authRoutes.GET("/sso/public", r.settings.PublicSSO)
	authRoutes.GET("/sso/public/:providerID/auth-url", r.settings.PublicSSOAuthURL)
	authRoutes.POST("/sso/:providerID/callback", r.settings.SSOCallback)
	authRoutes.POST("/webhooks/github/:appID", r.webhooks.GitHub)
	authRoutes.POST("/webhooks/gitlab/:appID", r.webhooks.GitLab)
	authRoutes.POST("/webhooks/bitbucket/:appID", r.webhooks.Bitbucket)

	authed := api.Group("")
	authed.Use(r.auth.Middleware())
	{
		authed.GET("/auth/me", r.auth.Me)
		authed.GET("/auth/members", r.auth.ListMembers)
		authed.POST("/auth/members", r.auth.AddMember)
		authed.GET("/auth/keys", r.auth.ListKeys)
		authed.POST("/auth/keys", r.auth.CreateKey)
		authed.DELETE("/auth/keys/:keyID", r.auth.DeleteKey)
		authed.GET("/api-keys", r.auth.ListKeysV1)
		authed.POST("/api-keys", r.auth.CreateKeyV1)
		authed.DELETE("/api-keys/:keyID", r.auth.DeleteKeyV1)
		authed.GET("/auth/audit", r.auth.Audit)
		authed.POST("/auth/totp/enroll", r.auth.TOTPEnroll)
		authed.POST("/auth/totp/verify", r.auth.TOTPVerify)
		authed.DELETE("/auth/totp", r.auth.TOTPDisable)
		authed.GET("/me", r.auth.Me)
		authed.GET("/members", r.auth.MembersV1)
		authed.POST("/members", r.auth.AddMemberV1)
		authed.PATCH("/members/:userID", r.auth.UpdateMemberV1)
		authed.GET("/metrics", r.auth.Metrics)

		authed.GET("/projects", r.apps.ListProjects)
		authed.POST("/projects", r.apps.CreateProject)
		authed.GET("/projects/:projectID", r.apps.GetProject)
		authed.PATCH("/projects/:projectID", r.apps.UpdateProject)
		authed.DELETE("/projects/:projectID", r.apps.DeleteProject)

		authed.GET("/projects/:projectID/environments", r.apps.ListEnvironments)
		authed.POST("/projects/:projectID/environments", r.apps.CreateEnvironment)
		authed.PATCH("/projects/:projectID/environments/:envID", r.apps.UpdateEnvironment)
		authed.DELETE("/projects/:projectID/environments/:envID", r.apps.DeleteEnvironment)

		authed.GET("/projects/:projectID/apps", r.apps.ListApps)
		authed.POST("/projects/:projectID/apps", r.apps.CreateApp)
		authed.GET("/apps", r.apps.ListApps)
		authed.GET("/apps/:appID", r.apps.GetApp)
		authed.PATCH("/apps/:appID", r.apps.UpdateApp)
		authed.DELETE("/apps/:appID", r.apps.DeleteApp)

		authed.PUT("/apps/:appID/env", r.apps.SetEnv)
		authed.GET("/apps/:appID/env", r.apps.ListEnv)
		authed.GET("/apps/:appID/variables/effective", r.apps.EffectiveVariables)
		authed.DELETE("/apps/:appID/env/:name", r.apps.DeleteEnv)
		authed.PUT("/apps/:appID/webhook", r.apps.SetWebhook)

		authed.POST("/apps/:appID/deploy", r.deployments.Deploy)
		authed.POST("/apps/:appID/rollback", r.deployments.Rollback)
		authed.GET("/apps/:appID/deployments", r.deployments.List)
		authed.GET("/apps/:appID/deployments/:depID", r.deployments.Get)
		authed.POST("/apps/:appID/deployments/:depID/cancel", r.deployments.Cancel)
		authed.GET("/apps/:appID/deployments/:depID/log", r.deployments.DeploymentLog)
		authed.GET("/apps/:appID/deployments/:depID/logs", r.deployments.LogHistory)
		authed.GET("/apps/:appID/logs", r.deployments.Logs)
		authed.GET("/apps/:appID/logs/history", r.deployments.AppLogHistory)
		authed.GET("/deployments/:depID", r.deployments.Get)

		authed.POST("/apps/:appID/start", r.deployments.AppStart)
		authed.POST("/apps/:appID/stop", r.deployments.AppStop)
		authed.POST("/apps/:appID/restart", r.deployments.AppRestart)
		authed.POST("/apps/:appID/rebuild", r.deployments.AppRebuild)
		authed.GET("/apps/:appID/timeline", r.deployments.AppTimeline)
		authed.GET("/apps/states", r.deployments.AppStates)
		authed.GET("/apps/states/stream", r.deployments.AppStatesStream)

		authed.POST("/apps/:appID/domains", r.domains.AddDomain)
		authed.GET("/apps/:appID/domains", r.domains.ListDomains)
		authed.POST("/apps/:appID/domains/generate", r.domains.GenerateFreeDomain)
		authed.PATCH("/apps/:appID/domains/:domainID", r.domains.UpdateDomain)
		authed.DELETE("/apps/:appID/domains/:host", r.domains.RemoveDomain)
		authed.POST("/apps/:appID/domains/:domainID/verify", r.domains.VerifyDomain)
		authed.POST("/apps/:appID/domains/:domainID/reprovision", r.domains.ReprovisionDomain)
		authed.GET("/apps/:appID/domains/:domainID/status", r.domains.GetDomainStatus)
		authed.POST("/databases/:dbID/domains", r.domains.AddDomain)
		authed.GET("/databases/:dbID/domains", r.domains.ListDomains)
		authed.POST("/databases/:dbID/domains/generate", r.domains.GenerateFreeDomain)
		authed.PATCH("/databases/:dbID/domains/:domainID", r.domains.UpdateDomain)
		authed.DELETE("/databases/:dbID/domains/:host", r.domains.RemoveDomain)
		authed.POST("/databases/:dbID/domains/:domainID/verify", r.domains.VerifyDomain)
		authed.POST("/databases/:dbID/domains/:domainID/reprovision", r.domains.ReprovisionDomain)
		authed.GET("/databases/:dbID/domains/:domainID/status", r.domains.GetDomainStatus)
		authed.POST("/apps/:appID/previews", r.domains.CreatePreview)
		authed.GET("/apps/:appID/previews", r.domains.ListPreviews)
		authed.DELETE("/previews/:previewID", r.domains.DeletePreview)

		authed.POST("/apps/:appID/cron-jobs", r.jobs.CreateCronJob)
		authed.GET("/apps/:appID/cron-jobs", r.jobs.ListCronJobs)
		authed.GET("/cron-jobs", r.jobs.ListAllCronJobs)
		authed.PATCH("/cron-jobs/:jobID", r.jobs.UpdateCronJob)
		authed.DELETE("/cron-jobs/:jobID", r.jobs.DeleteCronJob)

		authed.POST("/apps/:appID/workers", r.jobs.CreateWorker)
		authed.GET("/apps/:appID/workers", r.jobs.ListWorkers)
		authed.POST("/workers/:workerID/start", r.jobs.StartWorker)
		authed.POST("/workers/:workerID/stop", r.jobs.StopWorker)
		authed.DELETE("/workers/:workerID", r.jobs.DeleteWorker)

		authed.GET("/apps/:appID/policy", r.jobs.GetPolicy)
		authed.PUT("/apps/:appID/policy", r.jobs.SavePolicy)
		authed.GET("/apps/:appID/policy/events", r.jobs.PolicyEvents)

		authed.POST("/databases", r.databases.Create)
		authed.GET("/databases", r.databases.List)
		authed.GET("/databases/:dbID", r.databases.Get)
		authed.DELETE("/databases/:dbID", r.databases.Delete)
		authed.POST("/databases/:dbID/deploy", r.databases.Deploy)
		authed.POST("/databases/:dbID/rebuild", r.databases.Rebuild)
		authed.GET("/databases/:dbID/deployments", r.databases.ListDeployments)
		authed.POST("/databases/:dbID/start", r.databases.Start)
		authed.POST("/databases/:dbID/stop", r.databases.Stop)
		authed.POST("/databases/:dbID/backup", r.backups.CreateDatabaseBackup)
		authed.POST("/databases/:dbID/restore", r.backups.RestoreDatabase)
		r.registerDatabaseBackupRoutes(authed)
		authed.GET("/ws/db-terminal/:dbID", r.databases.DbTerminal)

		authed.GET("/databases/:dbID/studio/meta", r.databases.StudioMeta)
		authed.GET("/databases/:dbID/studio/catalog", r.databases.StudioCatalog)
		authed.GET("/databases/:dbID/studio/schemas", r.databases.StudioSchemas)
		authed.GET("/databases/:dbID/studio/schemas/:schema/objects", r.databases.StudioObjects)

		authed.GET("/databases/:dbID/studio/schemas/:schema/objects/list", r.databases.StudioObjectsList)
		authed.GET("/databases/:dbID/studio/tables/:schema/:table", r.databases.StudioTable)
		authed.GET("/databases/:dbID/studio/tables/:schema/:table/data", r.databases.StudioTableData)
		authed.POST("/databases/:dbID/studio/query", r.databases.StudioQuery)
		authed.POST("/databases/:dbID/studio/exec", r.databases.StudioExec)
		authed.POST("/databases/:dbID/studio/tables", r.databases.StudioCreateTable)
		authed.POST("/databases/:dbID/studio/tables/rename", r.databases.StudioRenameTable)
		authed.POST("/databases/:dbID/studio/tables/drop", r.databases.StudioDropTable)
		authed.POST("/databases/:dbID/studio/tables/alter", r.databases.StudioAlterTable)
		authed.POST("/databases/:dbID/studio/refresh", r.databases.StudioRefresh)

		authed.GET("/backups", r.backups.List)
		authed.POST("/backups", r.backups.CreateStateBackup)
		authed.POST("/backups/:backupID/restore", r.backups.RestoreState)

		authed.GET("/templates", r.templates.List)
		authed.POST("/templates/:templateID/install", r.templates.Install)
		authed.GET("/compose", r.templates.ListCompose)
		authed.DELETE("/compose/:composeID", r.templates.DeleteCompose)
		authed.POST("/compose", r.templates.Create)
		authed.GET("/compose/:composeID", r.templates.Get)
		authed.POST("/compose/:composeID/up", r.templates.Up)
		authed.POST("/compose/:composeID/down", r.templates.Down)
		authed.POST("/compose/validate", r.templates.Validate)
		authed.GET("/apps/:appID/compose", r.templates.AppCompose)
		authed.GET("/apps/:appID/deployments/:depID/compose", r.templates.DeploymentCompose)
		authed.POST("/apps/:appID/compose/import", r.templates.ImportCompose)

		authed.GET("/gitops", r.gitops.List)
		authed.POST("/gitops", r.gitops.Create)
		authed.POST("/gitops/:gitopsID/sync", r.gitops.Sync)
		authed.DELETE("/gitops/:gitopsID", r.gitops.Delete)

		authed.GET("/alerts/rules", r.alerts.ListRules)
		authed.POST("/alerts/rules", r.alerts.CreateRule)
		authed.PATCH("/alerts/rules/:ruleID", r.alerts.PatchRule)
		authed.DELETE("/alerts/rules/:ruleID", r.alerts.DeleteRule)
		authed.GET("/alerts/events", r.alerts.ListEvents)
		authed.POST("/alerts/events/:eventID/resolve", r.alerts.ResolveEvent)

		authed.GET("/notifications", r.alerts.ListNotifications)
		authed.GET("/notifications/unread-count", r.alerts.UnreadCount)
		authed.POST("/notifications/:notifID/read", r.alerts.MarkRead)
		authed.POST("/notifications/read-all", r.alerts.MarkAllRead)

		authed.POST("/notification-channels", r.alerts.CreateChannel)
		authed.GET("/notification-channels", r.alerts.ListChannels)
		authed.DELETE("/notification-channels/:channelID", r.alerts.DeleteChannel)

		authed.GET("/snapshots", r.snapshots.List)
		authed.POST("/snapshots", r.snapshots.Create)
		authed.POST("/snapshots/:snapshotID/restore", r.snapshots.Restore)
		authed.DELETE("/snapshots/:snapshotID", r.snapshots.Delete)
		authed.GET("/snapshots/schedules", r.snapshots.ListSchedules)
		authed.POST("/snapshots/schedules", r.snapshots.CreateSchedule)
		authed.DELETE("/snapshots/schedules/:scheduleID", r.snapshots.DeleteSchedule)

		authed.GET("/clusters", r.clusters.ListClusters)
		authed.POST("/clusters", r.clusters.CreateCluster)
		authed.DELETE("/clusters/:clusterID", r.clusters.DeleteCluster)
		authed.POST("/clusters/:clusterID/servers", r.clusters.AddServer)
		authed.DELETE("/clusters/:clusterID/servers/:serverID", r.clusters.RemoveServer)

		authed.GET("/servers", r.clusters.ListServers)
		authed.POST("/servers/token", r.clusters.ServerToken)
		authed.DELETE("/servers/:serverID", r.clusters.DeleteServer)

		authed.GET("/registry", r.clusters.GetRegistry)
		authed.POST("/registry", r.clusters.SetRegistry)
		authed.GET("/registry/images", r.clusters.RegistryImages)
		authed.DELETE("/registry/images/:repo/:tag", r.clusters.RegistryImageDelete)

		authed.GET("/pipelines", r.pipelines.List)
		authed.POST("/pipelines", r.pipelines.Create)
		authed.DELETE("/pipelines/:pipelineID", r.pipelines.Delete)
		authed.POST("/pipelines/:pipelineID/run", r.pipelines.Run)
		authed.GET("/pipelines/:pipelineID/runs", r.pipelines.ListRuns)

		authed.GET("/branding", r.settings.GetBranding)
		authed.PUT("/branding", r.settings.SaveBranding)
		authed.POST("/s3-destinations", r.settings.CreateS3)
		authed.GET("/s3-destinations", r.settings.ListS3)
		authed.PATCH("/s3-destinations/:destID", r.settings.UpdateS3)
		authed.POST("/s3-destinations/:destID/test", r.settings.TestS3)
		authed.DELETE("/s3-destinations/:destID", r.settings.DeleteS3)
		authed.POST("/s3-destinations/:destID/google/connect", r.settings.GoogleConnect)
		authed.GET("/s3-destinations/google/callback", r.settings.GoogleCallback)
		authed.POST("/s3-destinations/:destID/google/disconnect", r.settings.GoogleDisconnect)

		authed.GET("/sso", r.settings.ListSSO)
		authed.POST("/sso", r.settings.CreateSSO)
		authed.POST("/sso/:providerID/auth-url", r.settings.AuthURL)
		authed.DELETE("/sso/:providerID", r.settings.DeleteSSO)

		authed.GET("/webhooks", r.webhooks.List)
		authed.POST("/webhooks", r.webhooks.Create)
		authed.DELETE("/webhooks/:webhookID", r.webhooks.Delete)

		authed.GET("/mirrors", r.mirrors.List)
		authed.POST("/mirrors", r.mirrors.Create)
		authed.POST("/mirrors/:mirrorID/run", r.mirrors.Run)
		authed.DELETE("/mirrors/:mirrorID", r.mirrors.Delete)

		authed.POST("/apps/:appID/volumes/:name/backup", r.volumes.BackupVolume)
		authed.GET("/apps/:appID/volumes", r.volumes.List)

		authed.GET("/certificates", r.domains.Certificates)

		authed.GET("/org/export", r.orgs.Export)
		authed.POST("/org/import", r.orgs.Import)
		authed.GET("/organizations", r.orgs.List)
		authed.POST("/organizations", r.orgs.Create)
		authed.GET("/organizations/:orgID", r.orgs.Get)
		authed.PATCH("/organizations/:orgID", r.orgs.Update)
		authed.DELETE("/organizations/:orgID", r.orgs.Delete)
		authed.GET("/organizations/:orgID/members", r.orgs.Members)
		authed.PATCH("/organizations/:orgID/members/:userID", r.orgs.UpdateMember)
		authed.DELETE("/organizations/:orgID/members/:userID", r.orgs.RemoveMember)
		authed.POST("/organizations/:orgID/members/:userID/projects/:projectID", r.orgs.AssignProject)
		authed.DELETE("/organizations/:orgID/members/:userID/projects/:projectID", r.orgs.RemoveAssignment)
		authed.GET("/organizations/:orgID/audit", r.orgs.Audit)

		authed.GET("/projects/:projectID/variables", r.variables.ListVariables)
		authed.POST("/projects/:projectID/variables", r.variables.SetVariable)
		authed.PUT("/projects/:projectID/variables", r.variables.ReplaceVariables)
		authed.DELETE("/projects/:projectID/variables/:key", r.variables.DeleteVariable)
		authed.GET("/projects/:projectID/variables/audit", r.variables.Audit)
		authed.GET("/projects/:projectID/variables/export", r.variables.Export)
		authed.POST("/projects/:projectID/variables/import", r.variables.Import)
		authed.GET("/projects/:projectID/environments/:envID/variables", r.variables.ListEnvironmentVariables)
		authed.PUT("/projects/:projectID/environments/:envID/variables", r.variables.ReplaceEnvironmentVariables)
		authed.POST("/projects/:projectID/environments/:envID/variables", r.variables.SetEnvironmentVariable)
		authed.PATCH("/projects/:projectID/environments/:envID/variables/:key", r.variables.SetEnvironmentVariable)
		authed.DELETE("/projects/:projectID/environments/:envID/variables/:key", r.variables.DeleteEnvironmentVariable)
		authed.GET("/projects/:projectID/environments/:envID/variables/export", r.variables.ExportEnvironmentVariables)
		authed.POST("/projects/:projectID/environments/:envID/variables/import", r.variables.ImportEnvironmentVariables)
		authed.GET("/projects/:projectID/environments/:envID/variables/audit", r.variables.EnvironmentAudit)
		authed.POST("/projects/:projectID/environments/:envID/default", r.variables.SetDefaultEnvironment)

		authed.GET("/host/stats", r.host.Stats)
		authed.GET("/host/stats/stream", r.host.StatsStream)
		authed.GET("/host/events", r.host.Events)
		authed.GET("/host/logs", r.host.Logs)
		authed.GET("/host/info", r.host.Info)

		authed.GET("/monitoring", r.monitoring.Overview)
		authed.GET("/monitoring/resources", r.monitoring.Resources)
		authed.GET("/monitoring/resources/:id/history", r.monitoring.ResourceHistory)
		authed.GET("/monitoring/history", r.monitoring.History)
		authed.GET("/monitoring/collector", r.monitoring.Collector)
		authed.GET("/monitoring/stream", r.monitoring.Stream)

		authed.POST("/detect", r.specs.Detect)
		authed.POST("/analyze", r.specs.Analyze)
		authed.POST("/plan/preview", r.specs.PlanPreview)
		authed.POST("/upload/zip", r.specs.UploadZip)
		authed.POST("/apps/:appID/detect", r.specs.AppDetect)
		authed.POST("/apps/:appID/plan", r.specs.AppPlan)
		authed.GET("/apps/:appID/plan", r.specs.GetAppPlan)

		authed.GET("/apps/:appID/spec", r.specs.AppSpec)
		authed.GET("/apps/:appID/export", r.specs.ExportRuntime)
		authed.GET("/apps/:appID/deployments/compare", r.specs.Compare)
		authed.GET("/system/summary", r.specs.SystemSummary)
		authed.GET("/apps/:appID/stats", r.stats.AppStats)
		authed.GET("/databases/:dbID/stats", r.stats.DatabaseStats)
		authed.GET("/databases/:dbID/logs", r.stats.DatabaseLogs)
		authed.GET("/databases/:dbID/deployments/:depID/logs", r.databases.DeploymentLog)

		authed.POST("/presence/join", r.realtime.Join)
		authed.POST("/presence/heartbeat", r.realtime.Heartbeat)
		authed.POST("/presence/leave", r.realtime.Leave)
		authed.GET("/presence/count", r.realtime.Count)
		authed.GET("/runtime/metrics", r.realtime.RuntimeMetrics)
		authed.GET("/network/quality", r.realtime.NetworkQuality)
		authed.GET("/events", r.realtime.Events)
		authed.GET("/events/stream", r.realtime.EventsStream)
		authed.GET("/ws/terminal/:appID", r.realtime.Terminal)
		authed.GET("/ws/realtime", r.realtime.RealtimeWS)
	}
}

func (r *Router) registerDatabaseBackupRoutes(authed *gin.RouterGroup) {
	if r.dbBackups == nil {
		return
	}
	authed.GET("/databases/:dbID/backup/configurations", r.dbBackups.ListConfigurations)
	authed.GET("/databases/:dbID/backup/configurations/:configID", r.dbBackups.GetConfiguration)
	authed.POST("/databases/:dbID/backup/configurations", r.dbBackups.CreateConfiguration)
	authed.PATCH("/databases/:dbID/backup/configurations/:configID", r.dbBackups.UpdateConfiguration)
	authed.DELETE("/databases/:dbID/backup/configurations/:configID", r.dbBackups.DeleteConfiguration)
	authed.GET("/databases/:dbID/backup/configuration", r.dbBackups.ListConfigurations)
	authed.POST("/databases/:dbID/backup/configuration", r.dbBackups.CreateConfiguration)
	authed.POST("/databases/:dbID/backups", r.dbBackups.CreateBackup)
	authed.GET("/databases/:dbID/backups", r.dbBackups.ListBackups)
	authed.GET("/databases/:dbID/backups/:backupID", r.dbBackups.GetBackup)
	authed.POST("/databases/:dbID/backups/:backupID/cancel", r.dbBackups.CancelBackup)
	authed.POST("/databases/:dbID/backups/:backupID/restore", r.dbBackups.RequestRestore)
	authed.GET("/databases/:dbID/backups/:backupID/preflight", r.dbBackups.PreflightRestore)
	authed.GET("/databases/:dbID/backups/:backupID/restore-jobs", r.dbBackups.ListRestoreJobs)
	authed.POST("/databases/:dbID/restores", r.dbBackups.CreateRestore)
	authed.POST("/databases/:dbID/restores/:restoreID/upload", r.dbBackups.UploadRestoreFile)
	authed.POST("/databases/:dbID/restores/:restoreID/validate", r.dbBackups.ValidateUpload)
	authed.POST("/databases/:dbID/restores/:restoreID/start", r.dbBackups.StartUploadRestore)
	authed.GET("/databases/:dbID/restores/:restoreID", r.dbBackups.GetRestore)
	authed.DELETE("/databases/:dbID/restores/:restoreID", r.dbBackups.DeleteRestore)
}

func (r *Router) Handler() http.Handler {
	return r.engine
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) ServeFrontend(distDir string) {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		return
	}
	r.engine.Static("/assets", filepath.Join(distDir, "assets"))
	r.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(filepath.Join(distDir, "index.html"))
	})
}
