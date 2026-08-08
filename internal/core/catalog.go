package core

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime/compose"
)

type catalogEntry struct {
	id, name, desc, category, icon string
	tags                           []string
	image                          string
	port                           int
	readme                         string
	homepage, github, license      string
	featured, editorsChoice        bool
}

func e(id, name, desc, category, icon string, tags []string, image string, port int, featured bool) catalogEntry {
	return catalogEntry{id: id, name: name, desc: desc, category: category, icon: icon, tags: tags, image: image, port: port, license: "MIT", featured: featured, editorsChoice: false}
}

func def(entries ...catalogEntry) []catalogEntry { return entries }

var catalog = def(
	// Databases
	e("postgresql", "PostgreSQL", "Advanced relational database", "database", "postgresql", []string{"sql", "relational", "postgres"}, "postgres:16", 5432, true),
	e("mysql", "MySQL", "Popular open-source SQL database", "database", "mysql", []string{"sql", "relational"}, "mysql:8.4", 3306, true),
	e("mariadb", "MariaDB", "MySQL-compatible SQL database", "database", "mariadb", []string{"sql", "relational"}, "mariadb:11", 3306, false),
	e("sqlserver", "SQL Server", "Microsoft relational database", "database", "sqlserver", []string{"sql", "microsoft"}, "mcr.microsoft.com/mssql/server:2022-latest", 1433, false),
	e("oracle", "Oracle Free", "Oracle Database 23c Free", "database", "oracle", []string{"sql", "oracle"}, "gvenzl/oracle-free:23-slim", 1521, false),
	e("mongodb", "MongoDB", "Document-oriented NoSQL database", "database", "mongodb", []string{"nosql", "document"}, "mongo:7", 27017, true),
	e("redis", "Redis", "In-memory data structure store", "cache", "redis", []string{"cache", "queue"}, "redis:7", 6379, true),
	e("valkey", "Valkey", "Redis-compatible in-memory store", "cache", "valkey", []string{"cache"}, "valkey/valkey:8", 6379, false),
	e("rabbitmq", "RabbitMQ", "Message broker with AMQP", "queue", "rabbitmq", []string{"queue", "amqp"}, "rabbitmq:4-management", 5672, false),
	e("kafka", "Apache Kafka", "Distributed event streaming", "streaming", "kafka", []string{"streaming", "events"}, "apache/kafka:3.8", 9092, false),
	e("clickhouse", "ClickHouse", "Columnar analytics database", "analytics", "clickhouse", []string{"analytics", "columnar"}, "clickhouse/clickhouse-server:24", 8123, false),
	e("qdrant", "Qdrant", "Vector similarity search engine", "vector", "qdrant", []string{"vector", "ai"}, "qdrant/qdrant:v1.10", 6333, false),
	e("meilisearch", "Meilisearch", "Fast and relevant search engine", "search", "meilisearch", []string{"search", "fulltext"}, "getmeili/meilisearch:v1.9", 7700, false),
	e("typesense", "Typesense", "Typo-tolerant search engine", "search", "typesense", []string{"search"}, "typesense/typesense:27", 8108, false),
	e("elasticsearch", "Elasticsearch", "Distributed search and analytics", "search", "elasticsearch", []string{"search", "logs"}, "docker.elastic.co/elasticsearch/elasticsearch:8.15", 9200, false),
	e("minio", "MinIO", "S3-compatible object storage", "storage", "minio", []string{"object-storage", "s3"}, "minio/minio:latest", 9000, true),
	e("influxdb", "InfluxDB", "Time series database", "timeseries", "influxdb", []string{"time-series", "metrics"}, "influxdb:2.7", 8086, false),
	e("mongo-express", "Mongo Express", "Web UI for MongoDB", "tools", "webtop", []string{"admin", "database"}, "mongo-express:latest", 8081, false),
	// CMS
	e("wordpress", "WordPress", "The world's most popular CMS", "cms", "wordpress", []string{"cms", "blog"}, "wordpress:php8.3", 80, true),
	e("ghost", "Ghost", "Modern publishing platform", "cms", "ghost", []string{"cms", "blog"}, "ghost:5", 2368, false),
	e("strapi", "Strapi", "Headless CMS for developers", "cms", "strapi", []string{"cms", "headless", "node"}, "strapi:latest", 1337, false),
	e("directus", "Directus", "Headless CMS with data engine", "cms", "directus", []string{"cms", "headless"}, "directus/directus:11", 8055, false),
	// Monitoring / Observability
	e("grafana", "Grafana", "Observability and data visualization", "monitoring", "grafana", []string{"metrics", "dashboards"}, "grafana/grafana:11", 3000, true),
	e("prometheus", "Prometheus", "Metrics collection and alerting", "monitoring", "prometheus", []string{"metrics"}, "prom/prometheus:v2.53", 9090, true), // editors
	e("uptime-kuma", "Uptime Kuma", "Self-hosted uptime monitoring", "monitoring", "uptime-kuma", []string{"uptime", "status"}, "louislam/uptime-kuma:1", 3001, false),
	e("netdata", "Netdata", "Real-time performance monitoring", "monitoring", "netdata", []string{"metrics", "realtime"}, "netdata/netdata:latest", 19999, false),
	e("loki", "Grafana Loki", "Log aggregation system", "logging", "loki", []string{"logs"}, "grafana/loki:3", 3100, false),
	e("vector", "Vector", "High-performance observability pipeline", "logging", "vector", []string{"logs", "metrics"}, "timberio/vector:0.40-alpine", 8686, false),
	// Analytics
	e("metabase", "Metabase", "Business intelligence and analytics", "analytics", "metabase", []string{"bi", "dashboards"}, "metabase/metabase:v0.50", 3000, false),
	e("superset", "Apache Superset", "Data exploration and visualization", "analytics", "superset", []string{"bi"}, "apache/superset:4", 8088, false),
	e("umami", "Umami", "Privacy-friendly web analytics", "analytics", "umami", []string{"analytics", "privacy"}, "ghcr.io/umami-software/umami:postgresql-latest", 3000, false),
	e("plausible", "Plausible", "Simple, privacy-first analytics", "analytics", "plausible", []string{"analytics"}, "plausible/analytics:v2", 8000, false),
	// AI / LLM
	e("ollama", "Ollama", "Run LLMs locally", "ai", "ollama", []string{"llm", "inference"}, "ollama/ollama:latest", 11434, true),
	e("open-webui", "Open WebUI", "User-friendly LLM web interface", "ai", "open-webui", []string{"llm", "chat"}, "ghcr.io/open-webui/open-webui:main", 3000, false),
	e("flowise", "Flowise", "Drag & drop LLM workflows", "ai", "flowise", []string{"llm", "rag"}, "flowiseai/flowise:latest", 3000, false),
	e("anything-llm", "AnythingLLM", "All-in-one AI assistant workspace", "ai", "anything-llm", []string{"llm", "rag"}, "mintplexlabs/anythingllm:latest", 3001, false),
	e("langflow", "Langflow", "Visual framework for AI agents", "ai", "langflow", []string{"llm", "agents"}, "langflowai/langflow:latest", 7860, false),
	// Developer tools
	e("gitea", "Gitea", "Lightweight self-hosted Git service", "devtools", "gitea", []string{"git", "forge"}, "gitea/gitea:1.22", 3000, true),
	e("code-server", "code-server", "VS Code in the browser", "devtools", "code-server", []string{"ide", "code"}, "lscr.io/linuxserver/code-server:latest", 8443, false),
	e("jenkins", "Jenkins", "Automation server for CI/CD", "devtools", "jenkins", []string{"ci", "cd"}, "jenkins/jenkins:lts-jdk17", 8080, false),
	e("jupyterlab", "JupyterLab", "Interactive notebooks for data science", "devtools", "jupyterlab", []string{"notebooks", "data"}, "jupyter/base-notebook:latest", 8888, false),
	e("gitlab", "GitLab CE", "Complete DevOps platform", "devtools", "gitlab", []string{"git", "ci", "devops"}, "gitlab/gitlab-ce:17", 80, false),
	// Identity / Security
	e("authentik", "Authentik", "Identity provider and SSO", "security", "authentik", []string{"identity", "sso"}, "ghcr.io/goauthentik/server:2024.8", 9000, false),
	e("keycloak", "Keycloak", "Open-source identity and access management", "security", "keycloak", []string{"identity", "oidc"}, "quay.io/keycloak/keycloak:25", 8080, true),
	e("vault", "HashiCorp Vault", "Secrets management and encryption", "security", "vault", []string{"secrets"}, "hashicorp/vault:1.17", 8200, false),
	e("authelia", "Authelia", "Authentication and authorization server", "security", "adguard", []string{"2fa", "auth"}, "authelia/authelia:4.38", 9091, false),
	// Media
	e("jellyfin", "Jellyfin", "Free media server", "media", "jellyfin", []string{"media", "streaming"}, "lscr.io/linuxserver/jellyfin:latest", 8096, false),
	e("immich", "Immich", "Self-hosted photo management", "media", "immich", []string{"photos"}, "ghcr.io/immich-app/immich-server:release", 2283, false),
	// Automation
	e("n8n", "n8n", "Workflow automation platform", "automation", "n8n", []string{"workflows", "automation"}, "n8nio/n8n:latest", 5678, true),
	e("nodered", "Node-RED", "Flow-based programming for IoT", "automation", "nodered", []string{"iot", "flows"}, "nodered/node-red:4", 1880, false),
	e("airflow", "Apache Airflow", "Orchestrate complex workflows", "automation", "airflow", []string{"pipelines", "scheduling"}, "apache/airflow:2.9", 8080, false),
	// Networking / Reverse proxy
	e("caddy", "Caddy", "Automatic HTTPS reverse proxy", "networking", "caddy", []string{"proxy", "tls"}, "caddy:2", 80, true),
	e("nginx-proxy", "Nginx", "High-performance web server", "networking", "nginx-proxy", []string{"proxy", "web-server"}, "nginx:alpine", 80, false),
	e("wg-easy", "WireGuard Easy", "Simple WireGuard VPN", "networking", "wg-easy", []string{"vpn"}, "ghcr.io/wg-easy/wg-easy:14", 51821, false),
	e("adguard", "AdGuard Home", "Network-wide ad blocker and DNS", "networking", "adguard", []string{"dns", "ads"}, "adguard/adguardhome:latest", 3000, false),
	// Email
	e("mailpit", "Mailpit", "Email testing tool", "email", "mailpit", []string{"smtp", "testing"}, "axllent/mailpit:v1", 8025, false),
	e("listmonk", "Listmonk", "Self-hosted newsletter manager", "email", "listmonk", []string{"newsletter"}, "listmonk/listmonk:latest", 9000, false),
	// Productivity
	e("nextcloud", "Nextcloud", "Self-hosted productivity suite", "productivity", "nextcloud", []string{"files", "cloud"}, "nextcloud:30", 80, true),
	e("paperless", "Paperless-ngx", "Document management system", "productivity", "paperless", []string{"documents", "dms"}, "ghcr.io/paperless-ngx/paperless-ngx:latest", 8000, false),
	e("searxng", "SearXNG", "Privacy-respecting metasearch engine", "productivity", "searxng", []string{"search", "privacy"}, "searxng/searxng:latest", 8080, false),
	e("filebrowser", "File Browser", "Web file manager", "productivity", "filebrowser", []string{"files"}, "filebrowser/filebrowser:latest", 80, false),
	// Wiki
	e("wikijs", "Wiki.js", "Powerful and extensible wiki", "wiki", "wikijs", []string{"wiki", "docs"}, "ghcr.io/requarks/wiki:2", 3000, false),
	e("outline", "Outline", "Team knowledge base", "wiki", "outline", []string{"wiki", "docs"}, "docker.getoutline.com/outlinewiki/outline:latest", 3000, false),
	e("bookstack", "BookStack", "Documentation and knowledge wiki", "wiki", "bookstack", []string{"wiki", "docs"}, "linuxserver/bookstack:latest", 80, false),
	// Home lab
	e("home-assistant", "Home Assistant", "Home automation platform", "homelab", "home-assistant", []string{"iot", "home"}, "ghcr.io/home-assistant/home-assistant:stable", 8123, false),
	e("portainer", "Portainer", "Container management UI", "homelab", "portainer", []string{"docker", "admin"}, "portainer/portainer-ce:latest", 9000, false),
	// Messaging
	e("mattermost", "Mattermost", "Open-source team messaging", "messaging", "mattermost", []string{"chat", "teams"}, "mattermost/mattermost-team-edition:10", 8065, false),
	e("rocket-chat", "Rocket.Chat", "Open-source communication platform", "messaging", "rocket-chat", []string{"chat"}, "rocket.chat:6", 3000, false),
	// ERP/CRM
	e("odoo", "Odoo", "Business management suite", "erp", "odoo", []string{"erp", "crm"}, "odoo:17", 8069, false),
	e("suitecrm", "SuiteCRM", "Open-source CRM", "erp", "suitecrm", []string{"crm"}, "bitnami/suitecrm:latest", 8080, false),
	e("monica", "Monica", "Personal relationship manager", "erp", "monica", []string{"crm"}, "monica/monicahq:latest", 8080, false),
	// Static / frameworks
	e("nginx-static", "Static Site", "Serve static HTML via Nginx", "web", "paperless", []string{"static", "html"}, "nginx:alpine", 80, false),
	e("node", "Node.js", "Node.js runtime container", "web", "node", []string{"node", "runtime"}, "node:22-alpine", 3000, false),
	e("python", "Python", "Python runtime container", "web", "python", []string{"python", "runtime"}, "python:3.12-slim", 8000, false),
	e("go", "Go", "Go runtime container", "web", "go", []string{"go", "runtime"}, "golang:1.22-alpine", 8080, false),
	e("postgres-admin", "PgAdmin", "PostgreSQL admin web interface", "database", "postgres-admin", []string{"admin", "sql"}, "dpage/pgadmin4:8", 80, false),
	e("phpmyadmin", "phpMyAdmin", "MySQL administration tool", "database", "phpmyadmin", []string{"admin", "sql"}, "phpmyadmin:5", 80, false),
	e("adminer", "Adminer", "Single-file database admin", "database", "adminer", []string{"admin", "sql"}, "adminer:4", 8080, false),
	e("dokuwiki", "DokuWiki", "Simple file-based wiki", "wiki", "dokuwiki", []string{"wiki"}, "lscr.io/linuxserver/dokuwiki:latest", 80, false),
	e("matrix", "Synapse", "Matrix homeserver", "messaging", "matrix", []string{"chat", "federation"}, "matrixdotorg/synapse:latest", 8008, false),
	e("element", "Element Web", "Matrix web client", "messaging", "element", []string{"chat"}, "vectorim/element-web:latest", 80, false),
	e("proxmox-dashboard", "Proxmox Helper", "Home lab proxmox utilities", "homelab", "proxmox", []string{"homelab"}, "ghcr.io/community-scripts/proxmox-ve:latest", 8006, false),
	e("tailscale", "Tailscale", "Zero-config VPN mesh", "networking", "tailscale", []string{"vpn", "mesh"}, "tailscale/tailscale:latest", 8080, false),
	e("cloudflared", "Cloudflared", "Cloudflare Tunnel daemon", "networking", "cloudflared", []string{"tunnel"}, "cloudflare/cloudflared:latest", 0, false),
	e("postfix", "Postfix", "SMTP mail transfer agent", "email", "postfix", []string{"smtp"}, "boky/postfix:latest", 25, false),
	e("dovecot", "Dovecot", "IMAP/POP3 mail server", "email", "dovecot", []string{"imap"}, "dovecot/dovecot:latest", 143, false),
	e("grasshopper", "Grasshopper", "Self-hosted financial tracker", "finance", "grasshopper", []string{"finance"}, "ghcr.io/grasshoppercodes/grasshopper:latest", 3000, false),
	e("firefly", "Firefly III", "Personal finance manager", "finance", "firefly", []string{"finance"}, "fireflyiii/core:latest", 8080, false),
	e("calcom", "Cal.com", "Open-source scheduling platform", "productivity", "calcom", []string{"calendar"}, "calcom/cal.com:latest", 3000, false),
	e("linkwarden", "Linkwarden", "Self-hosted bookmark manager", "productivity", "linkwarden", []string{"bookmarks"}, "ghcr.io/linkwarden/linkwarden:latest", 3000, false),
	e("stirling-pdf", "Stirling PDF", "PDF manipulation toolkit", "productivity", "stirling-pdf", []string{"pdf", "documents"}, "frooodle/s-pdf:latest", 8080, false),
	e("trilium", "Trilium Notes", "Hierarchical note-taking", "productivity", "trilium", []string{"notes"}, "zadam/trilium:latest", 8080, false),
	e("webtop", "Webtop", "Full desktop in the browser", "homelab", "webtop", []string{"desktop"}, "lscr.io/linuxserver/webtop:latest", 3000, false),
	e("codex", "OpenCodex", "Self-hosted code intelligence", "devtools", "codex", []string{"code"}, "ghcr.io/opencodex/codex:latest", 8080, false),
	e("grafana-oncall", "Grafana OnCall", "Incident management", "monitoring", "grafana-oncall", []string{"incidents"}, "grafana/oncall:latest", 8080, false),
	e("bazarr", "Bazarr", "Subtitle management for media", "media", "bazarr", []string{"subtitles", "media"}, "lscr.io/linuxserver/bazarr:latest", 6767, false),
	e("navidrome", "Navidrome", "Self-hosted music streaming", "media", "navidrome", []string{"music"}, "deluan/navidrome:latest", 4533, false),
	e("forgejo", "Forgejo", "Lightweight Git forge", "devtools", "forgejo", []string{"git", "forge"}, "codeberg.org/forgejo/forgejo:8", 3000, false),
	e("librechat", "LibreChat", "Multi-model AI chat UI", "ai", "librechat", []string{"llm", "chat"}, "ghcr.io/danny-avila/librechat:latest", 3080, false),
	e("supabase", "Supabase", "Open-source Firebase alternative", "database", "supabase", []string{"postgres", "auth", "realtime"}, "supabase/supabase:latest", 8000, false),
)

