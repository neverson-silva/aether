package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"aether/internal/security"

	"aether/internal/agent"
	"aether/internal/api"
	"aether/internal/client"
	"aether/internal/config"
	"aether/internal/core"
	"aether/internal/installer"
)

type Command struct {
	Name  string
	Usage string
	Run   func(args []string) error
}

func Commands() []Command {
	return []Command{
		{Name: "install", Usage: "aether install [--email E] [--password P] [--name N]", Run: cmdInstall},
		{Name: "serve", Usage: "aether serve [--addr HOST:PORT]", Run: cmdServe},
		{Name: "login", Usage: "aether login --email E --password P [--server URL]", Run: cmdLogin},
		{Name: "whoami", Usage: "aether whoami", Run: cmdWhoami},
		{Name: "members", Usage: "aether members list", Run: cmdMembers},
		{Name: "api-keys", Usage: "aether api-keys create --name N [--scope S]", Run: cmdApiKeys},
		{Name: "projects", Usage: "aether projects create NAME | aether projects list", Run: cmdProjects},
		{Name: "apps", Usage: "aether apps list | create | deploy | logs | rollback | env | domains | rm", Run: cmdApps},
		{Name: "deploy", Usage: "aether deploy APP --project P", Run: cmdDeploy},
		{Name: "backups", Usage: "aether backups create | list", Run: cmdBackups},
		{Name: "status", Usage: "aether status", Run: cmdStatus},
		{Name: "databases", Usage: "aether databases create|list|rm|backup", Run: cmdDatabases},
		{Name: "cron", Usage: "aether cron create|list|rm --app", Run: cmdCron},
		{Name: "workers", Usage: "aether workers create|list|start|stop|rm --app", Run: cmdWorkers},
		{Name: "compose", Usage: "aether compose create|list|up|down|rm", Run: cmdCompose},
		{Name: "previews", Usage: "aether previews create|list|rm --app --branch", Run: cmdPreviews},
		{Name: "templates", Usage: "aether templates list|install", Run: cmdTemplates},
		{Name: "server", Usage: "aether server token --name N | list | rm ID", Run: cmdServer},
		{Name: "migrate", Usage: "aether migrate --platform coolify|dokploy --dir DIR [--apply]", Run: cmdMigrate},
		{Name: "agent", Usage: "aether agent --core https://HOST:9443 --token T --name N [--labels k=v]", Run: cmdAgent},
		{Name: "export", Usage: "aether export [--file aether.yml]", Run: cmdExport},
		{Name: "import", Usage: "aether import --file aether.yml", Run: cmdImport},
		{Name: "update", Usage: "aether update --url URL [--sha256 HASH]", Run: cmdUpdate},
		{Name: "rollback-platform", Usage: "aether rollback-platform", Run: cmdRollbackPlatform},
		{Name: "uninstall", Usage: "aether uninstall [--purge]", Run: cmdUninstall},
	}
}

func flag(args []string, key string) string {
	for i, a := range args {
		if a == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func hasFlag(args []string, key string) bool {
	for _, a := range args {
		if a == key {
			return true
		}
	}
	return false
}

func cmdInstall(args []string) error {
	email := flag(args, "--email")
	if email == "" {
		email = "admin@aether.local"
	}
	password := flag(args, "--password")
	name := flag(args, "--name")
	if name == "" {
		name = "admin"
	}
	res, err := installer.Install(email, name, password)
	if err != nil {
		return err
	}
	fmt.Println("Aether instalado.")
	fmt.Printf("  estado:    %s\n", res.StateDir)
	fmt.Printf("  api:       %s\n", res.APIAddr)
	fmt.Printf("  admin:     %s\n", res.Email)
	fmt.Printf("  senha:     %s\n", res.Password)
	fmt.Println("Guarde a senha. Inicie o servidor com: aether serve")
	return nil
}

func cmdServe(args []string) error {
	addr := flag(args, "--addr")
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.SetOutput(security.NewSanitizingWriter(io.MultiWriter(os.Stderr, openLogFile(cfg))))
	if addr != "" {
		cfg.APIAddr = addr
	}
	c, err := core.New(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx); err != nil {
		return err
	}
	defer c.Stop(ctx)

	webDir := filepath.Join(os.Getenv("AETHER_WEB_DIR"))
	if webDir == "" {
		webDir = defaultWebDir()
	}
	server := api.New(c, webDir)
	ln, err := net.Listen("tcp", cfg.APIAddr)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(cfg.StateDir, "core.pid")
	os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o640)
	fmt.Printf("Aether core em http://%s (runtime: %s v%s)\n", cfg.APIAddr, c.DriverInfo.Driver, c.DriverInfo.Version)

	httpServer := &http.Server{Handler: server.Handler()}
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(ln) }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sig:
		fmt.Println("\nEncerrando...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		httpServer.Shutdown(shutdownCtx)
		os.Remove(pidPath)
		return nil
	}
}

