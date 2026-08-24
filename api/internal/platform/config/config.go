package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	StateDir                string
	DataDir                 string
	CertsDir                string
	LogsDir                 string
	BuildsDir               string
	UploadsDir              string
	CacheDir                string
	KeysDir                 string
	AlertIntervalSeconds    int
	ImageRetention          int
	APIAddr                 string
	WorkerHealthAddr        string
	MonitoringHealthAddr    string
	ProxyEndpoint           string
	ChallengeAddr           string
	AgentAddr               string
	TraefikBin              string
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	GoogleOAuthRedirectURI  string
	PublicURL               string
	DevMode                 bool
	CertEmail               string
	ACMEDirectory           string
	FreeDomainProvider      string
	FreeDomainBase          string
	IngressNetwork          string
	TraefikImage            string
	MetricsPath             string

	DatabaseURL              string
	DatabaseHost             string
	DatabasePort             int
	DatabaseName             string
	DatabaseUser             string
	DatabasePassword         string
	DatabaseSSLMode          string
	DatabaseSchema           string
	DatabasePoolMin          int
	DatabasePoolMax          int
	DatabaseConnectTimeout   int
	DatabaseIdleTimeout      int
	DatabaseStatementTimeout int
	DatabaseQueryTimeout     int
	DatabaseApplicationName  string
	DatabaseLogging          bool
	DatabaseMigrateOnStart   bool
	DatabaseSeedOnFirstStart bool
	DatabaseRetryAttempts    int
	DatabaseRetryDelay       int

	RuntimeBackend string
	NATSURL        string
	NATSName       string
	NATSUser       string
	NATSPassword   string

	StudioCacheTTLSeconds int

	CnbBuilder string

	GitHubAppID         int64
	GitHubAppSlug       string
	GitHubPrivateKey    string
	GitHubWebhookSecret string
	GitHubAPIURL        string

	CookieSecure bool
}

func DefaultStateDir() string {
	if v := os.Getenv("AETHER_STATE"); v != "" {
		return v
	}
	if os.Geteuid() == 0 {
		return "/var/lib/aether"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".aether"
	}
	return filepath.Join(home, ".aether")
}