var editorsChoiceIDs = map[string]bool{
	"tpl-prometheus": true, "tpl-grafana": true, "tpl-minio": true,
	"tpl-n8n": true, "tpl-wordpress": true, "tpl-ollama": true,
}

// templateVersions: 3 últimas versões pinadas por template (imagens fixadas, nunca :latest).
var templateVersions = map[string][]string{
	"postgresql":        {"16.4", "16.3", "16.2"},
	"mysql":             {"8.4.3", "8.4.2", "8.0.40"},
	"mariadb":           {"11.6.2", "11.4.4", "10.11.10"},
	"sqlserver":         {"2022-CU16", "2022-CU15", "2022-CU14"},
	"oracle":            {"23.4-slim", "23.3-slim", "23.2-slim"},
	"mongodb":           {"7.0.16", "7.0.15", "6.0.20"},
	"redis":             {"7.4.2", "7.4.1", "7.2.7"},
	"valkey":            {"8.0.2", "8.0.1", "8.0.0"},
	"rabbitmq":          {"4.0.4", "4.0.3", "3.13.7"},
	"kafka":             {"3.9.0", "3.8.1", "3.8.0"},
	"clickhouse":        {"24.9", "24.8", "24.3"},
	"qdrant":            {"v1.12.4", "v1.12.3", "v1.12.2"},
	"meilisearch":       {"v1.11.3", "v1.11.2", "v1.11.1"},
	"typesense":         {"27.1", "27.0", "26.0"},
	"elasticsearch":     {"8.15.3", "8.15.2", "8.15.1"},
	"minio":             {"RELEASE.2024-12-18T13-15-44Z", "RELEASE.2024-12-13T22-19-12Z", "RELEASE.2024-11-21T17-21-54Z"},
	"influxdb":          {"2.7.11", "2.7.10", "2.7.9"},
	"mongo-express":     {"1.0.2-20", "1.0.2-18", "1.0.2-15"},
	"wordpress":         {"6.7.1-php8.3", "6.7-php8.3", "6.6.2-php8.2"},
	"ghost":             {"5.101.1", "5.100.3", "5.99.2"},
	"strapi":            {"4.25.0", "4.24.4", "4.24.3"},
	"directus":          {"11.3.1", "11.2.1", "11.1.1"},
	"grafana":           {"11.3.0", "11.2.2", "11.1.7"},
	"prometheus":        {"v2.55.1", "v2.54.1", "v2.53.2"},
	"uptime-kuma":       {"1.23.16", "1.23.15", "1.23.13"},
	"netdata":           {"v1.47.4", "v1.47.3", "v1.47.2"},
	"loki":              {"3.3.2", "3.3.1", "3.2.1"},
	"vector":            {"0.42.0", "0.41.1", "0.41.0"},
	"grafana-oncall":    {"v1.10.12", "v1.10.11", "v1.10.10"},
	"metabase":          {"v0.50.23", "v0.50.21", "v0.50.19"},
	"superset":          {"4.1.1", "4.1.0", "4.0.2"},
	"umami":             {"v2.13.2", "v2.13.1", "v2.12.4"},
	"plausible":         {"v2.1.4", "v2.1.3", "v2.1.2"},
	"ollama":            {"0.5.1", "0.5.0", "0.4.7"},
	"open-webui":        {"v0.3.45", "v0.3.44", "v0.3.43"},
	"flowise":           {"2.1.5", "2.1.4", "2.1.3"},
	"anything-llm":      {"1.0.1", "1.0.0", "0.9.9"},
	"langflow":          {"1.2.4", "1.2.3", "1.2.2"},
	"librechat":         {"v0.7.5", "v0.7.4", "v0.7.3"},
	"gitea":             {"1.22.4", "1.22.3", "1.22.1"},
	"forgejo":           {"8.0.2", "8.0.1", "8.0.0"},
	"code-server":       {"4.96.2", "4.95.3", "4.94.0"},
	"jenkins":           {"2.492.2-lts-jdk17", "2.484.1-lts-jdk17", "2.479.1-lts-jdk17"},
	"jupyterlab":        {"2024-11-12", "2024-10-15", "2024-09-26"},
	"gitlab":            {"17.6.1-ce.0", "17.5.3-ce.0", "17.4.4-ce.0"},
	"authentik":         {"2024.12.1", "2024.10.2", "2024.8.3"},
	"keycloak":          {"26.0.5", "26.0.4", "25.0.6"},
	"vault":             {"1.18.2", "1.18.1", "1.17.6"},
	"authelia":          {"4.38.16", "4.38.15", "4.38.14"},
	"jellyfin":          {"10.10.3", "10.10.2", "10.9.11"},
	"immich":            {"v1.118.0", "v1.117.0", "v1.116.0"},
	"bazarr":            {"1.4.5", "1.4.4", "1.4.3"},
	"navidrome":         {"0.53.2", "0.53.1", "0.53.0"},
	"n8n":               {"1.73.0", "1.72.3", "1.72.1"},
	"nodered":           {"4.0.4", "4.0.3", "3.1.10"},
	"airflow":           {"2.10.3", "2.10.2", "2.9.3"},
	"caddy":             {"2.9.1", "2.9.0", "2.8.4"},
	"nginx-proxy":       {"1.27.3", "1.27.2", "1.27.1"},
	"nginx-static":      {"1.27.3", "1.27.2", "1.27.1"},
	"wg-easy":           {"14", "13", "12"},
	"adguard":           {"v0.107.54", "v0.107.53", "v0.107.52"},
	"tailscale":         {"v1.78.0", "v1.77.0", "v1.76.0"},
	"cloudflared":       {"2024.12.1", "2024.11.2", "2024.11.1"},
	"mailpit":           {"v1.21.5", "v1.21.4", "v1.21.3"},
	"listmonk":          {"v3.0.0", "v2.6.0", "v2.5.0"},
	"postfix":           {"3.9.0", "3.8.6", "3.8.5"},
	"dovecot":           {"2.3.21.1", "2.3.21", "2.3.20.1"},
	"nextcloud":         {"30.0.4", "30.0.2", "29.0.9"},
	"paperless":         {"2.13.5", "2.13.4", "2.13.3"},
	"searxng":           {"2024.11.26", "2024.11.15", "2024.11.06"},
	"filebrowser":       {"v2.31.1", "v2.31.0", "v2.30.1"},
	"trilium":           {"0.63.7", "0.63.6", "0.63.5"},
	"linkwarden":        {"v2.5.7", "v2.5.6", "v2.5.5"},
	"stirling-pdf":      {"0.36.6", "0.36.5", "0.36.3"},
	"webtop":            {"4.19.0", "4.18.0", "4.17.0"},
	"wikijs":            {"2.5.304", "2.5.303", "2.5.302"},
	"outline":           {"0.79.2", "0.79.1", "0.79.0"},
	"bookstack":         {"24.10.2", "24.10", "24.05.2"},
	"dokuwiki":          {"2024-02-06a", "2023-04-04a", "2023-04-04"},
	"home-assistant":    {"2024.12.3", "2024.12.2", "2024.11.3"},
	"portainer":         {"2.21.2", "2.21.1", "2.20.3"},
	"proxmox-dashboard": {"1.3.1", "1.3.0", "1.2.0"},
	"mattermost":        {"10.2.1", "10.1.1", "9.11.5"},
	"rocket-chat":       {"6.13.0", "6.12.1", "6.11.2"},
	"matrix":            {"v1.118.0", "v1.117.0", "v1.116.0"},
	"element":           {"v1.11.87", "v1.11.86", "v1.11.85"},
	"odoo":              {"17.0", "16.0", "15.0"},
	"suitecrm":          {"8.5.0", "8.4.0", "8.3.0"},
	"monica":            {"v4.1.2", "v4.1.1", "v4.0.0"},
	"firefly":           {"6.2.2", "6.2.1", "6.1.13"},
	"grasshopper":       {"0.2.3", "0.2.2", "0.2.1"},
	"node":              {"22.22.3-alpine", "22.21.0-alpine", "22.20.0-alpine", "20.18.1-alpine"},
	"python":            {"3.12.8-slim", "3.12.7-slim", "3.12.6-slim"},
	"go":                {"1.23.4-alpine", "1.23.3-alpine", "1.23.2-alpine"},
	"calcom":            {"v4.0.0", "v3.6.1", "v3.6.0"},
	"postgres-admin":    {"8.14", "8.13", "8.12"},
	"phpmyadmin":        {"5.2.1", "5.2.0", "5.1.3"},
	"adminer":           {"4.8.1", "4.8.0", "4.7.9"},
	"supabase":          {"v1.18.0", "v1.17.0", "v1.16.0"},
	"codex":             {"latest", "v1", "v0.9"},
}

