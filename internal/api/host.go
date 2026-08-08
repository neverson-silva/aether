package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"aether/internal/core"
)

func (s *Server) handleHostStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.core.HostStats(r.Context()))
}

func (s *Server) handleHostStatsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, 500, "stream não suportado")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)
	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	writeHostStatsEvent(w, flusher, s.core.HostStats(ctx))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			writeHostStatsEvent(w, flusher, s.core.HostStats(ctx))
		}
	}
}

func writeHostStatsEvent(w http.ResponseWriter, f http.Flusher, s any) {
	b, _ := json.Marshal(s)
	fmt.Fprintf(w, "event: stats\ndata: %s\n\n", b)
	f.Flush()
}

func (s *Server) handleHostEvents(w http.ResponseWriter, r *http.Request) {
	evs, err := s.core.Bus.Recent(r.Context(), 30)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(evs))
	for _, e := range evs {
		title, detail := formatHostEvent(e.Type, e.Payload, s.core)
		out = append(out, map[string]any{
			"ts":     e.TS.Format("15:04:05"),
			"type":   e.Type,
			"title":  title,
			"detail": detail,
		})
	}
	writeJSON(w, 200, out)
}

func formatHostEvent(typ string, payload []byte, c *core.Core) (string, string) {
	var p map[string]any
	_ = json.Unmarshal(payload, &p)
	appName := ""
	appID, _ := p["app_id"].(string)
	if appID == "" {
		if depID, _ := p["deployment_id"].(string); depID != "" {
			if d, err := c.Store.GetDeployment(depID); err == nil {
				appID = d.AppID
			}
		}
	}
	if appID != "" {
		if a, err := c.Store.GetApp(appID); err == nil {
			appName = a.Name
		}
	}
	num, _ := p["number"].(float64)
	numStr := ""
	if num > 0 {
		numStr = " #" + strconv.FormatFloat(num, 'f', 0, 64)
	}
	img, _ := p["image"].(string)
	dbName, _ := p["database"].(string)
	errMsg, _ := p["error"].(string)
	name, _ := p["name"].(string)
	parts := func(extra ...string) string {
		all := []string{}
		if appName != "" {
			all = append(all, appName)
		}
		for _, x := range extra {
			if x != "" {
				all = append(all, x)
			}
		}
		out := ""
		for i, s := range all {
			if i > 0 {
				out += " · "
			}
			out += s
		}
		return out
	}

	switch typ {
	case "app.deployed":
		return "Deploy successful" + numStr, parts(img)
	case "app.deploy_failed":
		return "Deploy failed" + numStr, parts(truncateStr(errMsg, 120))
	case "app.rolled_back":
		return "Rolled back" + numStr, appName
	case "deployment.created":
		return "Deploy queued" + numStr, appName
	case "deployment.building":
		bm, _ := p["build_method"].(string)
		return "Building" + numStr, parts(bm)
	case "deployment.starting":
		return "Starting" + numStr, appName
	case "deployment.healthcheck":
		path, _ := p["path"].(string)
		return "Health check" + numStr, parts(path)
	case "backup.started", "backup.finished", "backup.failed":
		title := map[string]string{"backup.started": "Backup started", "backup.finished": "Backup finished", "backup.failed": "Backup failed"}[typ]
		return title, dbName
	case "server.registered":
		return "Server registered", name
	case "server.marked_unhealthy":
		return "Server unhealthy", name
	case "server.recovered":
		return "Server recovered", name
	case "container.started":
		return "Container started", appName
	case "container.stopped":
		return "Container stopped", appName
	default:
		detail := parts()
		if detail == "" {
			detail = string(payload)
		}
		return typ, detail
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (s *Server) handleHostLogs(w http.ResponseWriter, r *http.Request) {
	follow := r.URL.Query().Get("follow") == "1"
	path := s.core.Cfg.LogsDir + "/aether.log"
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, 200, []map[string]string{})
		return
	}
	lines := tailLines(string(data), 400)
	if !follow {
		writeJSON(w, 200, lines)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, 500, "stream não suportado")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)
	for _, l := range lines {
		fmt.Fprintf(w, "event: log\ndata: %s\n\n", jsonStr(l))
	}
	flusher.Flush()
	if !follow {
		return
	}
	ctx := r.Context()
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	offset := stat.Size()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	buf := make([]byte, 8192)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cur, _ := f.Stat()
			if cur.Size() > offset {
				if _, err := f.Seek(offset, io.SeekStart); err != nil {
					return
				}
				for {
					n, rerr := f.Read(buf)
					if n > 0 {
						fmt.Fprintf(w, "event: log\ndata: %s\n\n", jsonStr(map[string]string{"line": string(buf[:n])}))
					}
					if rerr != nil {
						break
					}
				}
				flusher.Flush()
				offset = cur.Size()
			}
		}
	}
}

func tailLines(s string, n int) []map[string]string {
	out := []map[string]string{}
	start := 0
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			count++
			if count == n+1 {
				start = i + 1
				break
			}
		}
	}
	rest := s[start:]
	line := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == '\n' {
			if line != "" {
				out = append(out, map[string]string{"line": line})
			}
			line = ""
		} else {
			line += string(rest[i])
		}
	}
	if line != "" {
		out = append(out, map[string]string{"line": line})
	}
	return out
}

func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
