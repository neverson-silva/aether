package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"aether/internal/cert"
	"aether/internal/config"
	"aether/internal/db"
	"aether/internal/domain"
	"aether/internal/druntime"
	"aether/internal/druntime/adapter"
	rtEvents "aether/internal/druntime/events"
	"aether/internal/events"
	netx "aether/internal/net"
	"aether/internal/obs"
	"aether/internal/planner"
	"aether/internal/runtime"
	"aether/internal/security"
)

type Core struct {
	Cfg        *config.Config
	DB         *db.SQL
	Store      *db.Store
	Secrets    *security.Secrets
	Bus        *events.Bus
	Tokens     *security.TokenManager
	Driver     runtime.Driver
	Net        *netx.Engine
	Certs      *cert.Engine
	DriverInfo runtime.Info
	RT         *druntime.Runtime

	otel       *obs.OtelMetrics
	otelCancel context.CancelFunc
	otelDone   chan struct{}

	quadlet bool
	unitMgr *runtime.UnitManager

	// testMode marca containers criados durante testes (label aether.test=1)
	// para que o cleanup de teste não remova containers de produção.
	testMode bool

	proxyConfigSrv *http.Server

	liveMu sync.Mutex
	live   map[string]*obs.LiveLog

	collectorMu sync.Mutex
	collectors  map[string]*logCollector

	termMu    sync.Mutex
	terminals map[string]*terminalHost

	activeMu sync.Mutex
	active   map[string]string

	deployWkMu           sync.Mutex
	deployWorkersStarted bool
	deployCtx            context.Context
	deployCancel         context.CancelFunc
	deployWg             sync.WaitGroup

	startedAt time.Time
	agentSrv  *http.Server
	agentAddr string
	CA        *security.CA

	autopilot *autopilotState
	netQ      *netQState
	envCache  envCache

	notify *NotificationEngine
}

func New(cfg *config.Config) (*Core, error) {
	if err := cfg.EnsureDirs(); err != nil {
		_ = err
		return nil, err
	}
	sqldb, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	secrets, err := security.LoadSecrets(cfg.KeysDir)
	if err != nil {
		sqldb.Close()
		return nil, err
	}
	tokens, err := security.NewTokenManager(secrets)
	if err != nil {
		sqldb.Close()
		return nil, err
	}

	driverName := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_RUNTIME")))
	if driverName == "" || driverName == "auto" {
		driverName = "podman"
	}
	driver, err := runtime.NewDriver(driverName)
	if err != nil {
		sqldb.Close()
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	info, err := driver.Info(ctx)
	if err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("runtime driver %q indisponível: %w", driverName, err)
	}

	c := &Core{
		Cfg:        cfg,
		DB:         sqldb,
		Bus:        events.NewBus(sqldb),
		Secrets:    secrets,
		Tokens:     tokens,
		Driver:     driver,
		DriverInfo: info,
		live:       map[string]*obs.LiveLog{},
		collectors: map[string]*logCollector{},
		terminals:  map[string]*terminalHost{},
		active:     map[string]string{},
		startedAt:  time.Now(),
	}
	c.Store = db.NewStore(sqldb)
	c.Store.Secrets = secrets
	c.Net = netx.NewEngine(cfg)
	rt, rtErr := c.newDistributedRuntime(context.Background())
	if rtErr != nil {
		log.Printf("[runtime] backend indisponível (%v), usando memory", rtErr)
		rt, rtErr = adapter.New(context.Background(), druntime.Config{Backend: "memory"})
		if rtErr != nil {
			sqldb.Close()
			return nil, rtErr
		}
	}
	c.RT = rt
	if ep := os.Getenv("AETHER_OTEL_ENDPOINT"); ep != "" {
		if om, oerr := obs.NewOtelMetrics(context.Background(), ep); oerr == nil {
			c.otel = om
			log.Printf("[otel] métricas exportando para %s", ep)
		} else {
			log.Printf("[otel] falha ao iniciar exportação: %v", oerr)
		}
	}
	c.Bus.SetDistributed(func(ev events.Event) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.RT.Events.Publish(ctx, ev.Type, rtEvents.Event{
			ID:            ev.ID,
			Type:          ev.Type,
			AggregateType: ev.AggregateType,
			AggregateID:   ev.AggregateID,
			Payload:       ev.Payload,
			TS:            ev.TS,
		})
	})
	c.notify = newNotificationEngine(c.Store, c.RT.PubSub)
	c.quadlet = driverName == "podman" && os.Getenv("AETHER_QUADLET") == "1"
	if c.quadlet {
		user := os.Getenv("AETHER_APP_USER")
		if user == "" {
			user = "aether"
		}
		c.unitMgr = &runtime.UnitManager{User: user, Dir: filepath.Join(cfg.StateDir, "units", user)}
	}
	c.Certs = cert.NewEngine(cfg, secrets, c.Store)
	c.Certs.SetChallenge = c.Net.SetChallengeRoute
	c.Certs.ClearChallenge = c.Net.RemoveRoute
	c.Certs.SetLockFn(func(name string, fn func()) {
		c.withLockSkip(name, lockCertTTL, fn)
	})
	c.subscribe()
	return c, nil
}