// templateEnvDefaults: variáveis de ambiente default que cada template precisa no primeiro deploy.
// Valores com {{password}} são gerados aleatoriamente pelo frontend no wizard.
var templateEnvDefaults = map[string][]string{
	"postgresql":     {"POSTGRES_DB=app", "POSTGRES_USER=app", "POSTGRES_PASSWORD={{password}}"},
	"mysql":          {"MYSQL_DATABASE=app", "MYSQL_USER=app", "MYSQL_PASSWORD={{password}}", "MYSQL_ROOT_PASSWORD={{password}}"},
	"mariadb":        {"MARIADB_DATABASE=app", "MARIADB_USER=app", "MARIADB_PASSWORD={{password}}", "MARIADB_ROOT_PASSWORD={{password}}"},
	"mongodb":        {"MONGO_INITDB_ROOT_USERNAME=admin", "MONGO_INITDB_ROOT_PASSWORD={{password}}"},
	"redis":          {"REDIS_PASSWORD={{password}}"},
	"rabbitmq":       {"RABBITMQ_DEFAULT_USER=admin", "RABBITMQ_DEFAULT_PASS={{password}}"},
	"wordpress":      {"WORDPRESS_DB_HOST=db", "WORDPRESS_DB_USER=app", "WORDPRESS_DB_PASSWORD={{password}}", "WORDPRESS_DB_NAME=app"},
	"ghost":          {"url=http://localhost:2368"},
	"strapi":         {"APP_KEYS={{random64}}", "API_TOKEN_SALT={{random64}}", "ADMIN_JWT_SECRET={{random64}}", "JWT_SECRET={{random64}}"},
	"directus":       {"ADMIN_EMAIL=admin@example.com", "ADMIN_PASSWORD={{password}}", "SECRET={{random64}}", "KEY={{random64}}"},
	"grafana":        {"GF_SECURITY_ADMIN_USER=admin", "GF_SECURITY_ADMIN_PASSWORD={{password}}"},
	"metabase":       {"MB_DB_TYPE=postgres", "MB_DB_DBNAME=metabase", "MB_DB_PORT=5432"},
	"umami":          {"DATABASE_TYPE=postgresql", "HASH_SALT={{random64}}"},
	"nextcloud":      {"NEXTCLOUD_ADMIN_USER=admin", "NEXTCLOUD_ADMIN_PASSWORD={{password}}", "SQLITE_DATABASE=nextcloud.db"},
	"keycloak":       {"KEYCLOAK_ADMIN=admin", "KEYCLOAK_ADMIN_PASSWORD={{password}}"},
	"authentik":      {"AUTHENTIK_SECRET_KEY={{random64}}", "AUTHENTIK_BOOTSTRAP_PASSWORD={{password}}"},
	"vault":          {"VAULT_DEV_ROOT_TOKEN_ID=root"},
	"n8n":            {"N8N_BASIC_AUTH_ACTIVE=true", "N8N_BASIC_AUTH_USER=admin", "N8N_BASIC_AUTH_PASSWORD={{password}}"},
	"odoo":           {"POSTGRES_USER=odoo", "POSTGRES_PASSWORD={{password}}", "POSTGRES_DB=postgres"},
	"minio":          {"MINIO_ROOT_USER=minioadmin", "MINIO_ROOT_PASSWORD={{password}}"},
	"paperless":      {"PAPERLESS_SECRET_KEY={{random64}}", "PAPERLESS_ADMIN_USER=admin", "PAPERLESS_ADMIN_PASSWORD={{password}}"},
	"calcom":         {"NEXTAUTH_SECRET={{random64}}", "CALENDSO_ENCRYPTION_KEY={{random64}}"},
	"immich":         {"DB_USERNAME=postgres", "DB_PASSWORD={{password}}", "DB_DATABASE_NAME=immich"},
	"librechat":      {"ALLOW_REGISTRATION=false"},
	"open-webui":     {"WEBUI_SECRET_KEY={{random64}}"},
	"flowise":        {"FLOWISE_USERNAME=admin", "FLOWISE_PASSWORD={{password}}"},
	"langflow":       {"LANGFLOW_AUTO_LOGIN=true"},
	"anything-llm":   {"STORAGE_DIR=/app/server/storage"},
	"ollama":         {"OLLAMA_KEEP_ALIVE=24h"},
	"jellyfin":       {"TZ=UTC"},
	"home-assistant": {"TZ=UTC"},
	"adguard":        {"TZ=UTC"},
	"tailscale":      {"TS_AUTHKEY=tskey-auth-xxx"},
	"mattermost":     {"TZ=UTC"},
	"gitlab":         {"GITLAB_ROOT_PASSWORD={{password}}", "GITLAB_SHARED_RUNNERS_REGISTRATION_TOKEN={{random64}}"},
	"jenkins":        {"JENKINS_OPTS=--httpPort=8080"},
	"supabase":       {"POSTGRES_PASSWORD={{password}}", "JWT_SECRET={{random64}}", "ANON_KEY={{random64}}"},
	"gitea":          {"GITEA__security__INSTALL_LOCK=true"},
}

