package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"aether/internal/git"
	"aether/internal/planner"
	"aether/internal/specs/domain"
)

type Analyzer struct {
	UploadsDir string
}

type ZipUpload struct {
	ID     string `json:"upload_id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
}

func (a *Analyzer) SaveZipUpload(name string, data []byte) (*ZipUpload, error) {
	if err := os.MkdirAll(a.UploadsDir, 0o750); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	if err := os.WriteFile(filepath.Join(a.UploadsDir, id+".zip"), data, 0o640); err != nil {
		return nil, err
	}
	dest := filepath.Join(a.UploadsDir, id)
	if err := os.RemoveAll(dest); err != nil {
		return nil, err
	}
	if err := extractZip(data, dest); err != nil {
		_ = os.Remove(filepath.Join(a.UploadsDir, id+".zip"))
		return nil, domain.ErrValidation
	}
	flattenSingleRoot(dest)
	return &ZipUpload{ID: id, Name: name, Size: int64(len(data)), Status: "ready"}, nil
}

func (a *Analyzer) DetectRepo(ctx context.Context, gitURL, branch string) (*domain.DetectResult, error) {
	if gitURL == "" {
		return nil, domain.ErrValidation
	}
	dir, err := os.MkdirTemp("", "aether-detect-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := git.Clone(ctx, gitURL, branch, dir); err != nil {
		return nil, err
	}
	return detectInDir(dir), nil
}

func (a *Analyzer) AnalyzeRepo(ctx context.Context, gitURL, branch, uploadID string) (*domain.Plan, error) {
	if uploadID != "" {
		if a.UploadsDir == "" {
			return nil, domain.ErrValidation
		}
		srcDir := filepath.Join(a.UploadsDir, uploadID)
		if _, err := os.Stat(srcDir); err != nil {
			return nil, domain.ErrValidation
		}
		return planFrom(planner.Detect(srcDir))
	}
	if gitURL == "" {
		return nil, domain.ErrValidation
	}
	dir, err := os.MkdirTemp("", "aether-analyze-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if branch == "" {
		branch = "main"
	}
	if err := git.Clone(ctx, gitURL, branch, dir); err != nil {
		return nil, err
	}
	return planFrom(planner.Detect(dir))
}

func (a *Analyzer) PlanPreview(p *domain.Plan, port int) (*domain.PlanPreview, error) {
	if p == nil {
		return nil, domain.ErrValidation
	}
	plan := &planner.Plan{
		Framework: p.Framework, Library: p.Library, PackageManager: p.PackageManager,
		BuildCommand: p.BuildCommand, InstallCommand: p.InstallCommand,
		OutputDir: p.OutputDir, AppType: planner.AppType(p.AppType), WebServer: p.WebServer,
		ContainerPort: p.ContainerPort, SPAFallback: p.SPAFallback, IndexFile: p.IndexFile,
	}
	if port > 0 {
		plan.ContainerPort = port
	}
	if plan.AppType == planner.TypeSPA || plan.AppType == planner.TypeStatic {
		plan.SPAFallback = true
	}
	return &domain.PlanPreview{
		Dockerfile: planner.GenerateDockerfile(plan),
		NginxConf:  planner.GenerateNginxConf(plan),
	}, nil
}

func planFrom(plan *planner.Plan, err error) (*domain.Plan, error) {
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("empty plan")
	}
	return &domain.Plan{
		Framework: plan.Framework, Library: plan.Library, PackageManager: plan.PackageManager,
		BuildCommand: plan.BuildCommand, InstallCommand: plan.InstallCommand,
		OutputDir: plan.OutputDir, AppType: string(plan.AppType), WebServer: plan.WebServer,
		ContainerPort: plan.ContainerPort, SPAFallback: plan.SPAFallback, IndexFile: plan.IndexFile,
		HasLockfile: plan.HasLockfile, Detected: plan.Detected,
		NginxConf: plan.NginxConf, Dockerfile: plan.Dockerfile, Warnings: plan.Warnings,
	}, nil
}

func detectInDir(dir string) *domain.DetectResult {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return ""
		}
		return string(b)
	}
	if has("go.mod") {
		return &domain.DetectResult{Framework: "Go", BuildMethod: "buildpacks", BuildCommand: "go build -o /app/server", StartCommand: "./server", Port: 8080, Detected: true}
	}
	if has("Cargo.toml") {
		return &domain.DetectResult{Framework: "Rust", BuildMethod: "buildpacks", BuildCommand: "cargo build --release", StartCommand: "./target/release/server", Port: 8080, Detected: true}
	}
	if has("mix.exs") {
		return &domain.DetectResult{Framework: "Elixir/Phoenix", BuildMethod: "buildpacks", StartCommand: "mix phx.server", Port: 4000, Detected: true}
	}
	if has("Gemfile") {
		return &domain.DetectResult{Framework: "Rails", BuildMethod: "buildpacks", StartCommand: "bundle exec rails s -b 0.0.0.0", Port: 3000, Detected: true}
	}
	if has("pyproject.toml") || has("requirements.txt") || has("Pipfile") {
		framework := "Python"
		start := "python app.py"
		port := 8000
		if strings.Contains(read("pyproject.toml"), "fastapi") {
			framework = "FastAPI"
			start = "uvicorn main:app --host 0.0.0.0 --port 8000"
		} else if strings.Contains(read("pyproject.toml"), "django") {
			framework = "Django"
			start = "python manage.py runserver 0.0.0.0:8000"
		}
		return &domain.DetectResult{Framework: framework, BuildMethod: "buildpacks", StartCommand: start, Port: port, Detected: true}
	}
	if has("package.json") {
		pkg := read("package.json")
		framework := "Node.js"
		buildCmd := ""
		startCmd := "npm start"
		port := 3000
		outputDir := ""
		if strings.Contains(pkg, `"next"`) {
			framework = "Next.js"
			buildCmd = "next build"
			startCmd = "next start -p 3000"
			outputDir = ".next"
		} else if strings.Contains(pkg, `"nuxt"`) {
			framework = "Nuxt"
			buildCmd = "nuxt build"
			startCmd = "node .output/server/index.mjs"
			port = 3000
		} else if strings.Contains(pkg, `"svelte"`) && strings.Contains(pkg, "kit") {
			framework = "SvelteKit"
			buildCmd = "svelte-kit build"
			outputDir = "build"
		} else if strings.Contains(pkg, "vite") {
			if strings.Contains(pkg, `"react"`) {
				framework = "Vite React"
			} else {
				framework = "Vite"
			}
			buildCmd = "vite build"
			outputDir = "dist"
			port = 80
		} else if strings.Contains(pkg, `"express"`) {
			framework = "Express"
			startCmd = "node server.js"
		} else if strings.Contains(pkg, `"nest"`) {
			framework = "NestJS"
			buildCmd = "nest build"
			startCmd = "node dist/main"
			port = 3000
		} else if strings.Contains(pkg, `"remix"`) {
			framework = "Remix"
			buildCmd = "remix build"
			outputDir = "public"
		} else if strings.Contains(pkg, `"astro"`) {
			framework = "Astro"
			buildCmd = "astro build"
			outputDir = "dist"
			port = 80
		}
		return &domain.DetectResult{Framework: framework, BuildMethod: "buildpacks", BuildCommand: buildCmd, StartCommand: startCmd, OutputDir: outputDir, Port: port, Detected: true}
	}
	if has("deno.json") || has("deno.lock") {
		return &domain.DetectResult{Framework: "Deno", BuildMethod: "buildpacks", StartCommand: "deno task start", Port: 8000, Detected: true}
	}
	if has("bun.lock") || has("bun.lockb") {
		return &domain.DetectResult{Framework: "Bun", BuildMethod: "buildpacks", StartCommand: "bun start", Port: 3000, Detected: true}
	}
	if has("build.gradle") || has("pom.xml") {
		return &domain.DetectResult{Framework: "Java/Spring", BuildMethod: "buildpacks", BuildCommand: "./mvnw package", StartCommand: "java -jar target/*.jar", Port: 8080, Detected: true}
	}
	if has("laravel") || has("artisan") {
		return &domain.DetectResult{Framework: "Laravel", BuildMethod: "buildpacks", StartCommand: "php artisan serve --host=0.0.0.0", Port: 8000, Detected: true}
	}
	if has("Dockerfile") {
		return &domain.DetectResult{Framework: "Generic (Dockerfile)", BuildMethod: "dockerfile", Port: 8080, Detected: true}
	}
	return &domain.DetectResult{Framework: "Unknown", BuildMethod: "buildpacks", Port: 8080, Detected: false}
}

func extractZip(data []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	const maxTotal int64 = 512 << 20
	const maxFile int64 = 64 << 20
	var total int64
	for _, f := range zr.File {
		if isMacJunk(f.Name) {
			continue
		}
		if ignoreSourceEntry(f.Name) {
			continue
		}
		target := filepath.Join(dest, f.Name)
		if !within(dest, target) {
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
			continue
		}
		if f.UncompressedSize64 > uint64(maxFile) {
			return errors.New("file exceeds the limit")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return err
		}
		remaining := maxTotal - total
		if remaining <= 0 {
			_ = rc.Close()
			_ = out.Close()
			return errors.New("zip exceeds extraction limit")
		}
		limit := maxFile
		if remaining < limit {
			limit = remaining
		}
		n, err := io.Copy(out, io.LimitReader(rc, limit))
		_ = rc.Close()
		_ = out.Close()
		if err != nil {
			return err
		}
		total += n
	}
	return nil
}

func flattenSingleRoot(dest string) {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return
	}
	var root string
	for _, e := range entries {
		if isMacJunk(e.Name()) {
			continue
		}
		if root != "" || !e.IsDir() {
			return
		}
		root = e.Name()
	}
	if root == "" {
		return
	}
	rootDir := filepath.Join(dest, root)
	if err := copyDirContents(rootDir, dest); err != nil {
		return
	}
	_ = os.RemoveAll(rootDir)
}

func isMacJunk(name string) bool {
	base := filepath.Base(name)
	if base == ".DS_Store" || base == "__MACOSX" || strings.HasPrefix(base, "._") {
		return true
	}
	return strings.Contains(name, "__MACOSX/")
}

func ignoreSourceEntry(name string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSuffix(name, "/"), "./")
	parts := strings.Split(trimmed, "/")
	for _, p := range parts {
		if p == "node_modules" || p == ".git" {
			return true
		}
	}
	return false
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel)
}

func copyDirContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o750); err != nil {
				return err
			}
			if err := copyDirContents(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