func (c *Core) newDistributedRuntime(ctx context.Context) (*druntime.Runtime, error) {
	backend := c.Cfg.RuntimeBackend
	if backend == "" {
		if c.Cfg.RedisAddr != "" {
			backend = "redis"
		} else {
			backend = "memory"
		}
	}
	return adapter.New(ctx, druntime.Config{
		Backend:       backend,
		RedisAddr:     c.Cfg.RedisAddr,
		RedisPassword: c.Cfg.RedisPassword,
		RedisUsername: c.Cfg.RedisUsername,
		RedisDB:       c.Cfg.RedisDB,
	})
}

func (c *Core) startOtelCollector(ctx context.Context) {
	if c.otel == nil {
		return
	}
	octx, cancel := context.WithCancel(context.Background())
	c.otelCancel = cancel
	c.otelDone = make(chan struct{})
	go func() {
		defer close(c.otelDone)
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-octx.Done():
				return
			case <-t.C:
				m := c.RT.Metrics(octx)
				depth, _ := c.RT.Queue.Len(octx, deployQueueStream)
				c.otel.Record(octx, m.CacheHits, m.CacheMisses, m.CacheErrors, m.Subscribers, depth, 0)
			}
		}
	}()
}

func (c *Core) subscribe() {
	c.Bus.Subscribe(c.onDeploymentCreated)
	if c.notify != nil {
		c.notify.subscribe(c.Bus)
	}
}

func (c *Core) Start(ctx context.Context) error {
	if _, err := c.Bus.ReplayUnpublished(ctx); err != nil {
		return err
	}
	c.ensureDeployWorkers()
	if err := c.Certs.StartChallengeServer(); err != nil {
		return err
	}
	c.Certs.RenewLoop(ctx, 24*time.Hour)
	c.logsGC(ctx)
	c.StartScheduler(ctx)
	c.StartAutopilot(ctx)
	c.StartGitOpsWatch(ctx)
	c.StartNetQ(ctx)
	c.StartAlertLoop(ctx)
	c.StartSnapshotScheduler(ctx)
	c.StartImageRetentionLoop(ctx)
	c.SeedTemplates()
	c.reconcileRoutes(ctx)
	c.reconcileLogCollectors(ctx)
	c.reconcileAppStates(ctx)
	c.startOtelCollector(ctx)
	if err := c.startProxyConfigServer(); err != nil {
		return err
	}
	if err := c.Net.StartProxy(ctx); err != nil {
		return err
	}
	if err := c.StartAgentServer(); err != nil {
		return err
	}
	return nil
}

func (c *Core) startProxyConfigServer() error {
	ln, err := net.Listen("tcp", c.Cfg.ProxyEndpoint)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: c.Net.Handler()}
	c.proxyConfigSrv = srv
	go srv.Serve(ln)
	return nil
}

func (c *Core) logsGC(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Minute):
				c.liveMu.Lock()
				now := time.Now()
				var remove []string
				for id, ll := range c.live {
					info, err := os.Stat(ll.Path())
					if err != nil {
						remove = append(remove, id)
						continue
					}
					if now.Sub(info.ModTime()) > time.Hour {
						remove = append(remove, id)
					}
				}
				for _, id := range remove {
					ll := c.live[id]
					ll.Close()
					delete(c.live, id)
				}
				c.liveMu.Unlock()
			}
		}
	}()
}

