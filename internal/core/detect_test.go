package core

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetectFrameworks(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
		port  int
	}{
		{"nextjs", map[string]string{"package.json": `{"dependencies":{"next":"14"}}`}, "Next.js", 3000},
		{"vite-react", map[string]string{"package.json": `{"dependencies":{"react":"18","vite":"5"}}`}, "Vite React", 80},
		{"express", map[string]string{"package.json": `{"dependencies":{"express":"4"}}`}, "Express", 3000},
		{"go", map[string]string{"go.mod": "module github.com/x/app\n"}, "Go", 8080},
		{"rust", map[string]string{"Cargo.toml": "[package]\nname=app\n"}, "Rust", 8080},
		{"fastapi", map[string]string{"pyproject.toml": "[project]\ndependencies=['fastapi']\n"}, "FastAPI", 8000},
		{"rails", map[string]string{"Gemfile": "gem 'rails'\n"}, "Rails", 3000},
		{"dockerfile", map[string]string{"Dockerfile": "FROM nginx\n"}, "Generic (Dockerfile)", 8080},
		{"unknown", map[string]string{"README.md": "hello\n"}, "Unknown", 8080},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			res := detectInDir(dir)
			if res.Framework != tc.want {
				t.Fatalf("framework: esperado %s, obtido %s", tc.want, res.Framework)
			}
			if res.Port != tc.port {
				t.Fatalf("port: esperado %d, obtido %d", tc.port, res.Port)
			}
			if !res.Detected && tc.want != "Unknown" {
				t.Fatal("deveria ter detectado")
			}
		})
	}
}
