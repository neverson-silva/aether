package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nhooyr.io/websocket"

	"aether/internal/core"
	"aether/internal/domain"
	"aether/internal/druntime/pubsub"
	"aether/internal/git"
)

func (s *Server) registerF2Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/databases", s.auth(s.perm("app.write", s.handleCreateDatabase)))
	mux.HandleFunc("GET /api/v1/databases", s.auth(s.perm("app.read", s.handleListDatabases)))
	mux.HandleFunc("GET /api/v1/databases/{id}", s.auth(s.perm("app.read", s.handleGetDatabase)))
	mux.HandleFunc("GET /api/v1/databases/{id}/stats", s.auth(s.perm("app.read", s.handleDatabaseStats)))
	mux.HandleFunc("GET /api/v1/databases/{id}/logs", s.auth(s.perm("app.read", s.handleDatabaseLogs)))
	mux.HandleFunc("DELETE /api/v1/databases/{id}", s.auth(s.perm("app.write", s.handleDeleteDatabase)))
	mux.HandleFunc("POST /api/v1/databases/{id}/backup", s.auth(s.perm("backup.write", s.handleDatabaseBackup)))
	mux.HandleFunc("POST /api/v1/databases/{id}/restore", s.auth(s.perm("backup.write", s.handleDatabaseRestore)))

	mux.HandleFunc("POST /api/v1/apps/{id}/cron-jobs", s.auth(s.perm("app.write", s.handleCreateCron)))
	mux.HandleFunc("GET /api/v1/apps/{id}/cron-jobs", s.auth(s.perm("app.read", s.handleListCron)))
	mux.HandleFunc("PATCH /api/v1/cron-jobs/{id}", s.auth(s.perm("app.write", s.handleUpdateCron)))
	mux.HandleFunc("DELETE /api/v1/cron-jobs/{id}", s.auth(s.perm("app.write", s.handleDeleteCron)))

	mux.HandleFunc("POST /api/v1/apps/{id}/workers", s.auth(s.perm("app.write", s.handleCreateWorker)))
	mux.HandleFunc("GET /api/v1/apps/{id}/workers", s.auth(s.perm("app.read", s.handleListWorkers)))
	mux.HandleFunc("POST /api/v1/workers/{id}/start", s.auth(s.perm("app.write", s.handleStartWorker)))
	mux.HandleFunc("POST /api/v1/workers/{id}/stop", s.auth(s.perm("app.write", s.handleStopWorker)))
	mux.HandleFunc("DELETE /api/v1/workers/{id}", s.auth(s.perm("app.write", s.handleDeleteWorker)))

	mux.HandleFunc("POST /api/v1/compose", s.auth(s.perm("app.write", s.handleCreateCompose)))
	mux.HandleFunc("GET /api/v1/compose", s.auth(s.perm("app.read", s.handleListCompose)))
	mux.HandleFunc("GET /api/v1/compose/{id}", s.auth(s.perm("app.read", s.handleGetCompose)))
	mux.HandleFunc("POST /api/v1/compose/{id}/up", s.auth(s.perm("app.write", s.handleComposeUp)))
	mux.HandleFunc("POST /api/v1/compose/{id}/down", s.auth(s.perm("app.write", s.handleComposeDown)))
	mux.HandleFunc("DELETE /api/v1/compose/{id}", s.auth(s.perm("app.write", s.handleDeleteCompose)))

	mux.HandleFunc("POST /api/v1/apps/{id}/previews", s.auth(s.perm("app.deploy", s.handleCreatePreview)))

	mux.HandleFunc("GET /api/v1/registry", s.auth(s.perm("app.read", s.handleRegistryGet)))
	mux.HandleFunc("POST /api/v1/registry", s.auth(s.perm("app.write", s.handleRegistryEnable)))
	mux.HandleFunc("GET /api/v1/registry/images", s.auth(s.perm("app.read", s.handleRegistryImages)))
	mux.HandleFunc("DELETE /api/v1/registry/images/{repo}/{tag}", s.auth(s.perm("app.write", s.handleRegistryImageDelete)))

	mux.HandleFunc("GET /api/v1/webhooks", s.auth(s.perm("org.read", s.handleListWebhooks)))
	mux.HandleFunc("POST /api/v1/webhooks", s.auth(s.perm("org.write", s.handleCreateWebhook)))
	mux.HandleFunc("DELETE /api/v1/webhooks/{id}", s.auth(s.perm("org.write", s.handleDeleteWebhook)))

	mux.HandleFunc("GET /metrics", s.auth(s.handleMetrics))
	mux.HandleFunc("GET /api/v1/metrics", s.auth(s.handleMetrics))
	mux.HandleFunc("GET /api/v1/servers", s.auth(s.perm("org.read", s.handleListServers)))
	mux.HandleFunc("POST /api/v1/servers/token", s.auth(s.perm("org.write", s.handleServerToken)))
	mux.HandleFunc("DELETE /api/v1/servers/{id}", s.auth(s.perm("org.write", s.handleDeleteServer)))
	mux.HandleFunc("GET /api/v1/apps/{id}/policy", s.auth(s.perm("app.read", s.handleGetPolicy)))
	mux.HandleFunc("PUT /api/v1/apps/{id}/policy", s.auth(s.perm("app.write", s.handleSavePolicy)))
	mux.HandleFunc("GET /api/v1/apps/{id}/policy/events", s.auth(s.perm("app.read", s.handlePolicyEvents)))
	mux.HandleFunc("GET /api/v1/gitops", s.auth(s.perm("org.read", s.handleListGitOps)))
	mux.HandleFunc("POST /api/v1/gitops", s.auth(s.perm("org.write", s.handleCreateGitOps)))
	mux.HandleFunc("POST /api/v1/gitops/{id}/sync", s.auth(s.perm("org.write", s.handleSyncGitOps)))
	mux.HandleFunc("DELETE /api/v1/gitops/{id}", s.auth(s.perm("org.write", s.handleDeleteGitOps)))
	mux.HandleFunc("GET /api/v1/mirrors", s.auth(s.perm("org.read", s.handleListMirrors)))
	mux.HandleFunc("POST /api/v1/mirrors", s.auth(s.perm("org.write", s.handleCreateMirror)))
	mux.HandleFunc("POST /api/v1/mirrors/{id}/run", s.auth(s.perm("org.write", s.handleRunMirror)))
	mux.HandleFunc("DELETE /api/v1/mirrors/{id}", s.auth(s.perm("org.write", s.handleDeleteMirror)))
	mux.HandleFunc("GET /api/v1/network/quality", s.auth(s.perm("org.read", s.handleNetQ)))
	mux.HandleFunc("GET /api/v1/host/stats", s.auth(s.perm("org.read", s.handleHostStats)))
	mux.HandleFunc("GET /api/v1/host/stats/stream", s.auth(s.perm("org.read", s.handleHostStatsStream)))
	mux.HandleFunc("GET /api/v1/host/events", s.auth(s.perm("org.read", s.handleHostEvents)))
	mux.HandleFunc("GET /api/v1/host/logs", s.auth(s.perm("org.read", s.handleHostLogs)))
	mux.HandleFunc("GET /api/v1/apps/states", s.auth(s.perm("app.read", s.handleAppStates)))
	mux.HandleFunc("GET /api/v1/apps/states/stream", s.auth(s.perm("app.read", s.handleAppStatesStream)))
	mux.HandleFunc("POST /api/v1/presence/join", s.auth(s.handlePresenceJoin))
	mux.HandleFunc("POST /api/v1/presence/heartbeat", s.auth(s.handlePresenceHeartbeat))
	mux.HandleFunc("POST /api/v1/presence/leave", s.auth(s.handlePresenceLeave))
	mux.HandleFunc("GET /api/v1/presence/count", s.auth(s.handlePresenceCount))
	mux.HandleFunc("GET /api/v1/runtime/metrics", s.auth(s.handleRuntimeMetrics))
	mux.HandleFunc("POST /api/v1/apps/{id}/detect", s.auth(s.perm("app.read", s.handleDetectApp)))
	mux.HandleFunc("POST /api/v1/apps/{id}/plan", s.auth(s.perm("app.write", s.handleCreatePlan)))
	mux.HandleFunc("GET /api/v1/apps/{id}/plan", s.auth(s.perm("app.read", s.handleGetPlan)))
	mux.HandleFunc("POST /api/v1/compose/validate", s.auth(s.perm("app.read", s.handleValidateCompose)))
	mux.HandleFunc("POST /api/v1/detect", s.auth(s.perm("app.read", s.handleDetectRepo)))
	mux.HandleFunc("POST /api/v1/analyze", s.auth(s.perm("app.read", s.handleAnalyzeRepo)))
	mux.HandleFunc("POST /api/v1/plan/preview", s.auth(s.perm("app.read", s.handlePlanPreview)))
	mux.HandleFunc("GET /api/v1/alerts/rules", s.auth(s.perm("app.read", s.handleAlertRules)))
	mux.HandleFunc("POST /api/v1/alerts/rules", s.auth(s.perm("app.write", s.handleCreateAlertRule)))
	mux.HandleFunc("PATCH /api/v1/alerts/rules/{id}", s.auth(s.perm("app.write", s.handlePatchAlertRule)))
	mux.HandleFunc("DELETE /api/v1/alerts/rules/{id}", s.auth(s.perm("app.write", s.handleDeleteAlertRule)))
	mux.HandleFunc("GET /api/v1/alerts/events", s.auth(s.perm("app.read", s.handleAlertEvents)))
	mux.HandleFunc("POST /api/v1/alerts/events/{id}/resolve", s.auth(s.perm("app.write", s.handleResolveAlert)))
	mux.HandleFunc("GET /api/v1/snapshots/schedules", s.auth(s.perm("backup.read", s.handleSnapshotSchedules)))
	mux.HandleFunc("POST /api/v1/snapshots/schedules", s.auth(s.perm("backup.write", s.handleCreateSnapshotSchedule)))
	mux.HandleFunc("DELETE /api/v1/snapshots/schedules/{id}", s.auth(s.perm("backup.write", s.handleDeleteSnapshotSchedule)))
	mux.HandleFunc("POST /api/v1/apps/{id}/start", s.auth(s.perm("app.write", s.handleAppStart)))
	mux.HandleFunc("POST /api/v1/apps/{id}/stop", s.auth(s.perm("app.write", s.handleAppStop)))
	mux.HandleFunc("POST /api/v1/apps/{id}/restart", s.auth(s.perm("app.write", s.handleAppRestart)))
	mux.HandleFunc("POST /api/v1/apps/{id}/rebuild", s.auth(s.perm("app.deploy", s.handleAppRebuild)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments", s.auth(s.perm("app.read", s.handleListEnvironments)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments", s.auth(s.perm("app.write", s.handleCreateEnvironment)))
	mux.HandleFunc("PATCH /api/v1/projects/{projectID}/environments/{environmentID}", s.auth(s.perm("app.write", s.handleUpdateEnvironment)))
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/environments/{environmentID}", s.auth(s.perm("app.write", s.handleDeleteEnvironment)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments/{environmentID}/default", s.auth(s.perm("app.write", s.handleSetDefaultEnvironment)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments/{environmentID}/variables", s.auth(s.perm("app.read", s.handleListEnvVars)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments/{environmentID}/variables", s.auth(s.perm("app.write", s.handleSetEnvVar)))
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/environments/{environmentID}/variables", s.auth(s.perm("app.write", s.handleReplaceEnvVars)))
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/environments/{environmentID}/variables/{key}", s.auth(s.perm("app.write", s.handleDeleteEnvVar)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments/{environmentID}/variables/export", s.auth(s.perm("app.read", s.handleExportEnvVars)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments/{environmentID}/variables/import", s.auth(s.perm("app.write", s.handleImportEnvVars)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments/{environmentID}/variables/audit", s.auth(s.perm("app.read", s.handleEnvVarAudit)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/variables", s.auth(s.perm("app.read", s.handleListProjectVars)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/variables", s.auth(s.perm("app.write", s.handleSetProjectVar)))
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/variables", s.auth(s.perm("app.write", s.handleReplaceProjectVars)))
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/variables/{key}", s.auth(s.perm("app.write", s.handleDeleteProjectVar)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/variables/export", s.auth(s.perm("app.read", s.handleExportProjectVars)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/variables/import", s.auth(s.perm("app.write", s.handleImportProjectVars)))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/variables/audit", s.auth(s.perm("app.read", s.handleProjectVarAudit)))
	mux.HandleFunc("PATCH /api/v1/projects/{id}", s.auth(s.perm("app.write", s.handleRenameProject)))
	mux.HandleFunc("DELETE /api/v1/projects/{id}", s.auth(s.perm("app.write", s.handleDeleteProject)))
	mux.HandleFunc("GET /api/v1/events/stream", s.handleEventStream)
	mux.HandleFunc("GET /api/v1/notifications", s.auth(s.perm("org.read", s.handleListNotifications)))
	mux.HandleFunc("GET /api/v1/notifications/unread-count", s.auth(s.perm("org.read", s.handleUnreadCount)))
	mux.HandleFunc("POST /api/v1/notifications/{id}/read", s.auth(s.perm("org.read", s.handleMarkNotificationRead)))
	mux.HandleFunc("POST /api/v1/notifications/read-all", s.auth(s.perm("org.read", s.handleMarkAllRead)))
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/sso/public", s.handlePublicSSO)
	mux.HandleFunc("GET /api/v1/sso/public/{id}/auth-url", s.handlePublicSSOAuthURL)
	mux.HandleFunc("GET /api/v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("GET /api/v1/system/summary", s.auth(s.perm("org.read", s.handleSystemSummary)))
	mux.HandleFunc("GET /api/v1/cron-jobs", s.auth(s.perm("org.read", s.handleAllCronJobs)))
	mux.HandleFunc("GET /api/v1/certificates", s.auth(s.perm("org.read", s.handleCertificates)))
	mux.HandleFunc("GET /api/v1/branding", s.auth(s.perm("org.read", s.handleGetBranding)))
	mux.HandleFunc("PUT /api/v1/branding", s.auth(s.perm("org.write", s.handleSaveBranding)))
	mux.HandleFunc("GET /api/v1/pipelines", s.auth(s.perm("org.read", s.handleListPipelines)))
	mux.HandleFunc("POST /api/v1/pipelines", s.auth(s.perm("org.write", s.handleCreatePipeline)))
	mux.HandleFunc("DELETE /api/v1/pipelines/{id}", s.auth(s.perm("org.write", s.handleDeletePipeline)))
	mux.HandleFunc("POST /api/v1/pipelines/{id}/run", s.auth(s.perm("app.deploy", s.handleRunPipeline)))
	mux.HandleFunc("GET /api/v1/pipelines/{id}/runs", s.auth(s.perm("org.read", s.handleListPipelineRuns)))
	mux.HandleFunc("GET /api/v1/clusters", s.auth(s.perm("org.read", s.handleListClusters)))
	mux.HandleFunc("POST /api/v1/clusters", s.auth(s.perm("org.write", s.handleCreateCluster)))
	mux.HandleFunc("DELETE /api/v1/clusters/{id}", s.auth(s.perm("org.write", s.handleDeleteCluster)))
	mux.HandleFunc("POST /api/v1/clusters/{id}/servers", s.auth(s.perm("org.write", s.handleClusterAddServer)))
	mux.HandleFunc("DELETE /api/v1/clusters/{id}/servers/{serverID}", s.auth(s.perm("org.write", s.handleClusterRemoveServer)))
	mux.HandleFunc("GET /api/v1/sso", s.auth(s.perm("org.read", s.handleListSSO)))
	mux.HandleFunc("POST /api/v1/sso", s.auth(s.perm("org.write", s.handleCreateSSO)))
	mux.HandleFunc("POST /api/v1/sso/{id}/auth-url", s.auth(s.perm("org.read", s.handleSSOAuthURL)))
	mux.HandleFunc("POST /api/v1/sso/{id}/callback", s.handleSSOCallback)
	mux.HandleFunc("DELETE /api/v1/sso/{id}", s.auth(s.perm("org.write", s.handleDeleteSSO)))
	mux.HandleFunc("GET /api/v1/snapshots", s.auth(s.perm("org.read", s.handleListSnapshots)))
	mux.HandleFunc("POST /api/v1/snapshots", s.auth(s.perm("app.write", s.handleCreateSnapshot)))
	mux.HandleFunc("POST /api/v1/snapshots/{id}/restore", s.auth(s.perm("app.write", s.handleRestoreSnapshot)))
	mux.HandleFunc("DELETE /api/v1/snapshots/{id}", s.auth(s.perm("app.write", s.handleDeleteSnapshot)))
	mux.HandleFunc("GET /api/v1/apps/{id}/previews", s.auth(s.perm("app.read", s.handleListPreviews)))
	mux.HandleFunc("DELETE /api/v1/previews/{id}", s.auth(s.perm("app.write", s.handleDeletePreview)))

	mux.HandleFunc("GET /api/v1/templates", s.auth(s.perm("app.read", s.handleListTemplates)))
	mux.HandleFunc("POST /api/v1/templates/{id}/install", s.auth(s.perm("app.write", s.handleInstallTemplate)))

	mux.HandleFunc("POST /api/v1/s3-destinations", s.auth(s.perm("backup.write", s.handleCreateS3)))
	mux.HandleFunc("GET /api/v1/s3-destinations", s.auth(s.perm("backup.read", s.handleListS3)))
	mux.HandleFunc("DELETE /api/v1/s3-destinations/{id}", s.auth(s.perm("backup.write", s.handleDeleteS3)))
	mux.HandleFunc("POST /api/v1/apps/{id}/volumes/{name}/backup", s.auth(s.perm("backup.write", s.handleVolumeBackup)))

	mux.HandleFunc("POST /api/v1/notification-channels", s.auth(s.perm("org.write", s.handleCreateChannel)))
	mux.HandleFunc("GET /api/v1/notification-channels", s.auth(s.perm("org.read", s.handleListChannels)))
	mux.HandleFunc("DELETE /api/v1/notification-channels/{id}", s.auth(s.perm("org.write", s.handleDeleteChannel)))

	mux.HandleFunc("POST /api/v1/auth/totp/enroll", s.auth(s.handleTOTPEnroll))
	mux.HandleFunc("POST /api/v1/auth/totp/verify", s.auth(s.handleTOTPVerify))
	mux.HandleFunc("DELETE /api/v1/auth/totp", s.auth(s.handleTOTPDisable))

	mux.HandleFunc("GET /api/v1/ws/terminal/{appID}", s.auth(s.handleTerminal))

	mux.HandleFunc("GET /api/v1/org/export", s.auth(s.perm("backup.read", s.handleExport)))
	mux.HandleFunc("POST /api/v1/org/import", s.auth(s.perm("org.write", s.handleImport)))

	mux.HandleFunc("POST /api/v1/webhooks/gitlab/{appID}", s.handleGitLabWebhook)
	mux.HandleFunc("POST /api/v1/webhooks/bitbucket/{appID}", s.handleBitbucketWebhook)
}

func (s *Server) handleCreateDatabase(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Engine    string `json:"engine"`
		Version   string `json:"version"`
		MemMB     int64  `json:"mem_mb"`
		StorageMB int64  `json:"storage_mb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.ProjectID == "" || req.Name == "" {
		writeErr(w, 400, "project_id e name obrigatórios")
		return
	}
	db, err := s.core.CreateDatabase(claims.OrgID, req.ProjectID, req.Name, domain.DBEngine(req.Engine), req.Version, req.MemMB, req.StorageMB)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, db)
}

func (s *Server) handleListDatabases(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	dbs, err := s.core.Store.ListDatabases(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, dbs)
}

func (s *Server) handleGetDatabase(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	db, err := s.core.Store.GetDatabase(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "banco não encontrado")
		return
	}
	if db.OrgID != claims.OrgID {
		writeErr(w, 403, "fora do escopo da organização")
		return
	}
	dsn, _ := s.core.DatabaseConnectionString(db)
	writeJSON(w, 200, map[string]any{"database": db, "dsn": dsn})
}

func (s *Server) handleDeleteDatabase(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteDatabase(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleDatabaseBackup(w http.ResponseWriter, r *http.Request) {
	b, err := s.core.BackupDatabase(r.PathValue("id"))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, b)
}

func (s *Server) handleDatabaseRestore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BackupID string `json:"backup_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BackupID == "" {
		writeErr(w, 400, "backup_id obrigatório")
		return
	}
	if err := s.core.RestoreDatabase(r.PathValue("id"), req.BackupID); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "restored"})
}

