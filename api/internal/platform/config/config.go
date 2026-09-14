package config

import (
	"fmt"
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
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	GoogleOAuthRedirectURI  string
	PublicURL               string
	CORSOrigins             []string
	DevMode                 bool
	CertEmail               string
	ACMEDirectory           string
	FreeDomainProvider      string
	FreeDomainBase          string
	IngressNetwork          string
	PublishedNetwork        string
	TraefikImage            string
	MetricsPath             string
	DockerHost              string
	BuildDockerHost         string

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
	RestoreMaxUploadBytes int64
	RestoreQuotaBytes     int64
	BackupMaxBytes        int64

	CnbBuilder         string
	BuildDockerNetwork string

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
	corsOrigins := splitCSV(envOr("AETHER_CORS_ORIGINS", ""))
	for _, origin := range corsOrigins {
		if origin == "*" {
			return nil, fmt.Errorf("AETHER_CORS_ORIGINS cannot contain wildcard origins")
		}
	}
	freeDomainBase := envOr("AETHER_FREE_DOMAIN_BASE", "")
	if devMode {
		if publicURL == "" {
			publicURL = "http://localhost:8080"
		}
		if freeDomainBase == "" {
			freeDomainBase = "localhost"
		}
	}
	if len(corsOrigins) == 0 && publicURL != "" {
		corsOrigins = []string{publicURL}
	}
	if ingressNetwork := envOr("AETHER_INGRESS_NETWORK", "aether-ingress"); ingressNetwork == envOr("AETHER_PUBLISHED_NETWORK", "aether-workload-host") {
		return nil, fmt.Errorf("AETHER_INGRESS_NETWORK and AETHER_PUBLISHED_NETWORK must be different")
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
		PublicURL:               publicURL,
		CORSOrigins:             corsOrigins,
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
		PublishedNetwork:         envOr("AETHER_PUBLISHED_NETWORK", "aether-workload-host"),
		TraefikImage:             envOr("AETHER_TRAEFIK_IMAGE", "docker.io/library/traefik:v3.2@sha256:e561a37f8710d9cf41c78bdf421d822b2c0b48267ec0552e644565fb55466ea9"),
		DockerHost:               dockerHost(),
		BuildDockerHost:          buildDockerHost(),
		RuntimeBackend:           envOr("AETHER_RUNTIME_BACKEND", "nats"),
		NATSURL:                  envOr("AETHER_NATS_URL", "nats://127.0.0.1:4222"),
		NATSName:                 envOr("AETHER_NATS_NAME", "aether-api"),
		NATSUser:                 os.Getenv("AETHER_NATS_USER"),
		NATSPassword:             os.Getenv("AETHER_NATS_PASSWORD"),
		CnbBuilder:               envOr("AETHER_CNB_BUILDER", "127.0.0.1:1500/builder:node-spa"),
		BuildDockerNetwork:       envOr("AETHER_BUILD_DOCKER_NETWORK", ""),
		GitHubAppID:              int64(envInt("AETHER_GITHUB_APP_ID", 0)),
		GitHubAppSlug:            os.Getenv("AETHER_GITHUB_APP_SLUG"),
		GitHubPrivateKey:         os.Getenv("AETHER_GITHUB_PRIVATE_KEY"),
		GitHubWebhookSecret:      os.Getenv("AETHER_GITHUB_WEBHOOK_SECRET"),
		GitHubAPIURL:             envOr("AETHER_GITHUB_API_URL", "https://api.github.com"),
		StudioCacheTTLSeconds:    envInt("AETHER_STUDIO_CACHE_TTL", 300),
		RestoreMaxUploadBytes:    int64(envInt("AETHER_RESTORE_MAX_UPLOAD_BYTES", 0)),
		RestoreQuotaBytes:        int64(envInt("AETHER_RESTORE_QUOTA_BYTES", 0)),
		BackupMaxBytes:           int64(envInt("AETHER_BACKUP_MAX_BYTES", 0)),
		CookieSecure:             envBool("AETHER_COOKIE_SECURE", true),
	}
	if err := validateInfrastructureImages(cfg); err != nil {
		return nil, err
	}
	if err := validateImageSignaturePolicy(cfg); err != nil {
		return nil, err
	}
	if cfg.ACMEDirectory == "" {
		cfg.ACMEDirectory = "https://acme-v02.api.letsencrypt.org/directory"
	}
	return cfg, nil
}

func validateImageSignaturePolicy(cfg *Config) error {
	if cfg.DevMode {
		return nil
	}
	if !envBool("AETHER_REQUIRE_IMAGE_SIGNATURES", false) {
		return fmt.Errorf("AETHER_REQUIRE_IMAGE_SIGNATURES must be true outside development mode")
	}
	if !envBool("AETHER_REQUIRE_IMAGE_DIGESTS", false) {
		return fmt.Errorf("AETHER_REQUIRE_IMAGE_DIGESTS must be true outside development mode")
	}
	key := strings.TrimSpace(os.Getenv("AETHER_COSIGN_PUBLIC_KEY"))
	if key == "" {
		return fmt.Errorf("AETHER_COSIGN_PUBLIC_KEY must point to a verification key outside development mode")
	}
	if _, err := os.Stat(key); err != nil {
		return fmt.Errorf("AETHER_COSIGN_PUBLIC_KEY is unavailable: %w", err)
	}
	return nil
}

func validateInfrastructureImages(cfg *Config) error {
	if cfg.DevMode {
		return nil
	}
	for name, image := range map[string]string{"AETHER_TRAEFIK_IMAGE": cfg.TraefikImage, "AETHER_CNB_BUILDER": cfg.CnbBuilder} {
		if !strings.Contains(image, "@sha256:") {
			return fmt.Errorf("%s must use a sha256 digest outside development mode", name)
		}
	}
	return nil
}

func dockerHost() string {
	if value := os.Getenv("AETHER_DOCKER_HOST"); value != "" {
		return value
	}
	return os.Getenv("DOCKER_HOST")
}

func buildDockerHost() string {
	if value := os.Getenv("AETHER_BUILD_DOCKER_HOST"); value != "" {
		return value
	}
	if value := os.Getenv("AETHER_IMAGE_DOCKER_HOST"); value != "" {
		return value
	}
	return "unix:///var/run/docker.sock"
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimRight(strings.TrimSpace(part), "/"); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