func (c *Core) buildCatalog() []domain.Template {
	now := time.Now().UTC()
	var out []domain.Template
	for _, entry := range catalog {
		base, tag := splitImageTag(entry.image)
		vers := templateVersions[entry.id]
		if len(vers) == 0 {
			vers = []string{tag}
		}
		def := `{"services":[{"name":"app","image":"` + base + `","port":` + itoa(entry.port) + `,"versions":[`
		for i, v := range vers {
			if i > 0 {
				def += `,`
			}
			def += strconv.Quote(v)
		}
		def += `]`
		if envs := templateEnvDefaults[entry.id]; len(envs) > 0 {
			def += `,"env":{`
			for i, e := range envs {
				parts := strings.SplitN(e, "=", 2)
				if i > 0 {
					def += `,`
				}
				def += strconv.Quote(parts[0]) + `:` + strconv.Quote(parts[1])
			}
			def += `}`
		}
		def += `}]}`
		t := domain.Template{
			ID:            "tpl-" + entry.id,
			Name:          entry.name,
			Description:   entry.desc,
			Category:      entry.category,
			Tags:          entry.tags,
			Icon:          entry.icon,
			Version:       "1",
			Definition:    def,
			Readme:        entry.readme,
			Homepage:      entry.homepage,
			GitHub:        entry.github,
			License:       entry.license,
			Featured:      entry.featured,
			EditorsChoice: editorsChoiceIDs["tpl-"+entry.id],
			Verified:      true,
			UpdatedAt:     now,
		}
		out = append(out, t)
	}
	return out
}