func Load() (*Config, error) {
	state := DefaultStateDir()
	devMode := envBool("DEV_MODE", false) || envBool("AETHER_DEV_MODE", false)
	publicURL := envOr("AETHER_PUBLIC_URL", "")
	freeDomainBase := envOr("AETHER_FREE_DOMAIN_BASE", "")
	if devMode {
		if publicURL == "" || !isLocalURL(publicURL) {
			publicURL = "http://localhost:8080"
		}
		if freeDomainBase == "" {
			freeDomainBase = "localhost"
		}
	}
	cfg := &Config{
		StateDir:                state,
		DataDir:                 filepath.Join(state, "data"),
		CertsDir:                filepath.Join(state, "certs"),
		LogsDir:                 filepath.Join(state, "logs"),
		BuildsDir:               filepath.Join(state, "builds"),
		UploadsDir:              filepath.Join(state, "builds", "uploads"),
		CacheDir:                filepath.Join(state, "cache"),
		KeysDir:                 filepath.Join(state, "keys"),
		APIAddr:                 envOr("AETHER_API_ADDR", "127.0.0.1:8080"),
		WorkerHealthAddr:        envOr("AETHER_WORKER_HEALTH_ADDR", "127.0.0.1:8081"),
		MonitoringHealthAddr:    envOr("AETHER_MONITORING_HEALTH_ADDR", "127.0.0.1:8082"),
		ProxyEndpoint:           envOr("AETHER_PROXY_ENDPOINT", "127.0.0.1:15090"),
		ChallengeAddr:           envOr("AETHER_CHALLENGE_ADDR", "127.0.0.1:15001"),
		AgentAddr:               envOr("AETHER_AGENT_ADDR", "127.0.0.1:9443"),
		TraefikBin:              envOr("AETHER_TRAEFIK_BIN", ""),
		PublicURL:               publicURL,
		DevMode:                 devMode,
		GoogleOAuthClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		GoogleOAuthClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		GoogleOAuthRedirectURI:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URI"),

		DatabaseURL:              os.Getenv("DATABASE_URL"),
		DatabaseHost:             envOr("DATABASE_HOST", "127.0.0.1"),
		DatabasePort:             envInt("DATABASE_PORT", 5432),
		DatabaseName:             envOr("DATABASE_NAME", "aether"),
		DatabaseUser:             envOr("DATABASE_USER", "postgres"),
		DatabasePassword:         os.Getenv("DATABASE_PASSWORD"),
		DatabaseSSLMode:          envOr("DATABASE_SSL_MODE", "prefer"),
		DatabaseSchema:           envOr("DATABASE_SCHEMA", "public"),
		DatabasePoolMin:          envInt("DATABASE_POOL_MIN", 2),
		DatabasePoolMax:          envInt("DATABASE_POOL_MAX", 20),
		DatabaseConnectTimeout:   envInt("DATABASE_CONNECTION_TIMEOUT", 10),
		DatabaseIdleTimeout:      envInt("DATABASE_IDLE_TIMEOUT", 300),
		DatabaseStatementTimeout: envInt("DATABASE_STATEMENT_TIMEOUT", 0),
		DatabaseQueryTimeout:     envInt("DATABASE_QUERY_TIMEOUT", 0),
		DatabaseApplicationName:  envOr("DATABASE_APPLICATION_NAME", "aether"),
		DatabaseLogging:          envBool("DATABASE_LOGGING", false),
		DatabaseMigrateOnStart:   envBool("DATABASE_MIGRATE_ON_START", true),
		DatabaseSeedOnFirstStart: envBool("DATABASE_SEED_ON_FIRST_START", true),
		DatabaseRetryAttempts:    envInt("DATABASE_RETRY_ATTEMPTS", 10),
		DatabaseRetryDelay:       envInt("DATABASE_RETRY_DELAY", 2),
		ImageRetention:           envInt("AETHER_IMAGE_RETENTION", 5),
		CertEmail:                envOr("AETHER_CERT_EMAIL", ""),
		ACMEDirectory:            envOr("AETHER_ACME_DIR", ""),
		FreeDomainProvider:       envOr("AETHER_FREE_DOMAIN_PROVIDER", "nip.io"),
		FreeDomainBase:           freeDomainBase,
		IngressNetwork:           envOr("AETHER_INGRESS_NETWORK", "aether-ingress"),
		TraefikImage:             envOr("AETHER_TRAEFIK_IMAGE", "docker.io/library/traefik:v3.2"),
		RuntimeBackend:           envOr("AETHER_RUNTIME_BACKEND", "nats"),
		NATSURL:                  envOr("AETHER_NATS_URL", "nats://127.0.0.1:4222"),
		NATSName:                 envOr("AETHER_NATS_NAME", "aether-api"),
		NATSUser:                 os.Getenv("AETHER_NATS_USER"),
		NATSPassword:             os.Getenv("AETHER_NATS_PASSWORD"),
		CnbBuilder:               envOr("AETHER_CNB_BUILDER", "127.0.0.1:5000/builder:node-spa"),
		GitHubAppID:              int64(envInt("AETHER_GITHUB_APP_ID", 0)),
		GitHubAppSlug:            os.Getenv("AETHER_GITHUB_APP_SLUG"),
		GitHubPrivateKey:         os.Getenv("AETHER_GITHUB_PRIVATE_KEY"),
		GitHubWebhookSecret:      os.Getenv("AETHER_GITHUB_WEBHOOK_SECRET"),
		GitHubAPIURL:             envOr("AETHER_GITHUB_API_URL", "https://api.github.com"),
		StudioCacheTTLSeconds:    envInt("AETHER_STUDIO_CACHE_TTL", 300),
		CookieSecure:             envBool("AETHER_COOKIE_SECURE", false),
	}
	if cfg.ACMEDirectory == "" {
		cfg.ACMEDirectory = "https://acme-v02.api.letsencrypt.org/directory"
	}
	return cfg, nil
}

func isLocalURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (c *Config) EnsureDirs() error {
	for _, d := range []string{
		c.StateDir, c.DataDir, c.CertsDir, c.LogsDir,
		c.BuildsDir, c.CacheDir, c.KeysDir,
		c.UploadsDir,
		filepath.Join(c.LogsDir, "apps"),
		filepath.Join(c.BuildsDir, "sources"),
		filepath.Join(c.StateDir, "traefik"),
	} {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0o750); err != nil {
			return err
		}
	}
	return nil
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "1" || v == "true" || v == "TRUE" || v == "yes"
	}
	return def
}