func (s *Server) handleCreateCron(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	j, err := s.core.CreateCronJob(app.ID, req.Name, req.Schedule, req.Command)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, j)
}

func (s *Server) handleListCron(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	jobs, err := s.core.Store.ListCronJobs(app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, jobs)
}

func (s *Server) handleUpdateCron(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schedule *string `json:"schedule"`
		Command  *string `json:"command"`
		Enabled  *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	j, err := s.core.Store.GetCronJob(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "cron não encontrado")
		return
	}
	if req.Schedule != nil {
		j.Schedule = *req.Schedule
	}
	if req.Command != nil {
		j.Command = *req.Command
	}
	if req.Enabled != nil {
		j.Enabled = *req.Enabled
	}
	updated, err := s.core.UpdateCronJob(j.ID, j.Schedule, j.Command, j.Enabled)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, updated)
}

func (s *Server) handleDeleteCron(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteCronJob(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleCreateWorker(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Name     string `json:"name"`
		Command  string `json:"command"`
		Replicas int    `json:"replicas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	worker, err := s.core.CreateWorker(app.ID, req.Name, req.Command, req.Replicas)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, worker)
}

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	workers, err := s.core.ListWorkers(app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, workers)
}

func (s *Server) handleStartWorker(w http.ResponseWriter, r *http.Request) {
	if err := s.core.StartWorker(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "started"})
}

func (s *Server) handleStopWorker(w http.ResponseWriter, r *http.Request) {
	if err := s.core.StopWorker(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) handleDeleteWorker(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteWorker(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleCreateCompose(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Compose   string `json:"compose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	ca, err := s.core.SaveCompose(claims.OrgID, req.ProjectID, req.Name, req.Compose)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, ca)
}

func (s *Server) handleGetCompose(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	ca, err := s.core.GetCompose(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "stack não encontrado")
		return
	}
	if ca.OrgID != "" && ca.OrgID != claims.OrgID && claims.GlobalRole != domain.GlobalAdmin {
		writeErr(w, 403, "fora do escopo da organização")
		return
	}
	writeJSON(w, 200, ca)
}

func (s *Server) handleListCompose(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	stacks, err := s.core.ListCompose(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, stacks)
}

func (s *Server) handleComposeUp(w http.ResponseWriter, r *http.Request) {
	if err := s.core.ComposeUp(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "running"})
}

