package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/security"
)

var (
	ErrNotFound = errors.New("not found")
)

type Store struct {
	db *SQL
	// Secrets é a única camada de criptografia. Quando definido, campos
	// sensíveis (compose_yaml, deploy_spec, env_snapshot) são criptografados
	// no write e descriptografados no read de forma transparente.
	Secrets *security.Secrets
}

func NewStore(db *SQL) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB { return s.db.DB }

func scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var created string
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, created)
	u.CreatedAt = t
	return &u, nil
}

func scanUserGlobal(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var created string
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.GlobalRole, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, created)
	u.CreatedAt = t
	return &u, nil
}

const userCols = `id,email,name,password_hash,global_role,created_at`

func (s *Store) CreateUser(u *domain.User) error {
	_, err := s.db.Exec(`INSERT INTO users(id,email,name,password_hash,global_role,created_at) VALUES(?,?,?,?,?,?)`,
		u.ID, u.Email, u.Name, u.PasswordHash, u.GlobalRole, u.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetUserByEmail(email string) (*domain.User, error) {
	return scanUserGlobal(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE email=?`, email))
}

func (s *Store) GetUser(id string) (*domain.User, error) {
	return scanUserGlobal(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE id=?`, id))
}

func (s *Store) SetGlobalRole(userID, role string) error {
	_, err := s.db.Exec(`UPDATE users SET global_role=? WHERE id=?`, role, userID)
	return err
}

func (s *Store) CreateOrg(o *domain.Org) error {
	if o.Slug == "" {
		o.Slug = slugify(o.Name)
	}
	_, err := s.db.Exec(`INSERT INTO orgs(id,slug,name,description,avatar,color,owner_user_id,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		o.ID, o.Slug, o.Name, o.Description, o.Avatar, o.Color, o.OwnerUserID, o.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

const orgCols = `id,slug,name,description,avatar,color,owner_user_id,created_at,updated_at,deleted_at`

func scanOrg(row *sql.Row) (*domain.Org, error) {
	var o domain.Org
	var created, updated, deleted string
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.Description, &o.Avatar, &o.Color, &o.OwnerUserID, &created, &updated, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	o.CreatedAt, _ = time.Parse(time.RFC3339, created)
	o.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	o.DeletedAt, _ = time.Parse(time.RFC3339, deleted)
	return &o, nil
}

func slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := strings.ToLower(re.ReplaceAllString(strings.TrimSpace(s), "-"))
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "org"
	}
	return slug
}

func (s *Store) ListOrgs() ([]domain.Org, error) {
	rows, err := s.db.Query(`SELECT ` + orgCols + ` FROM orgs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Org
	for rows.Next() {
		var o domain.Org
		var created, updated, deleted string
		if err := rows.Scan(&o.ID, &o.Slug, &o.Name, &o.Description, &o.Avatar, &o.Color, &o.OwnerUserID, &created, &updated, &deleted); err != nil {
			return nil, err
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, created)
		o.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		o.DeletedAt, _ = time.Parse(time.RFC3339, deleted)
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) GetOrg(id string) (*domain.Org, error) {
	return scanOrg(s.db.QueryRow(`SELECT `+orgCols+` FROM orgs WHERE id=? AND deleted_at=''`, id))
}

func (s *Store) GetOrgBySlug(slug string) (*domain.Org, error) {
	return scanOrg(s.db.QueryRow(`SELECT `+orgCols+` FROM orgs WHERE slug=? AND deleted_at=''`, slug))
}

func (s *Store) UpdateOrg(o *domain.Org) error {
	_, err := s.db.Exec(`UPDATE orgs SET slug=?,name=?,description=?,avatar=?,color=?,updated_at=? WHERE id=?`,
		o.Slug, o.Name, o.Description, o.Avatar, o.Color, o.UpdatedAt.UTC().Format(time.RFC3339), o.ID)
	return err
}

func (s *Store) DeleteOrg(id string) error {
	_, err := s.db.Exec(`UPDATE orgs SET deleted_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ListOrgsForUser returns the organizations the user belongs to, with their role.
func (s *Store) ListOrgsForUser(userID string) ([]domain.Member, error) {
	rows, err := s.db.Query(`SELECT org_id,user_id,role FROM members WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		var role string
		if err := rows.Scan(&m.OrgID, &m.UserID, &role); err != nil {
			return nil, err
		}
		m.Role = domain.Role(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateMember(m *domain.Member) error {
	_, err := s.db.Exec(`INSERT INTO members(org_id,user_id,role) VALUES(?,?,?)`, m.OrgID, m.UserID, string(m.Role))
	return err
}

func (s *Store) SetMemberRole(orgID, userID string, role domain.Role) error {
	_, err := s.db.Exec(`UPDATE members SET role=? WHERE org_id=? AND user_id=?`, string(role), orgID, userID)
	return err
}

func (s *Store) GetMember(orgID, userID string) (*domain.Member, error) {
	var m domain.Member
	var role string
	err := s.db.QueryRow(`SELECT org_id,user_id,role FROM members WHERE org_id=? AND user_id=?`, orgID, userID).Scan(
		&m.OrgID, &m.UserID, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.Role = domain.Role(role)
	return &m, nil
}

func (s *Store) ListMembers(orgID string) ([]domain.Member, error) {
	rows, err := s.db.Query(`SELECT org_id,user_id,role FROM members WHERE org_id=?`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		var role string
		if err := rows.Scan(&m.OrgID, &m.UserID, &role); err != nil {
			return nil, err
		}
		m.Role = domain.Role(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateApiKey(k *domain.ApiKey) error {
	scopes, _ := json.Marshal(k.Scopes)
	_, err := s.db.Exec(`INSERT INTO api_keys(id,org_id,user_id,name,key_hash,scopes,created_at,last_used_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		k.ID, k.OrgID, k.UserID, k.Name, k.KeyHash, string(scopes),
		k.CreatedAt.UTC().Format(time.RFC3339), "", "")
	return err
}

func (s *Store) GetApiKeyByHash(hash string) (*domain.ApiKey, error) {
	var k domain.ApiKey
	var scopes, created, lastUsed, expires string
	err := s.db.QueryRow(`SELECT id,org_id,user_id,name,key_hash,scopes,created_at,last_used_at,expires_at FROM api_keys WHERE key_hash=?`, hash).Scan(
		&k.ID, &k.OrgID, &k.UserID, &k.Name, &k.KeyHash, &scopes, &created, &lastUsed, &expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	json.Unmarshal([]byte(scopes), &k.Scopes)
	k.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if lastUsed != "" {
		k.LastUsedAt, _ = time.Parse(time.RFC3339, lastUsed)
	}
	if expires != "" {
		k.ExpiresAt, _ = time.Parse(time.RFC3339, expires)
	}
	return &k, nil
}

func (s *Store) ListApiKeys(orgID string) ([]domain.ApiKey, error) {
	rows, err := s.db.Query(`SELECT id,org_id,user_id,name,key_hash,scopes,created_at,last_used_at,expires_at FROM api_keys WHERE org_id=?`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ApiKey
	for rows.Next() {
		var k domain.ApiKey
		var scopes, created, lastUsed, expires string
		if err := rows.Scan(&k.ID, &k.OrgID, &k.UserID, &k.Name, &k.KeyHash, &scopes, &created, &lastUsed, &expires); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(scopes), &k.Scopes)
		k.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if lastUsed != "" {
			k.LastUsedAt, _ = time.Parse(time.RFC3339, lastUsed)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) TouchApiKey(id string) error {
	_, err := s.db.Exec(`UPDATE api_keys SET last_used_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) DeleteApiKey(id string) error {
	_, err := s.db.Exec(`DELETE FROM api_keys WHERE id=?`, id)
	return err
}

func (s *Store) UpdateProjectDetails(id, description, color string) error {
	_, err := s.db.Exec(`UPDATE projects SET description=?, color=?, updated_at=? WHERE id=?`,
		description, color, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) CreateProject(p *domain.Project) error {
	_, err := s.db.Exec(`INSERT INTO projects(id,org_id,name,created_at) VALUES(?,?,?,?)`,
		p.ID, p.OrgID, p.Name, p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetProject(id string) (*domain.Project, error) {
	return scanProject(s.db.QueryRow(`SELECT id,org_id,name,slug,description,color,created_at,updated_at,deleted_at FROM projects WHERE id=? AND deleted_at=''`, id))
}

func scanProject(row *sql.Row) (*domain.Project, error) {
	var p domain.Project
	var created, updated, deleted string
	if err := row.Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.Description, &p.Color, &created, &updated, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	p.DeletedAt, _ = time.Parse(time.RFC3339, deleted)
	return &p, nil
}

func (s *Store) ListProjects(orgID string) ([]domain.Project, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,slug,description,color,created_at,updated_at,deleted_at FROM projects WHERE org_id=? AND deleted_at='' ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Project
	for rows.Next() {
		var p domain.Project
		var created, updated, deleted string
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.Description, &p.Color, &created, &updated, &deleted); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		p.DeletedAt, _ = time.Parse(time.RFC3339, deleted)
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanApp(scan func(dest ...any) error) (*domain.App, error) {
	var a domain.App
	var st, image, gitURL, branch, dockerfile, buildType, previewDomain, serverID, clusterID string
	var cpus string
	var mem int64
	var port int
	var hcEnabled, hcInterval, hcTimeout, hcRetries int
	var hcPath, webhookSecret string
	var imageRetention int
	var storageMB int64
	var uploadID, installCmd, buildCmd, startCmd, rootFolder, distFolder, watchPaths string
	var created, updated string
	err := scan(&a.ID, &a.OrgID, &a.ProjectID, &a.EnvironmentID, &a.Name, &st, &image, &gitURL, &branch,
		&dockerfile, &port, &cpus, &mem, &storageMB, &hcEnabled, &hcPath, &hcInterval, &hcTimeout, &hcRetries,
		&buildType, &previewDomain, &serverID, &clusterID, &webhookSecret, &imageRetention,
		&uploadID, &installCmd, &buildCmd, &startCmd, &rootFolder, &distFolder, &watchPaths, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.WebhookSecret = webhookSecret
	a.ImageRetention = imageRetention
	a.SourceType = domain.SourceType(st)
	a.Image = image
	a.GitURL = gitURL
	a.GitBranch = branch
	a.Dockerfile = dockerfile
	a.BuildType = buildType
	a.PreviewDomain = previewDomain
	a.ServerID = serverID
	a.Port = port
	a.Resources = domain.Resources{CPUs: cpus, MemMB: mem, StorageMB: storageMB}
	a.UploadID = uploadID
	a.InstallCommand = installCmd
	a.BuildCommand = buildCmd
	a.StartCommand = startCmd
	a.RootFolder = rootFolder
	a.DistFolder = distFolder
	a.WatchPaths = watchPaths
	a.HealthCheck = domain.HealthCheck{
		Enabled:    hcEnabled == 1,
		Path:       hcPath,
		IntervalMS: hcInterval,
		TimeoutMS:  hcTimeout,
		Retries:    hcRetries,
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, created)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &a, nil
}

const appCols = `id,org_id,project_id,environment_id,name,source_type,image,git_url,git_branch,dockerfile,port,cpus,mem_mb,storage_mb,hc_enabled,hc_path,hc_interval_ms,hc_timeout_ms,hc_retries,build_type,preview_domain,server_id,cluster_id,webhook_secret,image_retention,upload_id,install_command,build_command,start_command,root_folder,dist_folder,watch_paths,created_at,updated_at`

func (s *Store) CreateApp(a *domain.App) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO apps(`+appCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.OrgID, a.ProjectID, a.EnvironmentID, a.Name, string(a.SourceType), a.Image, a.GitURL, a.GitBranch,
		a.Dockerfile, a.Port, a.Resources.CPUs, a.Resources.MemMB, a.Resources.StorageMB, boolInt(a.HealthCheck.Enabled),
		a.HealthCheck.Path, a.HealthCheck.IntervalMS, a.HealthCheck.TimeoutMS, a.HealthCheck.Retries,
		a.BuildType, a.PreviewDomain, a.ServerID, a.ClusterID, a.WebhookSecret, a.ImageRetention,
		a.UploadID, a.InstallCommand, a.BuildCommand, a.StartCommand, a.RootFolder, a.DistFolder, a.WatchPaths, now, now)
	if err != nil {
		return err
	}
	for _, v := range a.Volumes {
		if _, err := s.db.Exec(`INSERT INTO app_volumes(id,app_id,name,mount_path) VALUES(?,?,?,?)`,
			domain.NewID(), a.ID, v.Name, v.MountPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetApp(id string) (*domain.App, error) {
	a, err := scanApp(func(dest ...any) error {
		return s.db.QueryRow(`SELECT `+appCols+` FROM apps WHERE id=?`, id).Scan(dest...)
	})
	if err != nil {
		return nil, err
	}
	return a, s.loadAppExtras(a)
}

func (s *Store) GetProjectByOrgName(orgID, name string) (*domain.Project, error) {
	rows, err := s.db.Query(`SELECT id,org_id,name,created_at FROM projects WHERE org_id=? AND name=?`, orgID, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var p domain.Project
		var created string
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &created); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		return &p, nil
	}
	return nil, ErrNotFound
}

func (s *Store) GetAppByOrgName(orgID, name string) (*domain.App, error) {
	a, err := scanApp(func(dest ...any) error {
		return s.db.QueryRow(`SELECT `+appCols+` FROM apps WHERE org_id=? AND name=?`, orgID, name).Scan(dest...)
	})
	if err != nil {
		return nil, err
	}
	return a, s.loadAppExtras(a)
}

func (s *Store) ListAppsByProject(projectID string) ([]domain.App, error) {
	rows, err := s.db.Query(`SELECT `+appCols+` FROM apps WHERE project_id=? ORDER BY name`, projectID)
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

func (s *Store) ListApps(orgID string) ([]domain.App, error) {
	rows, err := s.db.Query(`SELECT `+appCols+` FROM apps WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	var apps []domain.App
	for rows.Next() {
		a, err := scanApp(func(dest ...any) error { return rows.Scan(dest...) })
		if err != nil {
			rows.Close()
			return nil, err
		}
		apps = append(apps, *a)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range apps {
		if err := s.loadAppExtras(&apps[i]); err != nil {
			return nil, err
		}
	}
	return apps, nil
}

func (s *Store) ListAllApps() ([]domain.App, error) {
	rows, err := s.db.Query(`SELECT ` + appCols + ` FROM apps ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var apps []domain.App
	for rows.Next() {
		a, err := scanApp(func(dest ...any) error { return rows.Scan(dest...) })
		if err != nil {
			rows.Close()
			return nil, err
		}
		apps = append(apps, *a)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range apps {
		if err := s.loadAppExtras(&apps[i]); err != nil {
			return nil, err
		}
	}
	return apps, nil
}

func (s *Store) ListAppVolumes(appID string) []string {
	rows, err := s.db.Query(`SELECT name FROM app_volumes WHERE app_id=?`, appID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			out = append(out, name)
		}
	}
	return out
}

func (s *Store) loadAppExtras(a *domain.App) error {
	vols, err := s.db.Query(`SELECT name,mount_path FROM app_volumes WHERE app_id=?`, a.ID)
	if err != nil {
		return err
	}
	defer vols.Close()
	for vols.Next() {
		var v domain.Volume
		if err := vols.Scan(&v.Name, &v.MountPath); err != nil {
			return err
		}
		a.Volumes = append(a.Volumes, v)
	}
	return vols.Err()
}

func (s *Store) UpdateApp(a *domain.App) error {
	_, err := s.db.Exec(`UPDATE apps SET name=?,image=?,git_url=?,git_branch=?,dockerfile=?,port=?,cpus=?,mem_mb=?,storage_mb=?,hc_enabled=?,hc_path=?,hc_interval_ms=?,hc_timeout_ms=?,hc_retries=?,build_type=?,preview_domain=?,server_id=?,cluster_id=?,environment_id=?,image_retention=?,upload_id=?,install_command=?,build_command=?,start_command=?,root_folder=?,dist_folder=?,watch_paths=?,updated_at=? WHERE id=?`,
		a.Name, a.Image, a.GitURL, a.GitBranch, a.Dockerfile, a.Port, a.Resources.CPUs, a.Resources.MemMB, a.Resources.StorageMB,
		boolInt(a.HealthCheck.Enabled), a.HealthCheck.Path, a.HealthCheck.IntervalMS,
		a.HealthCheck.TimeoutMS, a.HealthCheck.Retries, a.BuildType, a.PreviewDomain, a.ServerID, a.ClusterID, a.EnvironmentID,
		a.ImageRetention, a.UploadID, a.InstallCommand, a.BuildCommand, a.StartCommand, a.RootFolder, a.DistFolder, a.WatchPaths,
		time.Now().UTC().Format(time.RFC3339), a.ID)
	return err
}

func (s *Store) DeleteApp(id string) error {
	for _, t := range []string{"app_env", "app_volumes", "deployments", "domains"} {
		if _, err := s.db.Exec(`DELETE FROM `+t+` WHERE app_id=?`, id); err != nil {
			return err
		}
	}
	_, err := s.db.Exec(`DELETE FROM apps WHERE id=?`, id)
	return err
}

func (s *Store) SetEnvVar(appID, name, value string, secret bool) error {
	_, err := s.db.Exec(`INSERT INTO app_env(app_id,name,value,secret) VALUES(?,?,?,?) ON CONFLICT(app_id,name) DO UPDATE SET value=excluded.value, secret=excluded.secret`,
		appID, name, value, boolInt(secret))
	return err
}

func (s *Store) DeleteEnvVar(appID, name string) error {
	_, err := s.db.Exec(`DELETE FROM app_env WHERE app_id=? AND name=?`, appID, name)
	return err
}

func (s *Store) ListEnvVars(appID string) ([]domain.EnvVar, error) {
	rows, err := s.db.Query(`SELECT app_id,name,value,secret FROM app_env WHERE app_id=? ORDER BY name`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EnvVar
	for rows.Next() {
		var e domain.EnvVar
		var val string
		var secret int
		if err := rows.Scan(&e.AppID, &e.Name, &val, &secret); err != nil {
			return nil, err
		}
		e.Value = []byte(val)
		e.Secret = secret == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) NextDeploymentNumber(appID string) (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(number),0)+1 FROM deployments WHERE app_id=?`, appID).Scan(&n)
	return n, err
}

func (s *Store) CountDeployments(appID string) (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM deployments WHERE app_id=?`, appID).Scan(&n)
	return n, err
}

// encryptField nunca abre mão do segredo: se a criptografia falhar, o erro é
// propagado (fail-closed) — um campo secreto jamais é persistido em claro.
func encryptField(s *security.Secrets, v string) (string, error) {
	if s == nil || v == "" {
		return v, nil
	}
	return s.EncryptString(v)
}

// decryptField retorna o valor descriptografado. Se falhar, devolve o blob
// original (não há vazamento: é ciphertext), mas o erro de auth é preservado
// para diagnóstico — quem lê decide se é aceitável.
func decryptField(s *security.Secrets, v string) (string, error) {
	if s == nil || v == "" {
		return v, nil
	}
	return s.DecryptString(v)
}

func (s *Store) CreateDeployment(d *domain.Deployment) error {
	envSnap, err := encryptField(s.Secrets, d.EnvSnapshot)
	if err != nil {
		return err
	}
	composeYAML, err := encryptField(s.Secrets, d.ComposeYAML)
	if err != nil {
		return err
	}
	deploySpec, err := encryptField(s.Secrets, d.DeploySpec)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO deployments(id,app_id,number,status,commit_sha,image_ref,container_id,server_id,error,triggered_by,env_snapshot,compose_yaml,deploy_spec,compose_hash,created_at,started_at,finished_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.AppID, d.Number, string(d.Status), d.Commit, d.ImageRef, d.ContainerID, d.ServerID, d.Error, d.TriggeredBy, envSnap,
		composeYAML, deploySpec, d.ComposeHash,
		d.CreatedAt.UTC().Format(time.RFC3339), d.StartedAt.UTC().Format(time.RFC3339), "")
	return err
}

func scanDeployment(secrets *security.Secrets, scan func(dest ...any) error) (*domain.Deployment, error) {
	var d domain.Deployment
	var status, commitSHA, imageRef, containerID, serverID, errStr, triggeredBy, envSnap, composeYAML, deploySpec, composeHash, created, started, finished string
	err := scan(&d.ID, &d.AppID, &d.Number, &status, &commitSHA, &imageRef, &containerID, &serverID, &errStr, &triggeredBy, &envSnap,
		&composeYAML, &deploySpec, &composeHash, &created, &started, &finished)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.Status = domain.DeploymentStatus(status)
	d.Commit = commitSHA
	d.ImageRef = imageRef
	d.ContainerID = containerID
	d.ServerID = serverID
	d.Error = errStr
	d.TriggeredBy = triggeredBy
	if d.EnvSnapshot, err = decryptField(secrets, envSnap); err != nil {
		return nil, err
	}
	if d.ComposeYAML, err = decryptField(secrets, composeYAML); err != nil {
		return nil, err
	}
	if d.DeploySpec, err = decryptField(secrets, deploySpec); err != nil {
		return nil, err
	}
	d.ComposeHash = composeHash
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	d.StartedAt, _ = time.Parse(time.RFC3339, started)
	if finished != "" {
		d.FinishedAt, _ = time.Parse(time.RFC3339, finished)
	}
	return &d, nil
}

const depCols = `id,app_id,number,status,commit_sha,image_ref,container_id,server_id,error,triggered_by,env_snapshot,compose_yaml,deploy_spec,compose_hash,created_at,started_at,finished_at`

func (s *Store) GetDeployment(id string) (*domain.Deployment, error) {
	return scanDeployment(s.Secrets, func(dest ...any) error {
		return s.db.QueryRow(`SELECT `+depCols+` FROM deployments WHERE id=?`, id).Scan(dest...)
	})
}

func (s *Store) ListDeployments(appID string, limit int) ([]domain.Deployment, error) {
	rows, err := s.db.Query(`SELECT `+depCols+` FROM deployments WHERE app_id=? ORDER BY number DESC LIMIT ?`, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Deployment
	for rows.Next() {
		d, err := scanDeployment(s.Secrets, func(dest ...any) error { return rows.Scan(dest...) })
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *Store) LastReadyDeployment(appID string, beforeNumber int64) (*domain.Deployment, error) {
	return scanDeployment(s.Secrets, func(dest ...any) error {
		return s.db.QueryRow(`SELECT `+depCols+` FROM deployments WHERE app_id=? AND status='ready' AND number<CAST(? AS BIGINT) ORDER BY number DESC LIMIT 1`,
			appID, beforeNumber).Scan(dest...)
	})
}

func (s *Store) UpdateDeployment(d *domain.Deployment) error {
	finished := ""
	if !d.FinishedAt.IsZero() {
		finished = d.FinishedAt.UTC().Format(time.RFC3339)
	}
	composeYAML, err := encryptField(s.Secrets, d.ComposeYAML)
	if err != nil {
		return err
	}
	deploySpec, err := encryptField(s.Secrets, d.DeploySpec)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE deployments SET status=?,commit_sha=?,image_ref=?,container_id=?,server_id=?,error=?,triggered_by=?,compose_yaml=?,deploy_spec=?,compose_hash=?,started_at=?,finished_at=? WHERE id=?`,
		string(d.Status), d.Commit, d.ImageRef, d.ContainerID, d.ServerID, d.Error, d.TriggeredBy,
		composeYAML, deploySpec, d.ComposeHash,
		d.StartedAt.UTC().Format(time.RFC3339), finished, d.ID)
	return err
}

func (s *Store) CreateDomain(d *domain.Domain) error {
	_, err := s.db.Exec(`INSERT INTO domains(id,app_id,host,https,cert_status,created_at) VALUES(?,?,?,?,?,?)`,
		d.ID, d.AppID, d.Host, boolInt(d.HTTPS), d.CertStatus, d.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetDomainByHost(host string) (*domain.Domain, error) {
	var d domain.Domain
	var created, cs string
	var https int
	err := s.db.QueryRow(`SELECT id,app_id,host,https,cert_status,created_at FROM domains WHERE host=?`, host).Scan(
		&d.ID, &d.AppID, &d.Host, &https, &cs, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.HTTPS = https == 1
	d.CertStatus = cs
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &d, nil
}

func (s *Store) ListDomains(appID string) ([]domain.Domain, error) {
	rows, err := s.db.Query(`SELECT id,app_id,host,https,cert_status,created_at FROM domains WHERE app_id=?`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Domain
	for rows.Next() {
		var d domain.Domain
		var created, cs string
		var https int
		if err := rows.Scan(&d.ID, &d.AppID, &d.Host, &https, &cs, &created); err != nil {
			return nil, err
		}
		d.HTTPS = https == 1
		d.CertStatus = cs
		d.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeleteDomain(id string) error {
	_, err := s.db.Exec(`DELETE FROM domains WHERE id=?`, id)
	return err
}

func (s *Store) UpdateDomainCert(host, status string) error {
	_, err := s.db.Exec(`UPDATE domains SET cert_status=? WHERE host=?`, status, host)
	return err
}

func (s *Store) CreateBackup(b *domain.Backup) error {
	_, err := s.db.Exec(`INSERT INTO backups(id,path,size,created_at,kind,dest,app_id) VALUES(?,?,?,?,?,?,?)`,
		b.ID, b.Path, b.Size, b.CreatedAt.UTC().Format(time.RFC3339), b.Kind, b.Dest, b.AppID)
	return err
}

func (s *Store) ListBackups(limit int) ([]domain.Backup, error) {
	rows, err := s.db.Query(`SELECT id,path,size,created_at,kind,dest,app_id FROM backups ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Backup
	for rows.Next() {
		var b domain.Backup
		var created string
		if err := rows.Scan(&b.ID, &b.Path, &b.Size, &created, &b.Kind, &b.Dest, &b.AppID); err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) GetBackup(id string) (*domain.Backup, error) {
	var b domain.Backup
	var created string
	err := s.db.QueryRow(`SELECT id,path,size,created_at,kind,dest,app_id FROM backups WHERE id=?`, id).Scan(
		&b.ID, &b.Path, &b.Size, &created, &b.Kind, &b.Dest, &b.AppID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &b, nil
}

func (s *Store) DeleteBackup(id string) error {
	_, err := s.db.Exec(`DELETE FROM backups WHERE id=?`, id)
	return err
}

func (s *Store) SaveCert(host, certPath, keyPath, notAfter, provider string) error {
	_, err := s.db.Exec(`INSERT INTO certs(host,cert_path,key_path,not_after,provider,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(host) DO UPDATE SET cert_path=excluded.cert_path,key_path=excluded.key_path,not_after=excluded.not_after,provider=excluded.provider,updated_at=excluded.updated_at`,
		host, certPath, keyPath, notAfter, provider, time.Now().UTC().Format(time.RFC3339))
	return err
}

type CertRow struct {
	Host     string
	CertPath string
	KeyPath  string
	NotAfter time.Time
}

func (s *Store) ListCerts() ([]CertRow, error) {
	rows, err := s.db.Query(`SELECT host,cert_path,key_path,not_after FROM certs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CertRow
	for rows.Next() {
		var c CertRow
		var na string
		if err := rows.Scan(&c.Host, &c.CertPath, &c.KeyPath, &na); err != nil {
			return nil, err
		}
		c.NotAfter, _ = time.Parse(time.RFC3339, na)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCert(host string) (*CertRow, error) {
	var c CertRow
	var na string
	err := s.db.QueryRow(`SELECT host,cert_path,key_path,not_after FROM certs WHERE host=?`, host).Scan(
		&c.Host, &c.CertPath, &c.KeyPath, &na)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.NotAfter, _ = time.Parse(time.RFC3339, na)
	return &c, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) UpdateProject(id, name string) error {
	_, err := s.db.Exec(`UPDATE projects SET name=? WHERE id=?`, name, id)
	return err
}

func (s *Store) DeleteProject(id string) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}