func (c *Core) Stop(ctx context.Context) {
	c.stopDeployWorkers()
	c.deployWg.Wait()
	c.StopTerminalHosts()
	if c.otel != nil {
		if c.otelCancel != nil {
			c.otelCancel()
		}
		select {
		case <-c.otelDone:
		case <-time.After(2 * time.Second):
		}
		c.otel.Shutdown(ctx)
	}
	c.Certs.StopChallengeServer()
	c.Net.StopProxy()
	if c.proxyConfigSrv != nil {
		c.proxyConfigSrv.Close()
	}
	if c.agentSrv != nil {
		_ = c.agentSrv.Close()
	}
	c.liveMu.Lock()
	for _, l := range c.live {
		l.Close()
	}
	c.liveMu.Unlock()
}

func (c *Core) reconcileRoutes(ctx context.Context) {
	apps, err := c.Store.ListAllApps()
	if err != nil {
		return
	}
	for _, app := range apps {
		deploys, err := c.Store.ListDeployments(app.ID, 1)
		if err != nil || len(deploys) == 0 {
			continue
		}
		dep := &deploys[0]
		if dep.Status == domain.DeploymentReady && dep.ContainerID != "" {
			c.attachRoutes(ctx, &app, dep)
		}
	}
}

func (c *Core) attachRoutes(ctx context.Context, app *domain.App, dep *domain.Deployment) {
	port := c.containerPort(ctx, dep)
	if port == "" {
		return
	}
	domains, err := c.Store.ListDomains(app.ID)
	if err != nil {
		return
	}
	for _, d := range domains {
		target := "http://127.0.0.1:" + port
		if d.HTTPS {
			if row, err := c.Store.GetCert(d.Host); err == nil {
				c.Net.SetRouteTLS(d.Host, target, row.CertPath, row.KeyPath)
				continue
			}
		}
		c.Net.SetRoute(d.Host, target)
	}
}

func (c *Core) containerPort(ctx context.Context, dep *domain.Deployment) string {
	if dep.ContainerID == "" {
		return ""
	}
	app, err := c.Store.GetApp(dep.AppID)
	if err != nil {
		return ""
	}
	port := c.internalPort(app)
	ports, err := c.Driver.Ports(ctx, dep.ContainerID)
	if err != nil {
		return ""
	}
	hostPort := ports[port]
	if hostPort == "" {
		for _, p := range ports {
			if p != "" && !strings.HasPrefix(p, "[") {
				hostPort = p
			}
		}
		if hostPort == "" {
			for _, p := range ports {
				if p != "" {
					hostPort = p
				}
			}
		}
	}
	// "0.0.0.0:8082" → "8082" | "[::]:8082" → "8082"
	hostPort = parseHostPort(hostPort)
	if hostPort == "" {
		hostPort = parseHostPort(ports[port])
	}
	return hostPort
}

func parseHostPort(v string) string {
	if i := strings.LastIndex(v, ":"); i >= 0 {
		v = v[i+1:]
	}
	return strings.TrimSuffix(v, "]")
}

// SetTestMode ativa o modo teste (label aether.test=1 nos containers), usado
// pelo cleanup de testes para remover apenas containers criados durante testes.
func (c *Core) SetTestMode() { c.testMode = true }

var ErrMFARequired = errors.New("mfa_required")

// internalPort retorna a porta em que o processo escuta DENTRO do container.
// A porta configurada (app.Port) é a porta do HOST (exposição pública);
// para web estático/SPA o nginx escuta na 80.
func (c *Core) internalPort(app *domain.App) string {
	if plan, err := c.Store.GetDeploymentPlan(app.ID); err == nil && plan.WebServer == "nginx" {
		return "80"
	}
	return strconv.Itoa(app.Port)
}