func defaultWebDir() string {
	bin, err := os.Executable()
	if err != nil {
		return ""
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return ""
	}
	root := filepath.Dir(filepath.Dir(bin))
	dist := filepath.Join(root, "web", "dist")
	if info, err := os.Stat(filepath.Join(dist, "index.html")); err == nil && !info.IsDir() {
		return dist
	}
	legacy := filepath.Join(root, "web")
	if info, err := os.Stat(filepath.Join(legacy, "index.html")); err == nil && !info.IsDir() {
		return legacy
	}
	return root + "/web"
}

func getClient() (*client.Client, error) {
	cfg, err := client.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("não logado; use: aether login (config: %s)", client.ConfigPath())
	}
	return client.New(cfg), nil
}

func cmdLogin(args []string) error {
	email := flag(args, "--email")
	password := flag(args, "--password")
	server := flag(args, "--server")
	if email == "" || password == "" {
		return fmt.Errorf("uso: aether login --email E --password P [--server URL]")
	}
	if server == "" {
		server = "http://127.0.0.1:8080"
	}
	cfg, err := client.New(&client.Config{Server: server}).Login(email, password)
	if err != nil {
		return err
	}
	if err := client.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Println("Login efetuado:", server)
	return nil
}

func cmdWhoami(args []string) error {
	cl, err := getClient()
	if err != nil {
		return err
	}
	var me map[string]any
	if err := cl.GetJSON("/api/v1/me", &me); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(me, "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdMembers(args []string) error {
	if len(args) < 1 || args[0] != "list" {
		return fmt.Errorf("uso: aether members list")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	var members []map[string]any
	if err := cl.GetJSON("/api/v1/members", &members); err != nil {
		return err
	}
	for _, m := range members {
		fmt.Printf("%-30s %-20s %s\n", m["email"], m["name"], m["role"])
	}
	return nil
}

func cmdApiKeys(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether api-keys create --name N [--scope S]")
	}
	switch args[0] {
	case "create":
		cl, err := getClient()
		if err != nil {
			return err
		}
		name := flag(args, "--name")
		if name == "" {
			name = "default"
		}
		scopes := []string{}
		for i, a := range args {
			if a == "--scope" && i+1 < len(args) {
				scopes = append(scopes, args[i+1])
			}
		}
		var resp struct {
			Key string `json:"key"`
		}
		if err := cl.PostJSON("/api/v1/api-keys", map[string]any{"name": name, "scopes": scopes}, &resp); err != nil {
			return err
		}
		fmt.Println("Chave criada (mostrada uma única vez):")
		fmt.Println(resp.Key)
		return nil
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdProjects(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether projects create NAME | aether projects list")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var projects []map[string]any
		if err := cl.GetJSON("/api/v1/projects", &projects); err != nil {
			return err
		}
		for _, p := range projects {
			fmt.Printf("%-36s %s\n", p["id"], p["name"])
		}
		return nil
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("uso: aether projects create NAME")
		}
		var p map[string]any
		if err := cl.PostJSON("/api/v1/projects", map[string]string{"name": args[1]}, &p); err != nil {
			return err
		}
		fmt.Printf("Projeto criado: %s (%s)\n", p["name"], p["id"])
		return nil
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdApps(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether apps list | create | deploy | logs | rollback | env | domains | rm")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var apps []map[string]any
		if err := cl.GetJSON("/api/v1/apps", &apps); err != nil {
			return err
		}
		fmt.Printf("%-36s %-24s %-8s %s\n", "id", "name", "source", "image/git")
		for _, a := range apps {
			src, _ := a["source_type"].(string)
			ref, _ := a["image"].(string)
			if src == "git" {
				ref, _ = a["git_url"].(string)
			}
			fmt.Printf("%-36s %-24s %-8s %s\n", a["id"], a["name"], src, ref)
		}
		return nil
	case "create":
		projectID := flag(args, "--project")
		if projectID == "" {
			return fmt.Errorf("--project obrigatório")
		}
		name := flag(args, "--name")
		source := flag(args, "--source")
		if source == "" {
			source = "image"
		}
		if name == "" {
			return fmt.Errorf("--name obrigatório")
		}
		body := map[string]any{
			"name":        name,
			"source_type": source,
			"image":       flag(args, "--image"),
			"git_url":     flag(args, "--git-url"),
			"git_branch":  flag(args, "--branch"),
			"dockerfile":  flag(args, "--dockerfile"),
		}
		if v := flag(args, "--port"); v != "" {
			body["port"], _ = strconv.Atoi(v)
		}
		if v := flag(args, "--mem"); v != "" {
			mem, _ := strconv.ParseInt(v, 10, 64)
			body["resources"] = map[string]any{"mem_mb": mem}
		}
		if hasFlag(args, "--health") {
			body["health_check"] = map[string]any{
				"enabled": true, "path": "/", "interval_ms": 5000, "timeout_ms": 2000, "retries": 3,
			}
		}
		var app map[string]any
		if err := cl.PostJSON("/api/v1/projects/"+projectID+"/apps", body, &app); err != nil {
			return err
		}
		fmt.Printf("App criado: %s (%s)\n", app["name"], app["id"])
		return nil
	case "deploy":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		var dep map[string]any
		if err := cl.PostJSON("/api/v1/apps/"+appID+"/deploy", nil, &dep); err != nil {
			return err
		}
		fmt.Printf("Deploy #%v iniciado (%s)\n", dep["number"], dep["id"])
		return nil
	case "logs":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		return cl.Logs(appID, hasFlag(args, "--follow"))
	case "rollback":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		var dep map[string]any
		if err := cl.PostJSON("/api/v1/apps/"+appID+"/rollback", nil, &dep); err != nil {
			return err
		}
		fmt.Printf("Rollback iniciado: deploy #%v\n", dep["number"])
		return nil
	case "env":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		if hasFlag(args, "--unset") {
			return cl.Delete("/api/v1/apps/" + appID + "/env/" + flag(args, "--unset"))
		}
		name := flag(args, "--var")
		value := flag(args, "--value")
		if name == "" {
			return fmt.Errorf("env: use --var NAME [--value V] [--secret]")
		}
		if value == "" && !hasFlag(args, "--secret") {
			fmt.Print("valor: ")
			fmt.Scanln(&value)
		}
		return cl.PutJSON("/api/v1/apps/"+appID+"/env", map[string]any{
			"name": name, "value": value, "secret": hasFlag(args, "--secret"),
		}, nil)
	case "domains":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		host := flag(args, "--add")
		if host != "" {
			return cl.PostJSON("/api/v1/apps/"+appID+"/domains", map[string]any{
				"host": host, "https": hasFlag(args, "--https"),
			}, nil)
		}
		var domains []map[string]any
		if err := cl.GetJSON("/api/v1/apps/"+appID+"/deployments", nil); err != nil {
			_ = err
		}
		var app map[string]any
		if err := cl.GetJSON("/api/v1/apps/"+appID, &app); err != nil {
			return err
		}
		_ = domains
		fmt.Println("Para listar domínios, use a API ou a UI (GET /api/v1/apps/{id}).")
		return nil
	case "rm":
		appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
		if err != nil {
			return err
		}
		return cl.Delete("/api/v1/apps/" + appID)
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func resolveApp(cl *client.Client, id, name string) (string, error) {
	if id != "" {
		return id, nil
	}
	var apps []map[string]any
	if err := cl.GetJSON("/api/v1/apps", &apps); err != nil {
		return "", err
	}
	if name == "" {
		if len(apps) == 0 {
			return "", fmt.Errorf("nenhuma aplicação; crie uma ou use --app")
		}
		id, _ = apps[0]["id"].(string)
		return id, nil
	}
	for _, a := range apps {
		if a["name"] == name {
			id, _ = a["id"].(string)
			return id, nil
		}
	}
	return "", fmt.Errorf("aplicação %q não encontrada", name)
}

func cmdDeploy(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether deploy APP [--project P]")
	}
	return cmdApps(append([]string{"deploy"}, args...))
}

func cmdBackups(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether backups create | list")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "create":
		var b map[string]any
		if err := cl.PostJSON("/api/v1/backups", nil, &b); err != nil {
			return err
		}
		fmt.Printf("Backup criado: %s (%d bytes)\n", b["id"], int64(b["size"].(float64)))
		return nil
	case "list":
		var backups []map[string]any
		if err := cl.GetJSON("/api/v1/backups", &backups); err != nil {
			return err
		}
		for _, b := range backups {
			fmt.Printf("%-36s %12d  %s\n", b["id"], int64(b["size"].(float64)), b["created_at"])
		}
		return nil
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdStatus(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fmt.Printf("estado:      %s\n", cfg.StateDir)
	if data, err := os.ReadFile(filepath.Join(cfg.StateDir, "core.pid")); err == nil {
		fmt.Printf("core pid:    %s\n", strings.TrimSpace(string(data)))
	}
	c, err := core.New(cfg)
	if err != nil {
		return err
	}
	defer c.Stop(context.Background())
	fmt.Printf("runtime:     %s v%s (rootless=%v)\n", c.DriverInfo.Driver, c.DriverInfo.Version, c.DriverInfo.Rootless)
	apps, err := c.Store.ListAllApps()
	if err == nil {
		fmt.Printf("aplicações:  %d\n", len(apps))
	}
	count, _ := c.Bus.Count()
	fmt.Printf("eventos:     %d\n", count)
	fmt.Printf("proxy:       %v\n", c.Net.ProxyRunning())
	return nil
}

func cmdUpdate(args []string) error {
	url := flag(args, "--url")
	if url == "" {
		return fmt.Errorf("--url obrigatório")
	}
	sha := flag(args, "--sha256")
	fmt.Println("Atualizando...")
	return installer.Update(url, sha)
}

func cmdRollbackPlatform(args []string) error {
	return installer.Rollback()
}

func cmdUninstall(args []string) error {
	return installer.Uninstall(hasFlag(args, "--purge"))
}

func cmdDatabases(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether databases create|list|rm|backup")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var dbs []map[string]any
		if err := cl.GetJSON("/api/v1/databases", &dbs); err != nil {
			return err
		}
		for _, d := range dbs {
			fmt.Printf("%-36s %-20s %-8s %s\n", d["id"], d["name"], d["engine"], d["status"])
		}
		return nil
	case "create":
		name := flag(args, "--name")
		engine := flag(args, "--engine")
		project := flag(args, "--project")
		if name == "" || engine == "" || project == "" {
			return fmt.Errorf("--name --engine --project obrigatórios")
		}
		var db map[string]any
		if err := cl.PostJSON("/api/v1/databases", map[string]any{
			"project_id": project, "name": name, "engine": engine, "version": flag(args, "--version"),
		}, &db); err != nil {
			return err
		}
		fmt.Printf("Banco criado: %s (%s)\n", db["name"], db["id"])
		return nil
	case "rm":
		id := flag(args, "--id")
		if id == "" {
			return fmt.Errorf("--id obrigatório")
		}
		return cl.Delete("/api/v1/databases/" + id)
	case "backup":
		id := flag(args, "--id")
		if id == "" {
			return fmt.Errorf("--id obrigatório")
		}
		var b map[string]any
		if err := cl.PostJSON("/api/v1/databases/"+id+"/backup", nil, &b); err != nil {
			return err
		}
		fmt.Printf("Backup criado: %s\n", b["id"])
		return nil
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdCron(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether cron create|list|rm --app")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
	if err != nil {
		return err
	}
	switch args[0] {
	case "create":
		name := flag(args, "--job")
		schedule := flag(args, "--schedule")
		command := flag(args, "--command")
		if name == "" || schedule == "" || command == "" {
			return fmt.Errorf("--job --schedule --command obrigatórios (schedule: cron 5 campos)")
		}
		var j map[string]any
		if err := cl.PostJSON("/api/v1/apps/"+appID+"/cron-jobs", map[string]string{
			"name": name, "schedule": schedule, "command": command,
		}, &j); err != nil {
			return err
		}
		fmt.Printf("Cron criado: %s\n", j["id"])
		return nil
	case "list":
		var jobs []map[string]any
		if err := cl.GetJSON("/api/v1/apps/"+appID+"/cron-jobs", &jobs); err != nil {
			return err
		}
		for _, j := range jobs {
			fmt.Printf("%-36s %-16s %-16s %s\n", j["id"], j["name"], j["schedule"], j["enabled"])
		}
		return nil
	case "rm":
		return cl.Delete("/api/v1/cron-jobs/" + flag(args, "--id"))
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdWorkers(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether workers create|list|start|stop|rm --app")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
	if err != nil {
		return err
	}
	switch args[0] {
	case "create":
		name := flag(args, "--worker")
		command := flag(args, "--command")
		if name == "" || command == "" {
			return fmt.Errorf("--worker --command obrigatórios")
		}
		var w map[string]any
		if err := cl.PostJSON("/api/v1/apps/"+appID+"/workers", map[string]string{
			"name": name, "command": command,
		}, &w); err != nil {
			return err
		}
		fmt.Printf("Worker criado: %s\n", w["id"])
		return nil
	case "list":
		var ws []map[string]any
		if err := cl.GetJSON("/api/v1/apps/"+appID+"/workers", &ws); err != nil {
			return err
		}
		for _, w := range ws {
			fmt.Printf("%-36s %-16s %-10s %s\n", w["id"], w["name"], w["status"], w["command"])
		}
		return nil
	case "start":
		return cl.PostJSON("/api/v1/workers/"+flag(args, "--id")+"/start", nil, nil)
	case "stop":
		return cl.PostJSON("/api/v1/workers/"+flag(args, "--id")+"/stop", nil, nil)
	case "rm":
		return cl.Delete("/api/v1/workers/" + flag(args, "--id"))
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdCompose(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether compose create|list|up|down|rm")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var stacks []map[string]any
		if err := cl.GetJSON("/api/v1/compose", &stacks); err != nil {
			return err
		}
		for _, s := range stacks {
			fmt.Printf("%-36s %-20s %s\n", s["id"], s["name"], s["status"])
		}
		return nil
	case "create":
		project := flag(args, "--project")
		name := flag(args, "--name")
		file := flag(args, "--file")
		if project == "" || name == "" || file == "" {
			return fmt.Errorf("--project --name --file obrigatórios")
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		var stack map[string]any
		if err := cl.PostJSON("/api/v1/compose", map[string]any{
			"project_id": project, "name": name, "compose": string(data),
		}, &stack); err != nil {
			return err
		}
		fmt.Printf("Stack criada: %s\n", stack["id"])
		return nil
	case "up":
		return cl.PostJSON("/api/v1/compose/"+flag(args, "--id")+"/up", nil, nil)
	case "down":
		return cl.PostJSON("/api/v1/compose/"+flag(args, "--id")+"/down", nil, nil)
	case "rm":
		return cl.Delete("/api/v1/compose/" + flag(args, "--id"))
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdPreviews(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether previews create|list|rm --app")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	appID, err := resolveApp(cl, flag(args, "--app"), flag(args, "--name"))
	if err != nil {
		return err
	}
	switch args[0] {
	case "create":
		branch := flag(args, "--branch")
		if branch == "" {
			return fmt.Errorf("--branch obrigatório")
		}
		var p map[string]any
		if err := cl.PostJSON("/api/v1/apps/"+appID+"/previews", map[string]string{"branch": branch}, &p); err != nil {
			return err
		}
		fmt.Printf("Preview criado: %s (%s)\n", p["id"], p["domain"])
		return nil
	case "list":
		var ps []map[string]any
		if err := cl.GetJSON("/api/v1/apps/"+appID+"/previews", &ps); err != nil {
			return err
		}
		for _, p := range ps {
			fmt.Printf("%-36s %-20s %-40s %s\n", p["id"], p["branch"], p["domain"], p["status"])
		}
		return nil
	case "rm":
		return cl.Delete("/api/v1/previews/" + flag(args, "--id"))
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdTemplates(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uso: aether templates list|install")
	}
	cl, err := getClient()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var templates []map[string]any
		if err := cl.GetJSON("/api/v1/templates", &templates); err != nil {
			return err
		}
		for _, t := range templates {
			fmt.Printf("%-28s %-32s %s\n", t["id"], t["name"], t["description"])
		}
		return nil
	case "install":
		id := flag(args, "--id")
		project := flag(args, "--project")
		name := flag(args, "--name")
		if id == "" || project == "" {
			return fmt.Errorf("--id --project obrigatórios")
		}
		var stack map[string]any
		if err := cl.PostJSON("/api/v1/templates/"+id+"/install", map[string]any{
			"project_id": project, "name": name,
		}, &stack); err != nil {
			return err
		}
		fmt.Printf("Template instalado: %s\n", stack["id"])
		return nil
	}
	return fmt.Errorf("subcomando desconhecido: %s", args[0])
}

func cmdExport(args []string) error {
	cl, err := getClient()
	if err != nil {
		return err
	}
	file := flag(args, "--file")
	if file == "" {
		file = "aether.yml"
	}
	req, err := http.NewRequest("GET", cl.Server+"/api/v1/org/export", nil)
	if err != nil {
		return err
	}
	if cl.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cl.Token)
	}
	resp, err := cl.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("export falhou: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o640)
}

func cmdImport(args []string) error {
	cl, err := getClient()
	if err != nil {
		return err
	}
	file := flag(args, "--file")
	if file == "" {
		return fmt.Errorf("--file obrigatório")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", cl.Server+"/api/v1/org/import", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/yaml")
	if cl.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cl.Token)
	}
	resp, err := cl.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import falhou: %s", strings.TrimSpace(string(body)))
	}
	fmt.Println("Importado com sucesso")
	return nil
}

func cmdServer(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: aether server token|list|rm")
	}
	switch args[0] {
	case "token":
		name := flag(args, "--name")
		if name == "" {
			return fmt.Errorf("--name obrigatório")
		}
		tok, err := core.NewServerToken(name)
		if err != nil {
			return err
		}
		fmt.Printf("registre o agente com:\n  aether agent --core https://127.0.0.1:9443 --token %s --name %s\n", tok, name)
		return nil
	case "list":
		servers, err := core.ListServersLocal()
		if err != nil {
			return err
		}
		for _, srv := range servers {
			fmt.Printf("%-24s %-12s %-10s %-16s load=%4.2f version=%s\n",
				srv.ID, srv.Name, srv.Status, srv.LastHeartbeat.Format("15:04:05"), srv.Load, srv.Version)
		}
		return nil
	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("uso: aether server rm ID")
		}
		return core.DeleteServerLocal(args[1])
	default:
		return fmt.Errorf("subcomando desconhecido: %s", args[0])
	}
}

func cmdAgent(args []string) error {
	coreURL := flag(args, "--core")
	token := flag(args, "--token")
	name := flag(args, "--name")
	if coreURL == "" || token == "" {
		return fmt.Errorf("uso: aether agent --core URL --token T --name N [--labels a=b,c=d]")
	}
	if name == "" {
		name, _ = os.Hostname()
	}
	var labels []string
	if l := flag(args, "--labels"); l != "" {
		labels = strings.Split(l, ",")
	}
	return agent.Run(coreURL, token, name, labels, filepath.Join(config.DefaultStateDir(), "agent"))
}

func cmdMigrate(args []string) error {
	platform := flag(args, "--platform")
	dir := flag(args, "--dir")
	if platform == "" || dir == "" {
		return fmt.Errorf("uso: aether migrate --platform coolify|dokploy --dir DIR [--apply]")
	}
	var res *core.MigrateResult
	var err error
	if platform == "coolify" {
		res, err = core.DiscoverCoolify(dir)
	} else if platform == "dokploy" {
		res, err = core.DiscoverDokploy(dir)
	} else {
		return fmt.Errorf("plataforma desconhecida: %s", platform)
	}
	if err != nil {
		return err
	}
	fmt.Printf("plataforma: %s — %d serviços encontrados\n", res.Platform, len(res.Services))
	for _, svc := range res.Services {
		fmt.Printf("  - %-28s env=%d secrets=%d compose=%d bytes\n",
			svc.Name, len(svc.Env), len(svc.Secrets), len(svc.Compose))
	}
	if flagBool(args, "--apply") {
		cfg := loadCLIConfig()
		c, err := core.New(cfg)
		if err != nil {
			return err
		}
		defer c.Stop(context.Background())
		user, _, err := c.Login("", "", "")
		_ = user
		orgs, _ := c.Store.ListOrgs()
		if len(orgs) == 0 {
			return fmt.Errorf("nenhuma org disponível")
		}
		n, err := c.ImportDiscovered(orgs[0].ID, res)
		if err != nil {
			return err
		}
		fmt.Printf("importados %d serviços na org %s\n", n, orgs[0].Name)
	}
	return nil
}

func loadCLIConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{StateDir: config.DefaultStateDir()}
	}
	_ = cfg.EnsureDirs()
	return cfg
}

func flagBool(args []string, key string) bool {
	for _, a := range args {
		if a == key {
			return true
		}
	}
	return false
}

func openLogFile(cfg *config.Config) io.Writer {
	if err := os.MkdirAll(cfg.LogsDir, 0o750); err != nil {
		return nil
	}
	f, err := os.OpenFile(cfg.LogsDir+"/aether.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil
	}
	return f
}
