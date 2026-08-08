package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/git"
)

type DetectResult struct {
	Framework    string `json:"framework"`
	BuildMethod  string `json:"build_method"`
	BuildCommand string `json:"build_command"`
	StartCommand string `json:"start_command"`
	OutputDir    string `json:"output_dir"`
	Port         int    `json:"port"`
	Detected     bool   `json:"detected"`
}

func (c *Core) DetectFramework(ctx context.Context, gitURL, branch string) (*DetectResult, error) {
	if gitURL == "" {
		return nil, nil
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

func detectInDir(dir string) *DetectResult {
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
		mod := read("go.mod")
		port := 8080
		if strings.Contains(mod, "module ") {
			port = 8080
		}
		return &DetectResult{Framework: "Go", BuildMethod: "nixpacks", BuildCommand: "go build -o /app/server", StartCommand: "./server", Port: port, Detected: true}
	}
	if has("Cargo.toml") {
		return &DetectResult{Framework: "Rust", BuildMethod: "nixpacks", BuildCommand: "cargo build --release", StartCommand: "./target/release/server", Port: 8080, Detected: true}
	}
	if has("mix.exs") {
		return &DetectResult{Framework: "Elixir/Phoenix", BuildMethod: "nixpacks", StartCommand: "mix phx.server", Port: 4000, Detected: true}
	}
	if has("Gemfile") {
		return &DetectResult{Framework: "Rails", BuildMethod: "nixpacks", StartCommand: "bundle exec rails s -b 0.0.0.0", Port: 3000, Detected: true}
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
			port = 8000
		}
		return &DetectResult{Framework: framework, BuildMethod: "nixpacks", StartCommand: start, Port: port, Detected: true}
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
			port = 3000
		} else if strings.Contains(pkg, `"nuxt"`) {
			framework = "Nuxt"
			buildCmd = "nuxt build"
			startCmd = "node .output/server/index.mjs"
			port = 3000
		} else if strings.Contains(pkg, `"svelte"`) && strings.Contains(pkg, "kit") {
			framework = "SvelteKit"
			buildCmd = "svelte-kit build"
			outputDir = "build"
			port = 3000
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
			port = 3000
		} else if strings.Contains(pkg, `"nest"`) {
			framework = "NestJS"
			buildCmd = "nest build"
			startCmd = "node dist/main"
			port = 3000
		} else if strings.Contains(pkg, `"remix"`) {
			framework = "Remix"
			buildCmd = "remix build"
			outputDir = "public"
			port = 3000
		} else if strings.Contains(pkg, `"astro"`) {
			framework = "Astro"
			buildCmd = "astro build"
			outputDir = "dist"
			port = 80
		}
		return &DetectResult{Framework: framework, BuildMethod: "nixpacks", BuildCommand: buildCmd, StartCommand: startCmd, OutputDir: outputDir, Port: port, Detected: true}
	}
	if has("deno.json") || has("deno.lock") {
		return &DetectResult{Framework: "Deno", BuildMethod: "nixpacks", StartCommand: "deno task start", Port: 8000, Detected: true}
	}
	if has("bun.lock") || has("bun.lockb") {
		return &DetectResult{Framework: "Bun", BuildMethod: "nixpacks", StartCommand: "bun start", Port: 3000, Detected: true}
	}
	if has("build.gradle") || has("pom.xml") {
		return &DetectResult{Framework: "Java/Spring", BuildMethod: "nixpacks", BuildCommand: "./mvnw package", StartCommand: "java -jar target/*.jar", Port: 8080, Detected: true}
	}
	if has("laravel") || has("artisan") {
		return &DetectResult{Framework: "Laravel", BuildMethod: "nixpacks", StartCommand: "php artisan serve --host=0.0.0.0", Port: 8000, Detected: true}
	}
	if has("Dockerfile") {
		return &DetectResult{Framework: "Generic (Dockerfile)", BuildMethod: "dockerfile", Port: 8080, Detected: true}
	}
	return &DetectResult{Framework: "Unknown", BuildMethod: "nixpacks", Port: 8080, Detected: false}
}