func (s *Server) handleComposeDown(w http.ResponseWriter, r *http.Request) {
	if err := s.core.ComposeDown(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) handleDeleteCompose(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteCompose(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleCreatePreview(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	p, err := s.core.CreatePreview(app.ID, req.Branch)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (s *Server) handleListPreviews(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	previews, err := s.core.ListPreviews(app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, previews)
}

func (s *Server) handleDeletePreview(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeletePreview(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	filter := core.TemplateFilter{Category: r.URL.Query().Get("category"), Search: r.URL.Query().Get("q")}
	if v := r.URL.Query().Get("editors_choice"); v == "true" {
		filter.EditorsChoice = true
	}
	if v := r.URL.Query().Get("featured"); v == "true" {
		b := true
		filter.Featured = &b
	}
	if v := r.URL.Query().Get("verified"); v == "true" {
		b := true
		filter.Verified = &b
	}
	if v := r.URL.Query().Get("trending"); v == "true" {
		trending, err := s.core.TrendingTemplates(12)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, trending)
		return
	}
	if v := r.URL.Query().Get("categories"); v == "true" {
		writeJSON(w, 200, s.core.TemplateCategories())
		return
	}
	key := "cache:templates:" + filterKey(filter)
	var cached []domain.Template
	if s.cacheGetJSON(key, &cached) {
		writeJSON(w, 200, cached)
		return
	}
	templates, err := s.core.ListTemplatesFiltered(filter)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.cacheSetJSON(key, templates, cacheTemplatesTTL)
	writeJSON(w, 200, templates)
}

func filterKey(f core.TemplateFilter) string {
	return f.Category + "|" + f.Search + "|" + fmtBool(f.EditorsChoice) + "|" + fmtBool(f.Featured != nil && *f.Featured) + "|" + fmtBool(f.Verified != nil && *f.Verified)
}

func fmtBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (s *Server) handleInstallTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string            `json:"project_id"`
		Name      string            `json:"name"`
		Overrides map[string]string `json:"overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	ca, err := s.core.InstallTemplate(r.PathValue("id"), claimsFrom(r).OrgID, req.ProjectID, req.Name, req.Overrides)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, ca)
}

func (s *Server) handleCreateS3(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name      string `json:"name"`
		Endpoint  string `json:"endpoint"`
		Bucket    string `json:"bucket"`
		Region    string `json:"region"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	d, err := s.core.CreateS3Destination(claims.OrgID, req.Name, req.Endpoint, req.Bucket, req.Region, req.AccessKey, req.SecretKey)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, d)
}

func (s *Server) handleListS3(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	dests, err := s.core.Store.ListS3Destinations(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, dests)
}

func (s *Server) handleDeleteS3(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteS3Destination(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleVolumeBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DestinationID string `json:"destination_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	b, err := s.core.BackupVolumeToDestination(r.PathValue("id"), r.PathValue("name"), req.DestinationID)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, b)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name   string             `json:"name"`
		Type   string             `json:"type"`
		Config core.ChannelConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	ch, err := s.core.CreateNotificationChannel(claims.OrgID, req.Name, req.Type, req.Config)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, ch)
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	channels, err := s.core.Store.ListNotificationChannels(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, channels)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteNotificationChannel(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleTOTPEnroll(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	user, err := s.core.Store.GetUser(claims.Subject)
	if err != nil {
		writeErr(w, 404, "usuário não encontrado")
		return
	}
	t, err := s.core.EnrollTOTP(user.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	uri, _ := s.core.TOTPProvisioningURI(user.ID, user.Email)
	writeJSON(w, 200, map[string]string{"secret": t.Secret, "uri": uri})
}

func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if err := s.core.EnableTOTP(claims.Subject, req.Code); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "enabled"})
}

func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if err := s.core.DisableTOTP(claims.Subject); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "disabled"})
}

// shellCommand resolve o shell solicitado pelo cliente (bash / /bin/sh / ash),
// com fallback seguro para /bin/sh (presente em imagens Alpine/scratch-distroless).
func shellCommand(r *http.Request) string {
	shell := r.URL.Query().Get("shell")
	switch shell {
	case "bash", "/bin/bash":
		return "bash"
	case "ash", "/bin/ash":
		return "ash"
	default:
		return "/bin/sh"
	}
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("appID"))
	if !ok {
		return
	}
	dep, err := s.core.Store.LastReadyDeployment(app.ID, 1<<62)
	if err != nil || dep.ContainerID == "" {
		writeErr(w, 404, "sem container ativo")
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closed")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	in, out, ctl := s.core.TerminalChannels(app.ID)
	s.core.StartTerminalHost(app.ID, dep.ContainerID, shellCommand(r))

	sub, err := s.core.RT.PubSub.Subscribe(ctx, out, func(_ context.Context, m pubsub.Message) {
		if werr := conn.Write(ctx, websocket.MessageBinary, m.Data); werr != nil {
			cancel()
		}
	}, pubsub.WithBuffer(512))
	if err != nil {
		conn.Close(websocket.StatusInternalError, "stream indisponível")
		return
	}
	defer sub.Unsubscribe()

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ == websocket.MessageText {
			var ctrl struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" && ctrl.Cols > 0 && ctrl.Rows > 0 {
				pctx, pcancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = s.core.RT.PubSub.Publish(pctx, ctl, data)
				pcancel()
			}
			continue
		}
		pctx, pcancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = s.core.RT.PubSub.Publish(pctx, in, data)
		pcancel()
	}
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	data, err := s.core.ExportOrg(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=aether.yml")
	w.Write(data)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	data, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := s.core.ImportOrg(claims.OrgID, data); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "imported"})
}

func (s *Server) handleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	app, err := s.core.Store.GetApp(r.PathValue("appID"))
	if err != nil {
		writeErr(w, 404, "aplicação não encontrada")
		return
	}
	if app.SourceType != domain.SourceGit {
		writeErr(w, 400, "aplicação não é de fonte git")
		return
	}
	token := r.Header.Get("X-Gitlab-Token")
	if token != "" && app.WebhookSecret != "" {
		secret, err := s.core.Secrets.DecryptString(app.WebhookSecret)
		if err != nil || token != secret {
			writeErr(w, 401, "token inválido")
			return
		}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if push, err := git.ParseGitLabPushEvent(body); err == nil {
		if push.Branch() == app.GitBranch {
			dep, err := s.core.Deploy(app.ID, core.DeployOpts{Trigger: "gitlab"})
			s.core.TriggerDeployPipelines(r.Context(), app.ID, "deploy")
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			writeJSON(w, 202, dep)
			return
		}
		if push.Branch() != "" && push.Branch() != app.GitBranch {
			s.core.CreatePreview(app.ID, push.Branch())
		}
		writeJSON(w, 200, map[string]string{"status": "ignored"})
		return
	}
	if mr, err := git.ParseGitLabMergeEvent(body); err == nil {
		if mr.ObjectAttributes.Action == "close" || mr.ObjectAttributes.State == "closed" {
			s.closePreviewForBranch(app.ID, mr.ObjectAttributes.SourceBranch)
		} else if mr.ObjectAttributes.SourceBranch != "" {
			s.core.CreatePreview(app.ID, mr.ObjectAttributes.SourceBranch)
		}
		writeJSON(w, 200, map[string]string{"status": "processed"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ignored"})
}

func (s *Server) handleBitbucketWebhook(w http.ResponseWriter, r *http.Request) {
	app, err := s.core.Store.GetApp(r.PathValue("appID"))
	if err != nil {
		writeErr(w, 404, "aplicação não encontrada")
		return
	}
	if app.SourceType != domain.SourceGit {
		writeErr(w, 400, "aplicação não é de fonte git")
		return
	}
	if app.WebhookSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		secret, err := s.core.Secrets.DecryptString(app.WebhookSecret)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErr(w, 400, "corpo inválido")
			return
		}
		if err := git.VerifyGitHubSignature(body, sig, []byte(secret)); err != nil {
			writeErr(w, 401, err.Error())
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if push, err := git.ParseBitbucketPushEvent(body); err == nil {
		if push.Branch() == app.GitBranch {
			dep, err := s.core.Deploy(app.ID, core.DeployOpts{Trigger: "bitbucket"})
			s.core.TriggerDeployPipelines(r.Context(), app.ID, "deploy")
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			writeJSON(w, 202, dep)
			return
		}
		if push.Branch() != "" {
			s.core.CreatePreview(app.ID, push.Branch())
		}
		writeJSON(w, 200, map[string]string{"status": "ignored"})
		return
	}
	if pr, err := git.ParseBitbucketPREvent(body); err == nil {
		branch := pr.PullRequest.Source.Branch.Name
		if pr.PullRequest.State == "CLOSED" || pr.PullRequest.State == "MERGED" {
			s.closePreviewForBranch(app.ID, branch)
		} else if branch != "" {
			s.core.CreatePreview(app.ID, branch)
		}
		writeJSON(w, 200, map[string]string{"status": "processed"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ignored"})
}

func (s *Server) closePreviewForBranch(appID, branch string) {
	previews, err := s.core.ListPreviews(appID)
	if err != nil {
		return
	}
	for _, p := range previews {
		if p.Branch == branch && p.Status == "active" {
			s.core.DeletePreview(p.ID)
		}
	}
}

func (s *Server) dbForOrg(w http.ResponseWriter, r *http.Request) (*domain.Database, bool) {
	claims := claimsFrom(r)
	db, err := s.core.Store.GetDatabase(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "banco não encontrado")
		return nil, false
	}
	if db.OrgID != claims.OrgID {
		writeErr(w, 403, "fora do escopo da organização")
		return nil, false
	}
	return db, true
}

func (s *Server) handleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	db, ok := s.dbForOrg(w, r)
	if !ok {
		return
	}
	if db.ContainerID == "" {
		writeErr(w, 404, "sem container")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	stats, err := s.core.Driver.Stats(ctx, db.ContainerID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	info, _ := s.core.Driver.Inspect(ctx, db.ContainerID)
	writeJSON(w, 200, map[string]any{"state": info.State, "stats": stats})
}

func (s *Server) handleDatabaseLogs(w http.ResponseWriter, r *http.Request) {
	db, ok := s.dbForOrg(w, r)
	if !ok {
		return
	}
	if db.ContainerID == "" {
		if r.URL.Query().Get("follow") == "1" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			return
		}
		writeJSON(w, 200, map[string]string{"logs": ""})
		return
	}
	if r.URL.Query().Get("follow") == "1" {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeErr(w, 500, "stream não suportado")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(200)

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		rc, err := s.core.Driver.Logs(ctx, db.ContainerID, true)
		if err != nil {
			fmt.Fprintf(w, "event: log\ndata: [logs] %s\n\n", err.Error())
			flusher.Flush()
			return
		}
		defer rc.Close()
		scanner := bufio.NewScanner(rc)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			default:
			}
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil && ctx.Err() == nil {
					fmt.Fprintf(w, "event: log\ndata: [logs] %s\n\n", err.Error())
					flusher.Flush()
				}
				return
			}
			line := scanner.Text()
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", strings.ReplaceAll(line, "\n", " "))
			flusher.Flush()
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	rc, err := s.core.Driver.Logs(ctx, db.ContainerID, false)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write(data)
}
