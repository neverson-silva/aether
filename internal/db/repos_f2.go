package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
)

func (s *Store) CreateDatabase(d *domain.Database) error {
	_, err := s.db.Exec(`INSERT INTO databases(id,org_id,project_id,name,engine,version,port,db_name,db_user,pass_enc,mem_mb,storage_mb,status,container_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.OrgID, d.ProjectID, d.Name, string(d.Engine), d.Version, d.Port, d.DBName, d.User, d.PassEnc, d.MemMB, d.StorageMB, d.Status, d.ContainerID, d.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetDatabase(id string) (*domain.Database, error) {
	var d domain.Database
	var engine, created string
	err := s.db.QueryRow(`SELECT id,org_id,project_id,name,engine,version,port,db_name,db_user,pass_enc,mem_mb,storage_mb,status,container_id,created_at FROM databases WHERE id=?`, id).Scan(
		&d.ID, &d.OrgID, &d.ProjectID, &d.Name, &engine, &d.Version, &d.Port, &d.DBName, &d.User, &d.PassEnc, &d.MemMB, &d.StorageMB, &d.Status, &d.ContainerID, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.Engine = domain.DBEngine(engine)
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &d, nil
}

func (s *Store) ListDatabasesByProject(projectID string) ([]domain.Database, error) {
	rows, err := s.db.Query(`SELECT id,org_id,project_id,name,engine,version,port,db_name,db_user,pass_enc,mem_mb,storage_mb,status,container_id,created_at FROM databases WHERE project_id=? ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Database
	for rows.Next() {
		var d domain.Database
		var engine, created string
		if err := rows.Scan(&d.ID, &d.OrgID, &d.ProjectID, &d.Name, &engine, &d.Version, &d.Port, &d.DBName, &d.User, &d.PassEnc, &d.MemMB, &d.StorageMB, &d.Status, &d.ContainerID, &created); err != nil {
			return nil, err
		}
		d.Engine = domain.DBEngine(engine)
		d.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) ListDatabases(orgID string) ([]domain.Database, error) {
	rows, err := s.db.Query(`SELECT id,org_id,project_id,name,engine,version,port,db_name,db_user,pass_enc,mem_mb,storage_mb,status,container_id,created_at FROM databases WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Database
	for rows.Next() {
		var d domain.Database
		var engine, created string
		if err := rows.Scan(&d.ID, &d.OrgID, &d.ProjectID, &d.Name, &engine, &d.Version, &d.Port, &d.DBName, &d.User, &d.PassEnc, &d.MemMB, &d.StorageMB, &d.Status, &d.ContainerID, &created); err != nil {
			return nil, err
		}
		d.Engine = domain.DBEngine(engine)
		d.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDatabaseStatus(id, status, containerID string) error {
	_, err := s.db.Exec(`UPDATE databases SET status=?, container_id=? WHERE id=?`, status, containerID, id)
	return err
}

func (s *Store) DeleteDatabase(id string) error {
	_, err := s.db.Exec(`DELETE FROM databases WHERE id=?`, id)
	return err
}

func (s *Store) CreateCronJob(j *domain.CronJob) error {
	_, err := s.db.Exec(`INSERT INTO cron_jobs(id,app_id,name,schedule,command,enabled,last_run,next_run,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		j.ID, j.AppID, j.Name, j.Schedule, j.Command, boolInt(j.Enabled),
		timeZero(j.LastRun), timeZero(j.NextRun), j.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetCronJob(id string) (*domain.CronJob, error) {
	var j domain.CronJob
	var last, next, created string
	err := s.db.QueryRow(`SELECT id,app_id,name,schedule,command,enabled,last_run,next_run,created_at FROM cron_jobs WHERE id=?`, id).Scan(
		&j.ID, &j.AppID, &j.Name, &j.Schedule, &j.Command, &j.Enabled, &last, &next, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	j.LastRun, _ = time.Parse(time.RFC3339, last)
	j.NextRun, _ = time.Parse(time.RFC3339, next)
	j.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &j, nil
}

func (s *Store) ListCronJobs(appID string) ([]domain.CronJob, error) {
	rows, err := s.db.Query(`SELECT id,app_id,name,schedule,command,enabled,last_run,next_run,created_at FROM cron_jobs WHERE app_id=? ORDER BY name`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CronJob
	for rows.Next() {
		var j domain.CronJob
		var last, next, created string
		if err := rows.Scan(&j.ID, &j.AppID, &j.Name, &j.Schedule, &j.Command, &j.Enabled, &last, &next, &created); err != nil {
			return nil, err
		}
		j.LastRun, _ = time.Parse(time.RFC3339, last)
		j.NextRun, _ = time.Parse(time.RFC3339, next)
		j.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCronJob(j *domain.CronJob) error {
	_, err := s.db.Exec(`UPDATE cron_jobs SET schedule=?,command=?,enabled=?,last_run=?,next_run=? WHERE id=?`,
		j.Schedule, j.Command, boolInt(j.Enabled), timeZero(j.LastRun), timeZero(j.NextRun), j.ID)
	return err
}

func (s *Store) DeleteCronJob(id string) error {
	_, err := s.db.Exec(`DELETE FROM cron_jobs WHERE id=?`, id)
	return err
}

func (s *Store) CreateWorker(w *domain.Worker) error {
	_, err := s.db.Exec(`INSERT INTO workers(id,app_id,name,command,replicas,enabled,status,container_id,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		w.ID, w.AppID, w.Name, w.Command, w.Replicas, boolInt(w.Enabled), w.Status, w.ContainerID, w.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetWorker(id string) (*domain.Worker, error) {
	var w domain.Worker
	var created string
	err := s.db.QueryRow(`SELECT id,app_id,name,command,replicas,enabled,status,container_id,created_at FROM workers WHERE id=?`, id).Scan(
		&w.ID, &w.AppID, &w.Name, &w.Command, &w.Replicas, &w.Enabled, &w.Status, &w.ContainerID, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &w, nil
}

func (s *Store) ListWorkers(appID string) ([]domain.Worker, error) {
	rows, err := s.db.Query(`SELECT id,app_id,name,command,replicas,enabled,status,container_id,created_at FROM workers WHERE app_id=? ORDER BY name`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Worker
	for rows.Next() {
		var w domain.Worker
		var created string
		if err := rows.Scan(&w.ID, &w.AppID, &w.Name, &w.Command, &w.Replicas, &w.Enabled, &w.Status, &w.ContainerID, &created); err != nil {
			return nil, err
		}
		w.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) UpdateWorker(w *domain.Worker) error {
	_, err := s.db.Exec(`UPDATE workers SET command=?,replicas=?,enabled=?,status=?,container_id=? WHERE id=?`,
		w.Command, w.Replicas, boolInt(w.Enabled), w.Status, w.ContainerID, w.ID)
	return err
}

func (s *Store) DeleteWorker(id string) error {
	_, err := s.db.Exec(`DELETE FROM workers WHERE id=?`, id)
	return err
}

func (s *Store) CreateS3Destination(d *domain.S3Destination) error {
	_, err := s.db.Exec(`INSERT INTO s3_destinations(id,org_id,name,endpoint,bucket,region,access_key_enc,secret_key_enc,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		d.ID, d.OrgID, d.Name, d.Endpoint, d.Bucket, d.Region, d.AccessKeyEnc, d.SecretKeyEnc, d.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetS3Destination(id string) (*domain.S3Destination, error) {
	var d domain.S3Destination
	var created string
	err := s.db.QueryRow(`SELECT id,org_id,name,endpoint,bucket,region,access_key_enc,secret_key_enc,created_at FROM s3_destinations WHERE id=?`, id).Scan(
		&d.ID, &d.OrgID, &d.Name, &d.Endpoint, &d.Bucket, &d.Region, &d.AccessKeyEnc, &d.SecretKeyEnc, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &d, nil
}

func (s *Store) ListS3Destinations(orgID string) ([]domain.S3Destination, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,endpoint,bucket,region,access_key_enc,secret_key_enc,created_at FROM s3_destinations WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.S3Destination
	for rows.Next() {
		var d domain.S3Destination
		var created string
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Name, &d.Endpoint, &d.Bucket, &d.Region, &d.AccessKeyEnc, &d.SecretKeyEnc, &created); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeleteS3Destination(id string) error {
	_, err := s.db.Exec(`DELETE FROM s3_destinations WHERE id=?`, id)
	return err
}

func (s *Store) CreateNotificationChannel(c *domain.NotificationChannel) error {
	_, err := s.db.Exec(`INSERT INTO notification_channels(id,org_id,name,type,config_enc,enabled,created_at) VALUES(?,?,?,?,?,?,?)`,
		c.ID, c.OrgID, c.Name, c.Type, c.ConfigEnc, boolInt(c.Enabled), c.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListNotificationChannels(orgID string) ([]domain.NotificationChannel, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,type,config_enc,enabled,created_at FROM notification_channels WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.NotificationChannel
	for rows.Next() {
		var c domain.NotificationChannel
		var created string
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.Type, &c.ConfigEnc, &c.Enabled, &created); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) DeleteNotificationChannel(id string) error {
	_, err := s.db.Exec(`DELETE FROM notification_channels WHERE id=?`, id)
	return err
}

func (s *Store) CreateTemplate(t *domain.Template) error {
	_, err := s.db.Exec(`INSERT INTO templates(id,name,description,category,tags,icon,version,definition,readme,homepage,github,license,installs,featured,editors_choice,verified,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Description, t.Category, strings.Join(t.Tags, ","), t.Icon, t.Version, t.Definition,
		t.Readme, t.Homepage, t.GitHub, t.License, t.Installs, boolInt(t.Featured), boolInt(t.EditorsChoice), boolInt(t.Verified),
		t.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

const templateCols = `id,name,description,category,tags,icon,version,definition,readme,homepage,github,license,installs,featured,editors_choice,verified,updated_at`

func scanTemplate(scanner interface{ Scan(...any) error }) (*domain.Template, error) {
	var t domain.Template
	var tags string
	var featured, editorsChoice, verified, installs int
	var updated string
	err := scanner.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &tags, &t.Icon, &t.Version, &t.Definition,
		&t.Readme, &t.Homepage, &t.GitHub, &t.License, &installs, &featured, &editorsChoice, &verified, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Tags = splitCSV(tags)
	t.Installs = installs
	t.Featured = featured == 1
	t.EditorsChoice = editorsChoice == 1
	t.Verified = verified == 1
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &t, nil
}

func (s *Store) ListTemplates() ([]domain.Template, error) {
	rows, err := s.db.Query(`SELECT ` + templateCols + ` FROM templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) IncrementTemplateInstalls(id string) error {
	_, err := s.db.Exec(`UPDATE templates SET installs=installs+1 WHERE id=?`, id)
	return err
}

func (s *Store) GetTemplate(id string) (*domain.Template, error) {
	row := s.db.QueryRow(`SELECT `+templateCols+` FROM templates WHERE id=?`, id)
	return scanTemplate(row)
}

func (s *Store) CreatePreview(p *domain.Preview) error {
	_, err := s.db.Exec(`INSERT INTO previews(id,app_id,branch,deployment_id,container_id,domain,status,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		p.ID, p.AppID, p.Branch, p.DeploymentID, p.ContainerID, p.Domain, p.Status, p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListPreviews(appID string) ([]domain.Preview, error) {
	rows, err := s.db.Query(`SELECT id,app_id,branch,deployment_id,container_id,domain,status,created_at FROM previews WHERE app_id=? ORDER BY created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Preview
	for rows.Next() {
		var p domain.Preview
		var created string
		if err := rows.Scan(&p.ID, &p.AppID, &p.Branch, &p.DeploymentID, &p.ContainerID, &p.Domain, &p.Status, &created); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPreview(id string) (*domain.Preview, error) {
	var p domain.Preview
	var created string
	err := s.db.QueryRow(`SELECT id,app_id,branch,deployment_id,container_id,domain,status,created_at FROM previews WHERE id=?`, id).Scan(
		&p.ID, &p.AppID, &p.Branch, &p.DeploymentID, &p.ContainerID, &p.Domain, &p.Status, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) UpdatePreview(p *domain.Preview) error {
	_, err := s.db.Exec(`UPDATE previews SET deployment_id=?,container_id=?,domain=?,status=? WHERE id=?`,
		p.DeploymentID, p.ContainerID, p.Domain, p.Status, p.ID)
	return err
}

func (s *Store) DeletePreview(id string) error {
	_, err := s.db.Exec(`DELETE FROM previews WHERE id=?`, id)
	return err
}

func (s *Store) SetTOTP(userID, secretEnc string, enabled bool) error {
	_, err := s.db.Exec(`UPDATE users SET totp_secret_enc=?, totp_enabled=? WHERE id=?`, secretEnc, boolInt(enabled), userID)
	return err
}

func (s *Store) GetTOTP(userID string) (string, bool, error) {
	var secret string
	var enabled int
	err := s.db.QueryRow(`SELECT totp_secret_enc, totp_enabled FROM users WHERE id=?`, userID).Scan(&secret, &enabled)
	if err != nil {
		return "", false, err
	}
	return secret, enabled == 1, nil
}

func timeZero(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *Store) CreateOutWebhook(w *domain.OutWebhook) error {
	_, err := s.db.Exec(`INSERT INTO out_webhooks(id,org_id,name,url,secret_enc,events,enabled,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		w.ID, w.OrgID, w.Name, w.URL, w.SecretEnc, strings.Join(w.Events, ","), boolInt(w.Enabled), w.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListOutWebhooks(orgID string) ([]domain.OutWebhook, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,url,secret_enc,events,enabled,created_at FROM out_webhooks WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OutWebhook
	for rows.Next() {
		var w domain.OutWebhook
		var events, created string
		if err := rows.Scan(&w.ID, &w.OrgID, &w.Name, &w.URL, &w.SecretEnc, &events, &w.Enabled, &created); err != nil {
			return nil, err
		}
		w.Events = splitCSV(events)
		w.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) DeleteOutWebhook(id string) error {
	_, err := s.db.Exec(`DELETE FROM out_webhooks WHERE id=?`, id)
	return err
}

func (s *Store) AllOutWebhooks(orgID string) ([]domain.OutWebhook, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,url,secret_enc,events,enabled,created_at FROM out_webhooks WHERE org_id=? AND enabled=1`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OutWebhook
	for rows.Next() {
		var w domain.OutWebhook
		var events, created string
		if err := rows.Scan(&w.ID, &w.OrgID, &w.Name, &w.URL, &w.SecretEnc, &events, &w.Enabled, &created); err != nil {
			return nil, err
		}
		w.Events = splitCSV(events)
		w.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetRegistrySettings() (domain.RegistrySettings, error) {
	var r domain.RegistrySettings
	var enabled int
	err := s.db.QueryRow(`SELECT id,enabled,host,port,container_id,status FROM registry_settings LIMIT 1`).Scan(&r.ID, &enabled, &r.Host, &r.Port, &r.ContainerID, &r.Status)
	if err == sql.ErrNoRows {
		r.ID = "registry"
		r.Host = "127.0.0.1"
		r.Port = 5000
		r.Status = "stopped"
		return r, nil
	}
	r.Enabled = enabled == 1
	return r, err
}

func (s *Store) SaveRegistrySettings(r *domain.RegistrySettings) error {
	_, err := s.db.Exec(`INSERT INTO registry_settings(id,enabled,host,port,container_id,status) VALUES(?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled, host=excluded.host, port=excluded.port, container_id=excluded.container_id, status=excluded.status`,
		r.ID, boolInt(r.Enabled), r.Host, r.Port, r.ContainerID, r.Status)
	return err
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) CreateServer(srv *domain.Server) error {
	_, err := s.db.Exec(`INSERT INTO servers(id,name,host,role,status,version,labels,cpu_cores,mem_total_bytes,load,last_heartbeat,cluster_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		srv.ID, srv.Name, srv.Host, srv.Role, srv.Status, srv.Version, strings.Join(srv.Labels, ","),
		srv.CPUCores, srv.MemTotalBytes, srv.Load, srv.LastHeartbeat.UTC().Format(time.RFC3339), srv.ClusterID, srv.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListServers() ([]domain.Server, error) {
	rows, err := s.db.Query(`SELECT id,name,host,role,status,version,labels,cpu_cores,mem_total_bytes,load,last_heartbeat,cluster_id,created_at FROM servers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Server
	for rows.Next() {
		var srv domain.Server
		var labels, hb, created string
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Host, &srv.Role, &srv.Status, &srv.Version, &labels,
			&srv.CPUCores, &srv.MemTotalBytes, &srv.Load, &hb, &srv.ClusterID, &created); err != nil {
			return nil, err
		}
		srv.Labels = splitCSV(labels)
		srv.LastHeartbeat, _ = time.Parse(time.RFC3339, hb)
		srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, srv)
	}
	return out, rows.Err()
}

func (s *Store) GetServer(id string) (*domain.Server, error) {
	var srv domain.Server
	var labels, hb, created string
	err := s.db.QueryRow(`SELECT id,name,host,role,status,version,labels,cpu_cores,mem_total_bytes,load,last_heartbeat,cluster_id,created_at FROM servers WHERE id=?`, id).Scan(
		&srv.ID, &srv.Name, &srv.Host, &srv.Role, &srv.Status, &srv.Version, &labels,
		&srv.CPUCores, &srv.MemTotalBytes, &srv.Load, &hb, &srv.ClusterID, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	srv.Labels = splitCSV(labels)
	srv.LastHeartbeat, _ = time.Parse(time.RFC3339, hb)
	srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &srv, nil
}

func (s *Store) UpdateServerHeartbeat(id string, load float64, cpuCores int, memTotal int64, status, version string) error {
	_, err := s.db.Exec(`UPDATE servers SET load=?,cpu_cores=?,mem_total_bytes=?,status=?,version=?,last_heartbeat=? WHERE id=?`,
		load, cpuCores, memTotal, status, version, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) DeleteServer(id string) error {
	_, err := s.db.Exec(`DELETE FROM servers WHERE id=?`, id)
	return err
}

func (s *Store) SaveServerToken(hash, serverID string, ttl time.Duration) error {
	expires := time.Now().UTC().Add(ttl).Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO server_tokens(token_hash,server_id,created_at,expires_at) VALUES(?,?,?,?)`,
		hash, serverID, time.Now().UTC().Format(time.RFC3339), expires)
	return err
}

func (s *Store) ConsumeServerToken(hash string) (string, error) {
	var serverID, expires string
	err := s.db.QueryRow(`SELECT server_id,expires_at FROM server_tokens WHERE token_hash=?`, hash).Scan(&serverID, &expires)
	if err != nil {
		return "", err
	}
	exp, _ := time.Parse(time.RFC3339, expires)
	if time.Now().UTC().After(exp) {
		return "", errors.New("token expirado")
	}
	_, err = s.db.Exec(`DELETE FROM server_tokens WHERE token_hash=?`, hash)
	return serverID, err
}

type ServerCommand struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Store) EnqueueServerCommand(serverID, kind, payload string) error {
	_, err := s.db.Exec(`INSERT INTO server_commands(id,server_id,kind,payload,created_at) VALUES(?,?,?,?,?)`,
		"cmd-"+domain.NewID(), serverID, kind, payload, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) DequeueServerCommands(serverID string, limit int) ([]ServerCommand, error) {
	rows, err := s.db.Query(`SELECT id,kind,payload FROM server_commands WHERE server_id=? AND delivered=0 ORDER BY created_at LIMIT ?`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServerCommand
	for rows.Next() {
		var c ServerCommand
		var payload string
		if err := rows.Scan(&c.ID, &c.Kind, &payload); err != nil {
			return nil, err
		}
		c.Payload = json.RawMessage(payload)
		out = append(out, c)
	}
	if len(out) > 0 {
		ids := make([]any, 0, len(out))
		placeholders := ""
		for i, c := range out {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			ids = append(ids, c.ID)
		}
		_, _ = s.db.Exec(`UPDATE server_commands SET delivered=1 WHERE id IN (`+placeholders+`)`, ids...)
	}
	return out, rows.Err()
}

func (s *Store) GetPolicy(appID string) (domain.AppPolicy, error) {
	var p domain.AppPolicy
	var enabled, up, down, cooldown int
	var updated string
	err := s.db.QueryRow(`SELECT app_id,enabled,cpu_min,cpu_max,mem_min_mb,mem_max_mb,scale_up_pct,scale_down_pct,cooldown_min,updated_at FROM app_policies WHERE app_id=?`, appID).Scan(
		&p.AppID, &enabled, &p.CPUMin, &p.CPUMax, &p.MemMinMB, &p.MemMaxMB, &up, &down, &cooldown, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AppPolicy{
				AppID: appID, Enabled: false, CPUMin: 0.25, CPUMax: 4, MemMinMB: 128, MemMaxMB: 2048,
				ScaleUpPct: 80, ScaleDownPct: 15, CooldownMin: 15,
			}, nil
		}
		return p, err
	}
	p.Enabled = enabled == 1
	p.ScaleUpPct = up
	p.ScaleDownPct = down
	p.CooldownMin = cooldown
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return p, nil
}

func (s *Store) SavePolicy(p *domain.AppPolicy) error {
	_, err := s.db.Exec(`INSERT INTO app_policies(app_id,enabled,cpu_min,cpu_max,mem_min_mb,mem_max_mb,scale_up_pct,scale_down_pct,cooldown_min,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(app_id) DO UPDATE SET enabled=excluded.enabled,cpu_min=excluded.cpu_min,cpu_max=excluded.cpu_max,
		mem_min_mb=excluded.mem_min_mb,mem_max_mb=excluded.mem_max_mb,scale_up_pct=excluded.scale_up_pct,
		scale_down_pct=excluded.scale_down_pct,cooldown_min=excluded.cooldown_min,updated_at=excluded.updated_at`,
		p.AppID, boolInt(p.Enabled), p.CPUMin, p.CPUMax, p.MemMinMB, p.MemMaxMB, p.ScaleUpPct, p.ScaleDownPct, p.CooldownMin,
		p.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListPolicies() ([]domain.AppPolicy, error) {
	rows, err := s.db.Query(`SELECT app_id,enabled,cpu_min,cpu_max,mem_min_mb,mem_max_mb,scale_up_pct,scale_down_pct,cooldown_min,updated_at FROM app_policies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AppPolicy
	for rows.Next() {
		var p domain.AppPolicy
		var enabled, up, down, cooldown int
		var updated string
		if err := rows.Scan(&p.AppID, &enabled, &p.CPUMin, &p.CPUMax, &p.MemMinMB, &p.MemMaxMB, &up, &down, &cooldown, &updated); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		p.ScaleUpPct = up
		p.ScaleDownPct = down
		p.CooldownMin = cooldown
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) AddAutopilotEvent(appID, action, detail string) error {
	_, err := s.db.Exec(`INSERT INTO autopilot_events(id,app_id,action,detail,created_at) VALUES(?,?,?,?,?)`,
		"ap-"+domain.NewID(), appID, action, detail, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListAutopilotEvents(appID string, limit int) ([]domain.AutopilotEvent, error) {
	rows, err := s.db.Query(`SELECT id,app_id,action,detail,created_at FROM autopilot_events WHERE app_id=? ORDER BY created_at DESC LIMIT ?`, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AutopilotEvent
	for rows.Next() {
		var e domain.AutopilotEvent
		var created string
		if err := rows.Scan(&e.ID, &e.AppID, &e.Action, &e.Detail, &created); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateGitOps(g *domain.GitOps) error {
	_, err := s.db.Exec(`INSERT INTO gitops(id,org_id,name,repo_url,branch,path,target_org_id,apply_mode,last_sha,last_status,drift_added,drift_changed,drift_removed,last_sync,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		g.ID, g.OrgID, g.Name, g.RepoURL, g.Branch, g.Path, g.TargetOrgID, g.ApplyMode, g.LastSHA, g.LastStatus,
		g.DriftAdded, g.DriftChanged, g.DriftRemoved, g.LastSync, g.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListGitOps(orgID string) ([]domain.GitOps, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,repo_url,branch,path,target_org_id,apply_mode,last_sha,last_status,drift_added,drift_changed,drift_removed,last_sync,created_at FROM gitops WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GitOps
	for rows.Next() {
		var g domain.GitOps
		var created string
		if err := rows.Scan(&g.ID, &g.OrgID, &g.Name, &g.RepoURL, &g.Branch, &g.Path, &g.TargetOrgID, &g.ApplyMode, &g.LastSHA, &g.LastStatus,
			&g.DriftAdded, &g.DriftChanged, &g.DriftRemoved, &g.LastSync, &created); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) UpdateGitOps(g *domain.GitOps) error {
	_, err := s.db.Exec(`UPDATE gitops SET target_org_id=?,apply_mode=?,last_sha=?,last_status=?,drift_added=?,drift_changed=?,drift_removed=?,last_sync=? WHERE id=?`,
		g.TargetOrgID, g.ApplyMode, g.LastSHA, g.LastStatus, g.DriftAdded, g.DriftChanged, g.DriftRemoved, g.LastSync, g.ID)
	return err
}

func (s *Store) DeleteGitOps(id string) error {
	_, err := s.db.Exec(`DELETE FROM gitops WHERE id=?`, id)
	return err
}

func (s *Store) GetGitOps(id string) (*domain.GitOps, error) {
	var g domain.GitOps
	var created string
	err := s.db.QueryRow(`SELECT id,org_id,name,repo_url,branch,path,target_org_id,apply_mode,last_sha,last_status,drift_added,drift_changed,drift_removed,last_sync,created_at FROM gitops WHERE id=?`, id).Scan(
		&g.ID, &g.OrgID, &g.Name, &g.RepoURL, &g.Branch, &g.Path, &g.TargetOrgID, &g.ApplyMode, &g.LastSHA, &g.LastStatus,
		&g.DriftAdded, &g.DriftChanged, &g.DriftRemoved, &g.LastSync, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &g, nil
}

func (s *Store) CreateMirror(m *domain.RegistryMirror) error {
	_, err := s.db.Exec(`INSERT INTO registry_mirrors(id,name,source,dest,dest_tls_verify,tags_filter,schedule,last_run,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.Name, m.Source, m.Dest, boolInt(m.DestTLSVerify), m.TagsFilter, m.Schedule, m.LastRun, m.Status, m.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListMirrors() ([]domain.RegistryMirror, error) {
	rows, err := s.db.Query(`SELECT id,name,source,dest,dest_tls_verify,tags_filter,schedule,last_run,status,created_at FROM registry_mirrors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RegistryMirror
	for rows.Next() {
		var m domain.RegistryMirror
		var verify int
		var created string
		if err := rows.Scan(&m.ID, &m.Name, &m.Source, &m.Dest, &verify, &m.TagsFilter, &m.Schedule, &m.LastRun, &m.Status, &created); err != nil {
			return nil, err
		}
		m.DestTLSVerify = verify == 1
		m.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) UpdateMirrorStatus(id, status, lastRun string) error {
	_, err := s.db.Exec(`UPDATE registry_mirrors SET status=?,last_run=? WHERE id=?`, status, lastRun, id)
	return err
}

func (s *Store) DeleteMirror(id string) error {
	_, err := s.db.Exec(`DELETE FROM registry_mirrors WHERE id=?`, id)
	return err
}

func (s *Store) CreateSnapshot(sn *domain.Snapshot) error {
	_, err := s.db.Exec(`INSERT INTO snapshots(id,org_id,app_id,volume,name,size,chunks,dedup_saved,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		sn.ID, sn.OrgID, sn.AppID, sn.Volume, sn.Name, sn.Size, sn.Chunks, sn.DedupSaved, sn.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListSnapshots(orgID string) ([]domain.Snapshot, error) {
	rows, err := s.db.Query(`SELECT id,org_id,app_id,volume,name,size,chunks,dedup_saved,created_at FROM snapshots WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Snapshot
	for rows.Next() {
		var sn domain.Snapshot
		var created string
		if err := rows.Scan(&sn.ID, &sn.OrgID, &sn.AppID, &sn.Volume, &sn.Name, &sn.Size, &sn.Chunks, &sn.DedupSaved, &created); err != nil {
			return nil, err
		}
		sn.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSnapshot(id string) error {
	_, err := s.db.Exec(`DELETE FROM snapshots WHERE id=?`, id)
	return err
}

func (s *Store) GetBranding(orgID string) (domain.Branding, error) {
	var b domain.Branding
	var dark int
	var updated string
	err := s.db.QueryRow(`SELECT org_id,name,logo_url,primary_color,accent_color,dark_mode,updated_at FROM branding WHERE org_id=?`, orgID).Scan(
		&b.OrgID, &b.Name, &b.LogoURL, &b.PrimaryColor, &b.AccentColor, &dark, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Branding{OrgID: orgID}, nil
		}
		return b, err
	}
	b.DarkMode = dark == 1
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return b, nil
}

func (s *Store) SaveBranding(b *domain.Branding) error {
	_, err := s.db.Exec(`INSERT INTO branding(org_id,name,logo_url,primary_color,accent_color,dark_mode,updated_at) VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(org_id) DO UPDATE SET name=excluded.name,logo_url=excluded.logo_url,primary_color=excluded.primary_color,
		accent_color=excluded.accent_color,dark_mode=excluded.dark_mode,updated_at=excluded.updated_at`,
		b.OrgID, b.Name, b.LogoURL, b.PrimaryColor, b.AccentColor, boolInt(b.DarkMode), b.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) CreatePipeline(p *domain.Pipeline) error {
	raw, _ := json.Marshal(p.Stages)
	_, err := s.db.Exec(`INSERT INTO pipelines(id,org_id,app_id,name,trigger,stages,enabled,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		p.ID, p.OrgID, p.AppID, p.Name, p.Trigger, string(raw), boolInt(p.Enabled), p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListPipelines(orgID string) ([]domain.Pipeline, error) {
	rows, err := s.db.Query(`SELECT id,org_id,app_id,name,trigger,stages,enabled,created_at FROM pipelines WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Pipeline
	for rows.Next() {
		var p domain.Pipeline
		var stages string
		var enabled int
		var created string
		if err := rows.Scan(&p.ID, &p.OrgID, &p.AppID, &p.Name, &p.Trigger, &stages, &enabled, &created); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(stages), &p.Stages)
		p.Enabled = enabled == 1
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPipeline(id string) (*domain.Pipeline, error) {
	var p domain.Pipeline
	var stages string
	var enabled int
	var created string
	err := s.db.QueryRow(`SELECT id,org_id,app_id,name,trigger,stages,enabled,created_at FROM pipelines WHERE id=?`, id).Scan(
		&p.ID, &p.OrgID, &p.AppID, &p.Name, &p.Trigger, &stages, &enabled, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	json.Unmarshal([]byte(stages), &p.Stages)
	p.Enabled = enabled == 1
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) DeletePipeline(id string) error {
	_, err := s.db.Exec(`DELETE FROM pipelines WHERE id=?`, id)
	return err
}

func (s *Store) CreatePipelineRun(r *domain.PipelineRun) error {
	_, err := s.db.Exec(`INSERT INTO pipeline_runs(id,pipeline_id,status,trigger,log,started_at,finished_at) VALUES(?,?,?,?,?,?,?)`,
		r.ID, r.PipelineID, r.Status, r.Trigger, r.Log, r.StartedAt.UTC().Format(time.RFC3339), r.FinishedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UpdatePipelineRun(r *domain.PipelineRun) error {
	finished := ""
	if !r.FinishedAt.IsZero() {
		finished = r.FinishedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`UPDATE pipeline_runs SET status=?,log=?,finished_at=? WHERE id=?`, r.Status, r.Log, finished, r.ID)
	return err
}

func (s *Store) ListPipelineRuns(pipelineID string, limit int) ([]domain.PipelineRun, error) {
	rows, err := s.db.Query(`SELECT id,pipeline_id,status,trigger,log,started_at,finished_at FROM pipeline_runs WHERE pipeline_id=? ORDER BY started_at DESC LIMIT ?`, pipelineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PipelineRun
	for rows.Next() {
		var r domain.PipelineRun
		var started, finished string
		if err := rows.Scan(&r.ID, &r.PipelineID, &r.Status, &r.Trigger, &r.Log, &started, &finished); err != nil {
			return nil, err
		}
		r.StartedAt, _ = time.Parse(time.RFC3339, started)
		if finished != "" {
			r.FinishedAt, _ = time.Parse(time.RFC3339, finished)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateOIDCProvider(p *domain.OIDCProvider) error {
	_, err := s.db.Exec(`INSERT INTO oidc_providers(id,org_id,name,issuer,client_id,client_secret_enc,scopes,enabled,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		p.ID, p.OrgID, p.Name, p.Issuer, p.ClientID, p.ClientSecretEnc, p.Scopes, boolInt(p.Enabled), p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListOIDCProviders(orgID string) ([]domain.OIDCProvider, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,issuer,client_id,client_secret_enc,scopes,enabled,created_at FROM oidc_providers WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OIDCProvider
	for rows.Next() {
		var p domain.OIDCProvider
		var enabled int
		var created string
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Issuer, &p.ClientID, &p.ClientSecretEnc, &p.Scopes, &enabled, &created); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetOIDCProvider(id string) (*domain.OIDCProvider, error) {
	var p domain.OIDCProvider
	var enabled int
	var created string
	err := s.db.QueryRow(`SELECT id,org_id,name,issuer,client_id,client_secret_enc,scopes,enabled,created_at FROM oidc_providers WHERE id=?`, id).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.Issuer, &p.ClientID, &p.ClientSecretEnc, &p.Scopes, &enabled, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.Enabled = enabled == 1
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) DeleteOIDCProvider(id string) error {
	_, err := s.db.Exec(`DELETE FROM oidc_providers WHERE id=?`, id)
	return err
}

func (s *Store) CreateCluster(c *domain.Cluster) error {
	_, err := s.db.Exec(`INSERT INTO clusters(id,org_id,name,labels,created_at) VALUES(?,?,?,?,?)`,
		c.ID, c.OrgID, c.Name, strings.Join(c.Labels, ","), c.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListClusters(orgID string) ([]domain.Cluster, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,labels,created_at FROM clusters WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Cluster
	for rows.Next() {
		var c domain.Cluster
		var labels, created string
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &labels, &created); err != nil {
			return nil, err
		}
		c.Labels = splitCSV(labels)
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCluster(id string) (*domain.Cluster, error) {
	var c domain.Cluster
	var labels, created string
	err := s.db.QueryRow(`SELECT id,org_id,name,labels,created_at FROM clusters WHERE id=?`, id).Scan(
		&c.ID, &c.OrgID, &c.Name, &labels, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.Labels = splitCSV(labels)
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &c, nil
}

func (s *Store) DeleteCluster(id string) error {
	_, err := s.db.Exec(`DELETE FROM clusters WHERE id=?`, id)
	return err
}

func (s *Store) SetServerCluster(serverID, clusterID string) error {
	_, err := s.db.Exec(`UPDATE servers SET cluster_id=? WHERE id=?`, clusterID, serverID)
	return err
}

func (s *Store) SetAppCluster(appID, clusterID string) error {
	_, err := s.db.Exec(`UPDATE apps SET cluster_id=? WHERE id=?`, clusterID, appID)
	return err
}

func (s *Store) HasUsers() (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n > 0, err
}

func (s *Store) CreateEnvironment(e *domain.Environment) error {
	_, err := s.db.Exec(`INSERT INTO environments(id,project_id,name,slug,description,color,is_default,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		e.ID, e.ProjectID, e.Name, e.Slug, e.Description, e.Color, boolInt(e.IsDefault),
		e.CreatedAt.UTC().Format(time.RFC3339), e.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListEnvironments(projectID string) ([]domain.Environment, error) {
	rows, err := s.db.Query(`SELECT id,project_id,name,slug,description,color,is_default,created_at,updated_at FROM environments WHERE project_id=? ORDER BY is_default DESC, name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Environment
	for rows.Next() {
		e, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (s *Store) GetEnvironment(id string) (*domain.Environment, error) {
	row := s.db.QueryRow(`SELECT id,project_id,name,slug,description,color,is_default,created_at,updated_at FROM environments WHERE id=?`, id)
	return scanEnvironment(row)
}

func (s *Store) GetEnvironmentBySlug(projectID, slug string) (*domain.Environment, error) {
	row := s.db.QueryRow(`SELECT id,project_id,name,slug,description,color,is_default,created_at,updated_at FROM environments WHERE project_id=? AND slug=?`, projectID, slug)
	return scanEnvironment(row)
}

func scanEnvironment(scanner interface{ Scan(...any) error }) (*domain.Environment, error) {
	var e domain.Environment
	var def int
	var created, updated string
	err := scanner.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.Description, &e.Color, &def, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	e.IsDefault = def == 1
	e.CreatedAt, _ = time.Parse(time.RFC3339, created)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &e, nil
}

func (s *Store) UpdateEnvironment(e *domain.Environment) error {
	_, err := s.db.Exec(`UPDATE environments SET name=?,slug=?,description=?,color=?,updated_at=? WHERE id=?`,
		e.Name, e.Slug, e.Description, e.Color, time.Now().UTC().Format(time.RFC3339), e.ID)
	return err
}

func (s *Store) DeleteEnvironment(id string) error {
	_, err := s.db.Exec(`DELETE FROM environments WHERE id=?`, id)
	return err
}

func (s *Store) SetDefaultEnvironment(projectID, envID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE environments SET is_default=0 WHERE project_id=?`, projectID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`UPDATE environments SET is_default=1,updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), envID); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) ListAppsByEnvironment(envID string) ([]domain.App, error) {
	rows, err := s.db.Query(`SELECT `+appCols+` FROM apps WHERE environment_id=? ORDER BY name`, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.App
	for rows.Next() {
		a, err := scanApp(func(dest ...any) error { return rows.Scan(dest...) })
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Store) CreateProjectWithEnv(p *domain.Project, e *domain.Environment) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO projects(id,org_id,name,slug,description,color,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		p.ID, p.OrgID, p.Name, p.Slug, p.Description, p.Color, p.CreatedAt.UTC().Format(time.RFC3339), p.UpdatedAt.UTC().Format(time.RFC3339)); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`INSERT INTO environments(id,project_id,name,slug,description,color,is_default,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		e.ID, e.ProjectID, e.Name, e.Slug, e.Description, e.Color, boolInt(e.IsDefault),
		e.CreatedAt.UTC().Format(time.RFC3339), e.UpdatedAt.UTC().Format(time.RFC3339)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) ListEnvVariables(environmentID string) ([]domain.EnvironmentVariable, error) {
	rows, err := s.db.Query(`SELECT id,project_id,environment_id,key,value,is_secret,created_at,updated_at FROM env_variables WHERE environment_id=? ORDER BY key`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EnvironmentVariable
	for rows.Next() {
		var v domain.EnvironmentVariable
		var secret int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.EnvironmentID, &v.Key, &v.Value, &secret, &created, &updated); err != nil {
			return nil, err
		}
		v.IsSecret = secret == 1
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetEnvVariable(environmentID, key string) (*domain.EnvironmentVariable, error) {
	var v domain.EnvironmentVariable
	var secret int
	var created, updated string
	err := s.db.QueryRow(`SELECT id,project_id,environment_id,key,value,is_secret,created_at,updated_at FROM env_variables WHERE environment_id=? AND key=?`, environmentID, key).Scan(
		&v.ID, &v.ProjectID, &v.EnvironmentID, &v.Key, &v.Value, &secret, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	v.IsSecret = secret == 1
	v.CreatedAt, _ = time.Parse(time.RFC3339, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &v, nil
}

func (s *Store) UpsertEnvVariable(v *domain.EnvironmentVariable) error {
	_, err := s.db.Exec(`INSERT INTO env_variables(id,project_id,environment_id,key,value,is_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(environment_id,key) DO UPDATE SET value=excluded.value,is_secret=excluded.is_secret,updated_at=excluded.updated_at`,
		v.ID, v.ProjectID, v.EnvironmentID, v.Key, v.Value, boolInt(v.IsSecret),
		v.CreatedAt.UTC().Format(time.RFC3339), v.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) DeleteEnvVariable(environmentID, key string) error {
	_, err := s.db.Exec(`DELETE FROM env_variables WHERE environment_id=? AND key=?`, environmentID, key)
	return err
}

func (s *Store) ReplaceEnvVariables(environmentID, projectID string, vars []domain.EnvironmentVariable) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM env_variables WHERE environment_id=?`, environmentID); err != nil {
		tx.Rollback()
		return err
	}
	for _, v := range vars {
		if _, err := tx.Exec(`INSERT INTO env_variables(id,project_id,environment_id,key,value,is_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			v.ID, v.ProjectID, v.EnvironmentID, v.Key, v.Value, boolInt(v.IsSecret),
			v.CreatedAt.UTC().Format(time.RFC3339), v.UpdatedAt.UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) AddVariableAudit(a *domain.VariableAudit) error {
	_, err := s.db.Exec(`INSERT INTO variable_audit(id,project_id,environment_id,action,var_key,previous_value,created_at) VALUES(?,?,?,?,?,?,?)`,
		a.ID, a.ProjectID, a.EnvironmentID, a.Action, a.Key, a.PreviousValue, a.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListVariableAudit(environmentID string, limit int) ([]domain.VariableAudit, error) {
	rows, err := s.db.Query(`SELECT id,project_id,environment_id,action,var_key,previous_value,created_at FROM variable_audit WHERE environment_id=? ORDER BY created_at DESC LIMIT ?`, environmentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VariableAudit
	for rows.Next() {
		var a domain.VariableAudit
		var created string
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.EnvironmentID, &a.Action, &a.Key, &a.PreviousValue, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ListProjectVariables(projectID string) ([]domain.ProjectVariable, error) {
	rows, err := s.db.Query(`SELECT id,project_id,key,value,is_secret,created_at,updated_at FROM project_variables WHERE project_id=? ORDER BY key`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ProjectVariable
	for rows.Next() {
		var v domain.ProjectVariable
		var secret int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Key, &v.Value, &secret, &created, &updated); err != nil {
			return nil, err
		}
		v.IsSecret = secret == 1
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetProjectVariable(projectID, key string) (*domain.ProjectVariable, error) {
	var v domain.ProjectVariable
	var secret int
	var created, updated string
	err := s.db.QueryRow(`SELECT id,project_id,key,value,is_secret,created_at,updated_at FROM project_variables WHERE project_id=? AND key=?`, projectID, key).Scan(
		&v.ID, &v.ProjectID, &v.Key, &v.Value, &secret, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	v.IsSecret = secret == 1
	v.CreatedAt, _ = time.Parse(time.RFC3339, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &v, nil
}

func (s *Store) UpsertProjectVariable(v *domain.ProjectVariable) error {
	_, err := s.db.Exec(`INSERT INTO project_variables(id,project_id,key,value,is_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(project_id,key) DO UPDATE SET value=excluded.value,is_secret=excluded.is_secret,updated_at=excluded.updated_at`,
		v.ID, v.ProjectID, v.Key, v.Value, boolInt(v.IsSecret),
		v.CreatedAt.UTC().Format(time.RFC3339), v.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) DeleteProjectVariable(projectID, key string) error {
	_, err := s.db.Exec(`DELETE FROM project_variables WHERE project_id=? AND key=?`, projectID, key)
	return err
}

func (s *Store) ReplaceProjectVariables(projectID string, vars []domain.ProjectVariable) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project_variables WHERE project_id=?`, projectID); err != nil {
		tx.Rollback()
		return err
	}
	for _, v := range vars {
		if _, err := tx.Exec(`INSERT INTO project_variables(id,project_id,key,value,is_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
			v.ID, v.ProjectID, v.Key, v.Value, boolInt(v.IsSecret),
			v.CreatedAt.UTC().Format(time.RFC3339), v.UpdatedAt.UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListVariableAuditByProject(projectID string, limit int) ([]domain.VariableAudit, error) {
	rows, err := s.db.Query(`SELECT id,project_id,environment_id,action,var_key,previous_value,created_at FROM variable_audit WHERE project_id=? AND environment_id='' ORDER BY created_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VariableAudit
	for rows.Next() {
		var a domain.VariableAudit
		var created string
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.EnvironmentID, &a.Action, &a.Key, &a.PreviousValue, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CreateNotification(n *domain.Notification) error {
	_, err := s.db.Exec(`INSERT INTO notifications(id,org_id,type,message,payload,read,created_at) VALUES(?,?,?,?,?,?,?)`,
		n.ID, n.OrgID, n.Type, n.Message, n.Payload, boolInt(n.Read), n.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListNotifications(orgID, before string, limit int) ([]domain.Notification, error) {
	limit = min(limit, 100)
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id,org_id,type,message,payload,read,created_at FROM notifications WHERE org_id=?`
	args := []any{orgID}
	if before != "" {
		q += ` AND created_at < (SELECT created_at FROM notifications WHERE id=$2)`
		args = append(args, before)
	}
	q += ` ORDER BY created_at DESC LIMIT ` + strconv.Itoa(limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		var read int
		var created string
		if err := rows.Scan(&n.ID, &n.OrgID, &n.Type, &n.Message, &n.Payload, &read, &created); err != nil {
			return nil, err
		}
		n.Read = read == 1
		n.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) UnreadNotificationCount(orgID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE org_id=? AND read=0`, orgID).Scan(&n)
	return n, err
}

func (s *Store) MarkNotificationRead(orgID, id string) error {
	_, err := s.db.Exec(`UPDATE notifications SET read=1 WHERE id=? AND org_id=?`, id, orgID)
	return err
}

func (s *Store) MarkAllNotificationsRead(orgID string) error {
	_, err := s.db.Exec(`UPDATE notifications SET read=1 WHERE org_id=? AND read=0`, orgID)
	return err
}

func (s *Store) SaveDeploymentPlan(p *domain.DeploymentPlan) error {
	_, err := s.db.Exec(`INSERT INTO deployment_plans(id,app_id,framework,library,package_manager,runtime,build_command,install_command,output_dir,app_type,web_server,container_port,spa_fallback,index_file,nginx_conf,dockerfile,warnings,detected_at,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(app_id) DO UPDATE SET framework=excluded.framework,library=excluded.library,package_manager=excluded.package_manager,runtime=excluded.runtime,build_command=excluded.build_command,install_command=excluded.install_command,output_dir=excluded.output_dir,app_type=excluded.app_type,web_server=excluded.web_server,container_port=excluded.container_port,spa_fallback=excluded.spa_fallback,index_file=excluded.index_file,nginx_conf=excluded.nginx_conf,dockerfile=excluded.dockerfile,warnings=excluded.warnings,detected_at=excluded.detected_at`,
		p.ID, p.AppID, p.Framework, p.Library, p.PackageManager, p.Runtime, p.BuildCommand, p.InstallCommand,
		p.OutputDir, p.AppType, p.WebServer, p.ContainerPort, boolInt(p.SPAFallback), p.IndexFile,
		p.NginxConf, p.Dockerfile, strings.Join(p.Warnings, "\n"), p.DetectedAt.UTC().Format(time.RFC3339),
		p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetDeploymentPlan(appID string) (*domain.DeploymentPlan, error) {
	var p domain.DeploymentPlan
	var fallback int
	var warnings, detected, created string
	err := s.db.QueryRow(`SELECT id,app_id,framework,library,package_manager,runtime,build_command,install_command,output_dir,app_type,web_server,container_port,spa_fallback,index_file,nginx_conf,dockerfile,warnings,detected_at,created_at FROM deployment_plans WHERE app_id=?`, appID).Scan(
		&p.ID, &p.AppID, &p.Framework, &p.Library, &p.PackageManager, &p.Runtime, &p.BuildCommand, &p.InstallCommand,
		&p.OutputDir, &p.AppType, &p.WebServer, &p.ContainerPort, &fallback, &p.IndexFile,
		&p.NginxConf, &p.Dockerfile, &warnings, &detected, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.SPAFallback = fallback == 1
	if warnings != "" {
		p.Warnings = strings.Split(warnings, "\n")
	}
	p.DetectedAt, _ = time.Parse(time.RFC3339, detected)
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

// ---- Multi-tenancy: project assignments & audit log ----

func (s *Store) AddProjectAssignment(a *domain.ProjectAssignment) error {
	_, err := s.db.Exec(`INSERT INTO project_assignments(org_id,user_id,project_id,created_at) VALUES(?,?,?,?) ON CONFLICT(org_id,user_id,project_id) DO NOTHING`,
		a.OrgID, a.UserID, a.ProjectID, a.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) RemoveProjectAssignment(orgID, userID, projectID string) error {
	_, err := s.db.Exec(`DELETE FROM project_assignments WHERE org_id=? AND user_id=? AND project_id=?`, orgID, userID, projectID)
	return err
}

func (s *Store) RemoveProjectAssignmentsForUser(orgID, userID string) error {
	_, err := s.db.Exec(`DELETE FROM project_assignments WHERE org_id=? AND user_id=?`, orgID, userID)
	return err
}

func (s *Store) DeleteMember(orgID, userID string) error {
	_, err := s.db.Exec(`DELETE FROM members WHERE org_id=? AND user_id=?`, orgID, userID)
	return err
}

func (s *Store) ListProjectAssignments(orgID, userID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT project_id FROM project_assignments WHERE org_id=? AND user_id=?`, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		out = append(out, pid)
	}
	return out, rows.Err()
}

func (s *Store) ProjectAssigned(orgID, userID, projectID string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM project_assignments WHERE org_id=? AND user_id=? AND project_id=?`, orgID, userID, projectID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) CreateAudit(a *domain.AuditLog) error {
	_, err := s.db.Exec(`INSERT INTO audit_logs(id,org_id,user_id,action,resource_type,resource_id,details,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		a.ID, a.OrgID, a.UserID, a.Action, a.ResourceType, a.ResourceID, a.Details, a.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListAudit(orgID string, limit int) ([]domain.AuditLog, error) {
	rows, err := s.db.Query(`SELECT id,org_id,user_id,action,resource_type,resource_id,details,created_at FROM audit_logs WHERE org_id=? ORDER BY created_at DESC LIMIT ?`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		var created string
		if err := rows.Scan(&a.ID, &a.OrgID, &a.UserID, &a.Action, &a.ResourceType, &a.ResourceID, &a.Details, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, a)
	}
	return out, rows.Err()
}
