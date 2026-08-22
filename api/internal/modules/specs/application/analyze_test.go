package application

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/modules/specs/domain"
)

func TestDetectInDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"},"dependencies":{"react":"^18"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res := detectInDir(dir)
	if res.Framework != "Vite React" {
		t.Fatalf("framework: %s", res.Framework)
	}
	if !res.Detected || res.Port != 80 {
		t.Fatalf("detected/port: %v %d", res.Detected, res.Port)
	}
}

func TestDetectInDirGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := detectInDir(dir)
	if res.Framework != "Go" || res.Port != 8080 {
		t.Fatalf("go: %+v", res)
	}
}

func TestDetectInDirUnknown(t *testing.T) {
	res := detectInDir(t.TempDir())
	if res.Detected {
		t.Fatalf("expected not detected: %+v", res)
	}
}

func TestPlanPreview(t *testing.T) {
	a := &Analyzer{}
	plan := &domain.Plan{AppType: "spa", Framework: "Vite React"}
	preview, err := a.PlanPreview(plan, 8080)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Dockerfile, "nginx:alpine") {
		t.Fatalf("dockerfile: %s", preview.Dockerfile)
	}
	if !strings.Contains(preview.NginxConf, "try_files") {
		t.Fatalf("nginx spa fallback: %s", preview.NginxConf)
	}
}

func TestPlanPreviewNil(t *testing.T) {
	a := &Analyzer{}
	if _, err := a.PlanPreview(nil, 0); err != domain.ErrValidation {
		t.Fatalf("err: %v", err)
	}
}

func TestAnalyzeRepoUpload(t *testing.T) {
	uploads := t.TempDir()
	appDir := filepath.Join(uploads, "proj")
	if err := os.MkdirAll(appDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"scripts":{"build":"vite build"},"devDependencies":{"vite":"^5"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Analyzer{UploadsDir: uploads}
	plan, err := a.AnalyzeRepo(t.Context(), "", "", "proj")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Framework != "Vite" || !plan.Detected {
		t.Fatalf("plan: %+v", plan)
	}
}

func TestAnalyzeRepoUploadMissing(t *testing.T) {
	a := &Analyzer{UploadsDir: t.TempDir()}
	if _, err := a.AnalyzeRepo(t.Context(), "", "", "nope"); err != domain.ErrValidation {
		t.Fatalf("err: %v", err)
	}
}

func TestExtractZipPathTraversal(t *testing.T) {
	dest := t.TempDir()
	raw := makeZip(t, map[string]string{"../../evil.txt": "bad"})
	err := extractZip(raw, dest)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "evil.txt")); err == nil {
		t.Fatalf("path traversal escreveu fora do destino")
	}
}

func TestExtractZipLimits(t *testing.T) {
	dest := t.TempDir()
	raw := makeZip(t, map[string]string{"a.txt": string(make([]byte, 70<<20))})
	err := extractZip(raw, dest)
	if err == nil {
		t.Fatalf("arquivo grande deveria falhar")
	}
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