func splitImageTag(image string) (string, string) {
	if i := strings.LastIndex(image, ":"); i > 0 && !strings.Contains(image[i:], "/") {
		return image[:i], image[i+1:]
	}
	return image, "latest"
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func (c *Core) SeedTemplates() error {
	valid := map[string]bool{}
	for _, t := range c.buildCatalog() {
		valid[t.ID] = true
		var existing domain.Template
		err := c.DB.QueryRow(`SELECT id FROM templates WHERE id=?`, t.ID).Scan(&existing.ID)
		if err == nil {
			_, _ = c.DB.Exec(`UPDATE templates SET editors_choice=?, featured=?, verified=?, icon=?, definition=? WHERE id=?`,
				boolInt(t.EditorsChoice), boolInt(t.Featured), boolInt(t.Verified), t.Icon, t.Definition, t.ID)
			continue
		}
		if err := c.Store.CreateTemplate(&t); err != nil {
			return err
		}
	}
	rows, err := c.DB.Query(`SELECT id FROM templates`)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && !valid[id] {
			stale = append(stale, id)
		}
	}
	rows.Close()
	for _, id := range stale {
		_, _ = c.DB.Exec(`DELETE FROM templates WHERE id=?`, id)
	}
	return nil
}

type TemplateFilter struct {
	Category      string
	Search        string
	Verified      *bool
	Featured      *bool
	EditorsChoice bool
}