func (c *Core) Login(email, password, code string) (*domain.User, string, error) {
	user, err := c.Store.GetUserByEmail(email)
	if err != nil {
		return nil, "", errors.New("credenciais inválidas")
	}
	var hasher security.PasswordHasher
	ok, err := hasher.Verify(password, user.PasswordHash)
	if err != nil || !ok {
		return nil, "", errors.New("credenciais inválidas")
	}
	_, enabled, err := c.Store.GetTOTP(user.ID)
	if err != nil {
		return nil, "", err
	}
	if enabled {
		if code == "" {
			return nil, "", ErrMFARequired
		}
		valid, err := c.VerifyTOTP(user.ID, code)
		if err != nil || !valid {
			return nil, "", errors.New("código MFA inválido")
		}
	}
	orgID, role, err := c.defaultOrg(user.ID, user.Name)
	if err != nil {
		return nil, "", err
	}
	token, err := c.Tokens.Sign(security.Claims{
		Subject:    user.ID,
		OrgID:      orgID,
		Role:       string(role),
		GlobalRole: user.GlobalRole,
	}, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// defaultOrg returns the user's primary organization, creating a personal
// organization ("<Name>'s Organization") on first login — mirroring GitHub.
func (c *Core) defaultOrg(userID, userName string) (string, domain.Role, error) {
	rows, err := c.DB.Query(`SELECT org_id, role FROM members WHERE user_id=? LIMIT 1`, userID)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return c.createPersonalOrg(userID, userName)
	}
	var orgID, role string
	if err := rows.Scan(&orgID, &role); err != nil {
		return "", "", err
	}
	return orgID, domain.Role(role), nil
}

func (c *Core) createPersonalOrg(userID, userName string) (string, domain.Role, error) {
	name := userName + "'s Organization"
	if strings.TrimSpace(userName) == "" {
		name = "Personal Organization"
	}
	org := &domain.Org{ID: domain.NewID(), Slug: c.uniqueSlug(name), Name: name, OwnerUserID: userID, CreatedAt: time.Now().UTC()}
	if err := c.Store.CreateOrg(org); err != nil {
		return "", "", err
	}
	if err := c.Store.CreateMember(&domain.Member{OrgID: org.ID, UserID: userID, Role: domain.RoleOwner}); err != nil {
		return "", "", err
	}
	return org.ID, domain.RoleOwner, nil
}

func (c *Core) uniqueSlug(name string) string {
	base := slugify(name)
	if base == "" {
		base = "org"
	}
	probe := base
	for i := 1; ; i++ {
		if _, err := c.Store.GetOrgBySlug(probe); err != nil {
			return probe
		}
		probe = base + "-" + strconv.Itoa(i)
	}
}

func slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	out := strings.ToLower(re.ReplaceAllString(strings.TrimSpace(s), "-"))
	return strings.Trim(out, "-")
}

func (c *Core) VerifyToken(token string) (*security.Claims, error) {
	return c.Tokens.Verify(token)
}

func (c *Core) VerifyAPIKey(token string) (*security.Claims, error) {
	if !strings.HasPrefix(token, "ak_") {
		return nil, errors.New("chave inválida")
	}
	key, err := c.Store.GetApiKeyByHash(security.HashAPIKey(token))
	if err != nil {
		return nil, errors.New("chave inválida")
	}
	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		return nil, errors.New("chave expirada")
	}
	member, err := c.Store.GetMember(key.OrgID, key.UserID)
	if err != nil {
		return nil, errors.New("usuário não pertence à organização")
	}
	c.Store.TouchApiKey(key.ID)
	return &security.Claims{Subject: key.UserID, OrgID: key.OrgID, Role: string(member.Role)}, nil
}

func (c *Core) CreateMemberUser(orgID, email, name, password, role string) (*domain.User, error) {
	user, err := c.Store.GetUserByEmail(email)
	if err != nil {
		var hasher security.PasswordHasher
		hash, herr := hasher.Hash(password)
		if herr != nil {
			return nil, herr
		}
		user = &domain.User{ID: domain.NewID(), Email: email, Name: name, PasswordHash: hash, CreatedAt: time.Now().UTC()}
		if cErr := c.Store.CreateUser(user); cErr != nil {
			return nil, cErr
		}
	}
	if mErr := c.Store.CreateMember(&domain.Member{OrgID: orgID, UserID: user.ID, Role: domain.Role(role)}); mErr != nil {
		return nil, mErr
	}
	return user, nil
}

