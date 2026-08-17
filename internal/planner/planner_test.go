package planner

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o750)
		os.WriteFile(path, []byte(content), 0o644)
	}
	return dir
}

const viteReact = `{"scripts":{"build":"vite build"},"dependencies":{"react":"^18","react-dom":"^18"},"devDependencies":{"vite":"^5"}}`
const nextApp = `{"scripts":{"build":"next build"},"dependencies":{"next":"^14","react":"^18"}}`
const astroApp = `{"scripts":{"build":"astro build"},"dependencies":{"astro":"^4"}}`
const angularApp = `{"scripts":{"build":"ng build"},"dependencies":{"@angular/core":"^17"}}`
const nuxtApp = `{"scripts":{"build":"nuxt build"},"dependencies":{"nuxt":"^3"}}`

func TestDetectViteReact(t *testing.T) {
	dir := write(t, map[string]string{"package.json": viteReact, "vite.config.ts": "export default { build: { outDir: 'custom-out' } }"})
	p, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Framework != "Vite" || p.Library != "React" {
		t.Fatalf("framework: %s/%s", p.Framework, p.Library)
	}
	if p.AppType != TypeSPA || !p.SPAFallback {
		t.Fatalf("deveria ser SPA com fallback: %v", p.AppType)
	}
	if p.OutputDir != "custom-out" {
		t.Fatalf("output custom: %s", p.OutputDir)
	}
	if !stringsContains(p.NginxConf, "/index.html") {
		t.Fatalf("nginx sem SPA fallback:\n%s", p.NginxConf)
	}
	if !stringsContains(p.Dockerfile, "nginx:alpine") || stringsContains(p.Dockerfile, "node_modules") {
		t.Fatalf("dockerfile inválido:\n%s", p.Dockerfile)
	}
}

func TestDetectNextSSR(t *testing.T) {
	dir := write(t, map[string]string{"package.json": nextApp})
	p, _ := Detect(dir)
	if p.AppType != TypeSSR {
		t.Fatalf("next deveria ser SSR, foi %v", p.AppType)
	}
	if len(p.Warnings) == 0 || !stringsContains(p.Warnings[0], "Server Side Rendering") {
		t.Fatalf("warnings: %v", p.Warnings)
	}
}

func TestDetectAstroSSG(t *testing.T) {
	dir := write(t, map[string]string{"package.json": astroApp})
	p, _ := Detect(dir)
	if p.AppType != TypeSSG || p.SPAFallback {
		t.Fatalf("astro deveria ser SSG sem fallback: %v", p.AppType)
	}
	if p.OutputDir != "dist" {
		t.Fatalf("output: %s", p.OutputDir)
	}
	if stringsContains(p.NginxConf, "/index.html;") {
		t.Fatalf("SSG não deve ter fallback global:\n%s", p.NginxConf)
	}
}

func TestDetectAngular(t *testing.T) {
	dir := write(t, map[string]string{"package.json": angularApp, "angular.json": `{"projects":{"myapp":{"architect":{"build":{"options":{"outputPath":"dist/myapp"}}}}}}`})
	p, _ := Detect(dir)
	if p.Framework != "Angular" || p.AppType != TypeSPA {
		t.Fatalf("%s %v", p.Framework, p.AppType)
	}
	if p.OutputDir != "dist/myapp" {
		t.Fatalf("output: %s", p.OutputDir)
	}
}

func TestDetectNuxtSSR(t *testing.T) {
	dir := write(t, map[string]string{"package.json": nuxtApp})
	p, _ := Detect(dir)
	if p.AppType != TypeSSR {
		t.Fatalf("nuxt deveria ser SSR: %v", p.AppType)
	}
}

func TestPackageManagers(t *testing.T) {
	cases := []struct{ file, want string }{
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"package-lock.json", "npm"},
		{"yarn.lock", "yarn"},
	}
	for _, c := range cases {
		dir := write(t, map[string]string{"package.json": viteReact, c.file: ""})
		p, _ := Detect(dir)
		if p.PackageManager != c.want {
			t.Fatalf("%s: esperado %s, obtido %s", c.file, c.want, p.PackageManager)
		}
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestDetectNextSSRDockerfile(t *testing.T) {
	dir := write(t, map[string]string{"package.json": nextApp})
	p, _ := Detect(dir)
	if p.WebServer != "node" || p.ContainerPort != 3000 {
		t.Fatalf("SSR deveria usar node:3000, foi %s:%d", p.WebServer, p.ContainerPort)
	}
	if p.NginxConf != "" {
		t.Fatalf("SSR não deve ter nginx.conf:\n%s", p.NginxConf)
	}
	if !stringsContains(p.Dockerfile, "AS runtime") || !stringsContains(p.Dockerfile, "next start") {
		t.Fatalf("Dockerfile SSR inválido:\n%s", p.Dockerfile)
	}
	if stringsContains(p.Dockerfile, "nginx:alpine") {
		t.Fatalf("SSR não deve usar nginx:\n%s", p.Dockerfile)
	}
}
