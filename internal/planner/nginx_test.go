package planner

import (
	"strings"
	"testing"
)

func TestNginxDeniesEnvFiles(t *testing.T) {
	p := &Plan{IndexFile: "index.html", SPAFallback: true}
	conf := GenerateNginxConf(p)
	if !strings.Contains(conf, "location ~ /\\.env { deny all; }") {
		t.Fatalf("nginx.conf deveria negar .env files:\n%s", conf)
	}
	if !strings.Contains(conf, "try_files $uri $uri/ /index.html;") {
		t.Fatalf("nginx.conf deveria ter SPA fallback")
	}
}