func (c *Core) CreateUserAndOrg(email, name, password string) (*domain.User, *domain.Org, error) {
	var hasher security.PasswordHasher
	hash, err := hasher.Hash(password)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	user := &domain.User{ID: domain.NewID(), Email: email, Name: name, PasswordHash: hash, CreatedAt: now}
	if err := c.Store.CreateUser(user); err != nil {
		return nil, nil, err
	}
	org := &domain.Org{ID: domain.NewID(), Name: "Default", OwnerUserID: user.ID, CreatedAt: now}
	if err := c.Store.CreateOrg(org); err != nil {
		return nil, nil, err
	}
	if err := c.Store.CreateMember(&domain.Member{OrgID: org.ID, UserID: user.ID, Role: domain.RoleOwner}); err != nil {
		return nil, nil, err
	}
	return user, org, nil
}

func (c *Core) CreateApp(orgID string, a *domain.App) error {
	project, err := c.Store.GetProject(a.ProjectID)
	if err != nil {
		return err
	}
	if project.OrgID != orgID {
		return errors.New("projeto não pertence à organização")
	}
	if a.Dockerfile == "" {
		a.Dockerfile = "Dockerfile"
	}
	if a.HealthCheck.IntervalMS == 0 {
		a.HealthCheck.IntervalMS = 5000
	}
	if a.HealthCheck.TimeoutMS == 0 {
		a.HealthCheck.TimeoutMS = 2000
	}
	if a.HealthCheck.Retries == 0 {
		a.HealthCheck.Retries = 3
	}
	if a.Port == 0 {
		a.Port = 80
	}
	if a.EnvironmentID == "" {
		if def, err := c.DefaultEnvironment(a.ProjectID); err == nil {
			a.EnvironmentID = def.ID
		}
	}
	a.OrgID = orgID
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	return c.Store.CreateApp(a)
}

func (c *Core) Audit(orgID, userID, action, resourceType, resourceID, details string) {
	c.Store.CreateAudit(&domain.AuditLog{
		ID: domain.NewID(), OrgID: orgID, UserID: userID, Action: action,
		ResourceType: resourceType, ResourceID: resourceID, Details: details,
		CreatedAt: time.Now().UTC(),
	})
}

func (c *Core) CreateProject(orgID, name string) (*domain.Project, error) {
	now := time.Now().UTC()
	project := &domain.Project{ID: domain.NewID(), OrgID: orgID, Name: name, Slug: slugify(name), CreatedAt: now, UpdatedAt: now}
	env := &domain.Environment{
		ID:        "env-" + domain.NewID(),
		ProjectID: project.ID,
		Name:      "production",
		Slug:      "production",
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := c.Store.CreateProjectWithEnv(project, env); err != nil {
		return nil, err
	}
	return project, nil
}

func (c *Core) CreateAPIKey(orgID, userID, name string, scopes []string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	key := "ak_" + base64.RawURLEncoding.EncodeToString(raw)
	k := &domain.ApiKey{
		ID:        domain.NewID(),
		OrgID:     orgID,
		UserID:    userID,
		Name:      name,
		KeyHash:   security.HashAPIKey(key),
		Scopes:    scopes,
		CreatedAt: time.Now().UTC(),
	}
	if err := c.Store.CreateApiKey(k); err != nil {
		return "", err
	}
	return key, nil
}

func (c *Core) EnsureAppEnv(appID string) ([]string, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	merged := map[string]string{}
	addVar := func(name, value string) { merged[name] = value }

	lookup := map[string]string{}
	for _, v := range c.cachedProjectVars(app.ProjectID) {
		lookup[v.Key] = c.decryptProjectVar(&v)
	}
	if app.EnvironmentID != "" {
		for _, v := range c.cachedEnvVars(app.EnvironmentID) {
			lookup[v.Key] = c.decryptVar(&v)
		}
	}

	vars, err := c.Store.ListEnvVars(appID)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		val := string(v.Value)
		if v.Secret {
			dec, err := c.Secrets.DecryptString(val)
			if err != nil {
				return nil, fmt.Errorf("env %s: %w", v.Name, err)
			}
			val = dec
		}
		lookup[v.Name] = val
	}
	for k, v := range lookup {
		addVar(k, v)
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+merged[k])
	}
	return ResolveInterpolations(out, lookup), nil
}

func (c *Core) SetAppEnv(appID, name, value string, secret bool) error {
	if secret {
		enc, err := c.Secrets.EncryptString(value)
		if err != nil {
			return err
		}
		value = enc
	}
	return c.Store.SetEnvVar(appID, name, value, secret)
}