func (c *Core) ListTemplatesFiltered(f TemplateFilter) ([]domain.Template, error) {
	all, err := c.Store.ListTemplates()
	if err != nil {
		return nil, err
	}
	var out []domain.Template
	for _, t := range all {
		if f.Category != "" && t.Category != f.Category {
			continue
		}
		if f.Featured != nil && t.Featured != *f.Featured {
			continue
		}
		if f.EditorsChoice && !t.EditorsChoice {
			continue
		}
		if f.Verified != nil && t.Verified != *f.Verified {
			continue
		}
		if f.Search != "" {
			hay := strings.ToLower(t.Name + " " + t.Description + " " + strings.Join(t.Tags, " "))
			if !strings.Contains(hay, strings.ToLower(f.Search)) {
				continue
			}
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Core) TrendingTemplates(limit int) ([]domain.Template, error) {
	all, err := c.Store.ListTemplates()
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Installs > all[j].Installs })
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (c *Core) TemplateCategories() []string {
	seen := map[string]bool{}
	var out []string
	for _, entry := range catalog {
		if !seen[entry.category] {
			seen[entry.category] = true
			out = append(out, entry.category)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Core) InstallTemplate(templateID, orgID, projectID, name string, overrides map[string]string) (*domain.Template, error) {
	t, err := c.Store.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}
	yml := templateCompose(t, overrides)
	composeName := name
	if composeName == "" {
		composeName = t.Name
	}
	if _, err := c.SaveCompose(orgID, projectID, composeName, yml); err != nil {
		return nil, err
	}
	if err := c.Store.IncrementTemplateInstalls(templateID); err != nil {
		return nil, err
	}
	t.ComposeYAML = yml
	return t, nil
}

func templateCompose(t *domain.Template, overrides map[string]string) string {
	var def struct {
		Services []struct {
			Name     string            `json:"name"`
			Image    string            `json:"image"`
			Port     int               `json:"port"`
			Env      map[string]string `json:"env"`
			Volumes  []string          `json:"volumes"`
			Versions []string          `json:"versions"`
		} `json:"services"`
	}
	if err := json.Unmarshal([]byte(t.Definition), &def); err != nil {
		def.Services = nil
	}
	// Deployment Spec First: cada serviço do template vira um DeploymentSpec
	// tipado e o compose é gerado pelo generator (fonte de verdade = spec).
	var specs []*compose.DeploymentSpec
	for _, svc := range def.Services {
		if svc.Name == "" {
			svc.Name = "app"
		}
		image := svc.Image
		if ov, ok := overrides["image"]; ok && ov != "" {
			image = ov
		} else if len(svc.Versions) > 0 {
			image = svc.Image + ":" + svc.Versions[0]
		}
		spec := &compose.DeploymentSpec{
			Service: compose.ServiceName(svc.Name),
			Image:   image,
			Restart: "unless-stopped",
		}
		if svc.Port > 0 {
			spec.Ports = []compose.PortMapping{{Host: strconv.Itoa(svc.Port), Container: strconv.Itoa(svc.Port)}}
		}
		if len(svc.Env) > 0 {
			spec.Environment = map[string]string{}
			for k, v := range svc.Env {
				val := v
				if strings.Contains(v, "{{password}}") {
					val = "change-me"
				}
				if strings.Contains(v, "{{random}}") {
					val = randomPassword()
				}
				spec.Environment[k] = val
			}
		}
		for _, v := range svc.Volumes {
			spec.Volumes = append(spec.Volumes, compose.VolumeSpec{Source: svc.Name + "-data", Target: v})
		}
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		return "services:\n"
	}
	yml, err := compose.GenerateMulti(specs)
	if err != nil {
		return "services:\n"
	}
	return yml
}

func (c *Core) ListTemplates() ([]domain.Template, error) {
	return c.Store.ListTemplates()
}

func (c *Core) GetTemplate(id string) (*domain.Template, error) {
	return c.Store.GetTemplate(id)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