func (c *Core) CreateDomain(appID, host string, https bool) error {
	d := &domain.Domain{
		ID:         domain.NewID(),
		AppID:      appID,
		Host:       host,
		HTTPS:      https,
		CertStatus: "pending",
		CreatedAt:  time.Now().UTC(),
	}
	if err := c.Store.CreateDomain(d); err != nil {
		return err
	}
	if https {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := c.Certs.Ensure(ctx, host); err != nil {
				log.Printf("[cert] emissão de %s falhou: %v", host, err)
				c.Store.UpdateDomainCert(host, "failed")
				return
			}
			c.Store.UpdateDomainCert(host, "issued")
			c.reapplyDomainRoute(d)
		}()
	}
	return nil
}

func (c *Core) reapplyDomainRoute(d *domain.Domain) {
	deploys, err := c.Store.ListDeployments(d.AppID, 1)
	if err != nil || len(deploys) == 0 {
		return
	}
	dep := &deploys[0]
	if dep.Status != domain.DeploymentReady {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	port := c.containerPort(ctx, dep)
	if port == "" {
		return
	}
	target := "http://127.0.0.1:" + port
	if row, err := c.Store.GetCert(d.Host); err == nil {
		c.Net.SetRouteTLS(d.Host, target, row.CertPath, row.KeyPath)
	} else {
		c.Net.SetRoute(d.Host, target)
	}
}

func (c *Core) BackupCreate() (*domain.Backup, error) {
	dir := filepath.Join(c.Cfg.StateDir, "backups")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	name := "state-" + time.Now().UTC().Format("20060102T150405") + ".dump"
	path := filepath.Join(dir, name)
	if err := c.pgDump(path); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	b := &domain.Backup{
		ID:        domain.NewID(),
		Path:      path,
		Size:      info.Size(),
		CreatedAt: time.Now().UTC(),
		Kind:      "state",
		Dest:      "local",
	}
	if err := c.Store.CreateBackup(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Core) createBackupRecord(b *domain.Backup, kind, dest, appID string) error {
	b.Kind = kind
	b.Dest = dest
	b.AppID = appID
	return c.Store.CreateBackup(b)
}

func (c *Core) RestoreBackup(id string) error {
	b, err := c.Store.GetBackup(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(b.Path); err != nil {
		return err
	}
	if err := c.StopActiveContainers(); err != nil {
		return err
	}
	c.DB.Close()
	if err := c.pgRestore(b.Path); err != nil {
		return err
	}
	return nil
}

func (c *Core) pgDump(path string) error {
	cmd := exec.CommandContext(context.Background(), "pg_dump",
		"--no-owner", "--no-privileges", "-Fc",
		"-h", c.Cfg.DatabaseHost, "-p", strconv.Itoa(c.Cfg.DatabasePort),
		"-U", c.Cfg.DatabaseUser, "-d", c.Cfg.DatabaseName, "-f", path)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+c.Cfg.DatabasePassword)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Core) pgRestore(path string) error {
	cmd := exec.CommandContext(context.Background(), "pg_restore",
		"--clean", "--if-exists", "--no-owner", "--no-privileges",
		"-h", c.Cfg.DatabaseHost, "-p", strconv.Itoa(c.Cfg.DatabasePort),
		"-U", c.Cfg.DatabaseUser, "-d", c.Cfg.DatabaseName, path)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+c.Cfg.DatabasePassword)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Core) StopActiveContainers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	apps, err := c.Store.ListAllApps()
	if err != nil {
		return nil
	}
	for _, app := range apps {
		deploys, err := c.Store.ListDeployments(app.ID, 1)
		if err != nil || len(deploys) == 0 {
			continue
		}
		dep := &deploys[0]
		if dep.ContainerID != "" && (dep.Status == domain.DeploymentReady || dep.Status == domain.DeploymentStarting) {
			c.Driver.Remove(ctx, dep.ContainerID, true)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o640)
}

func (c *Core) buildSource(ctx context.Context, app *domain.App, srcDir, imageRef string, ll *obs.LiveLog) error {
	df := filepath.Join(srcDir, app.Dockerfile)
	if app.Dockerfile == "" {
		app.Dockerfile = "Dockerfile"
		df = filepath.Join(srcDir, "Dockerfile")
	}
	// Frontend deployment plan: use generated nginx.conf + Dockerfile
	if plan, err := c.Store.GetDeploymentPlan(app.ID); err == nil && plan.Dockerfile != "" {
		ll.Write([]byte("[plan] aplicando plano de deploy: " + plan.Framework + " (" + plan.AppType + ")\n"))
		// Sempre regenera nginx.conf + Dockerfile com o gerador atual para
		// que a porta configurada e o fallback SPA sejam respeitados
		// (plano persistido pode estar desatualizado de versões anteriores).
		np := &planner.Plan{
			Framework:      plan.Framework,
			Library:        plan.Library,
			PackageManager: plan.PackageManager,
			Runtime:        plan.Runtime,
			BuildCommand:   plan.BuildCommand,
			InstallCommand: plan.InstallCommand,
			OutputDir:      plan.OutputDir,
			AppType:        planner.AppType(plan.AppType),
			WebServer:      plan.WebServer,
			ContainerPort:  app.Port,
			SPAFallback:    plan.SPAFallback || plan.AppType == "spa" || plan.AppType == "static",
			IndexFile:      plan.IndexFile,
			HasLockfile:    plan.PackageManager == "pnpm" || plan.PackageManager == "yarn" || plan.PackageManager == "bun" || hasFile(srcDir, "package-lock.json"),
		}
		conf := planner.GenerateNginxConf(np)
		if err := os.WriteFile(filepath.Join(srcDir, "nginx.conf"), []byte(conf), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(df, []byte(planner.GenerateDockerfile(np)), 0o644); err != nil {
			return err
		}
		app.Dockerfile = "Dockerfile"
		app.BuildType = "dockerfile"
		df = filepath.Join(srcDir, "Dockerfile")
	}
	switch app.BuildType {
	case "", "dockerfile":
		ll.Write([]byte("[build] dockerfile=" + app.Dockerfile + "\n"))
		_, err := c.Driver.BuildWithWriter(ctx, srcDir, df, imageRef, ll)
		return err
	case "nixpacks":
		if _, err := exec.LookPath("nixpacks"); err != nil {
			return fmt.Errorf("nixpacks não encontrado no PATH (instale em https://nixpacks.com)")
		}
		ll.Write([]byte("[build] nixpacks plan -> dockerfile\n"))
		cmd := exec.CommandContext(ctx, "nixpacks", "plan", srcDir, "--out", df)
		cmd.Dir = srcDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nixpacks plan: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		_, err = c.Driver.BuildWithWriter(ctx, srcDir, df, imageRef, ll)
		return err
	case "custom":
		if app.StartCommand == "" && app.BuildCommand == "" {
			return fmt.Errorf("build_type custom requer start command")
		}
		ll.Write([]byte("[build] custom dockerfile gerado\n"))
		df = filepath.Join(srcDir, "Dockerfile")
		dfContent := "FROM node:22-alpine AS builder\n"
		if app.InstallCommand != "" {
			dfContent += "RUN " + app.InstallCommand + "\n"
		}
		if app.BuildCommand != "" {
			dfContent += "RUN " + app.BuildCommand + "\n"
		}
		dfContent += "FROM node:22-alpine\n"
		dfContent += "WORKDIR /app\n"
		dfContent += "COPY --from=builder /app /app\n"
		if app.StartCommand != "" {
			dfContent += "CMD [" + strconv.Quote(app.StartCommand) + "]\n"
		} else {
			dfContent += "CMD [\"node\", \"index.js\"]\n"
		}
		if err := os.WriteFile(df, []byte(dfContent), 0o644); err != nil {
			return err
		}
		_, err := c.Driver.BuildWithWriter(ctx, srcDir, df, imageRef, ll)
		return err
	case "buildpacks":
		if _, err := exec.LookPath("pack"); err != nil {
			return fmt.Errorf("pack CLI não encontrado (instale em https://buildpacks.io)")
		}
		ll.Write([]byte("[build] pack build (buildpacks)\n"))
		cmd := exec.CommandContext(ctx, "pack", "build", imageRef,
			"--builder", "paketobuildpacks/builder-jammy-base", "--path", srcDir)
		cmd.Dir = srcDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pack build: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("build_type desconhecido: %s", app.BuildType)
	}
}
