// Package planner implements the Smart Frontend Detection & Deployment Engine.
// It analyzes a source directory and produces a full deployment plan:
// framework, package manager, runtime, build command, output dir, routing
// strategy, generated nginx.conf and Dockerfile.
package planner

import (
	"os"
	"path/filepath"
	"strings"
)

type AppType string

const (
	TypeStatic  AppType = "static" // plain HTML, no SPA fallback
	TypeSPA     AppType = "spa"    // client-side routing, needs /index.html fallback
	TypeSSG     AppType = "ssg"    // static generated, no global fallback
	TypeSSR     AppType = "ssr"    // server rendered — cannot be static
	TypeUnknown AppType = "unknown"
)

type Plan struct {
	Framework      string   `json:"framework"`
	Library        string   `json:"library"`
	PackageManager string   `json:"package_manager"`
	BuildCommand   string   `json:"build_command"`
	InstallCommand string   `json:"install_command"`
	OutputDir      string   `json:"output_dir"`
	AppType        AppType  `json:"app_type"`
	WebServer      string   `json:"web_server"`
	ContainerPort  int      `json:"container_port"`
	SPAFallback    bool     `json:"spa_fallback"`
	IndexFile      string   `json:"index_file"`
	HasLockfile    bool     `json:"has_lockfile"`
	Detected       bool     `json:"detected"`
	NginxConf      string   `json:"nginx_conf"`
	Dockerfile     string   `json:"dockerfile"`
	Warnings       []string `json:"warnings"`
}

// Detect analyzes a source directory and produces a deployment plan.
func Detect(srcDir string) (*Plan, error) {
	p := &Plan{
		Framework:     "Unknown",
		WebServer:     "nginx",
		ContainerPort: 80,
		IndexFile:     "index.html",
		AppType:       TypeUnknown,
	}
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(srcDir, name))
		return err == nil
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			return ""
		}
		return string(b)
	}

	p.HasLockfile = has("bun.lockb") || has("bun.lock") || has("pnpm-lock.yaml") || has("package-lock.json") || has("yarn.lock")

	switch {
	case has("bun.lockb") || has("bun.lock"):
		p.PackageManager = "bun"
	case has("pnpm-lock.yaml"):
		p.PackageManager = "pnpm"
	case has("package-lock.json") || has("npm-shrinkwrap.json"):
		p.PackageManager = "npm"
	case has("yarn.lock"):
		p.PackageManager = "yarn"
	case has("deno.json") || has("deno.lock"):
		p.PackageManager = "deno"
	default:
		p.PackageManager = "npm"
	}

	pkg := read("package.json")
	framework, library, buildScript := detectFromPackage(pkg)
	p.Framework = framework
	p.Library = library

	if p.Framework == "" {
		detectFromFiles(srcDir, read, p)
	}

	p.BuildCommand = detectBuildCommand(p, buildScript)

	p.OutputDir = detectOutputDir(srcDir, p, read)

	detectRouting(p)

	discoverIndex(srcDir, p)

	p.Detected = p.Framework != "" && p.AppType != TypeUnknown

	if p.AppType == TypeSSR {
		p.WebServer = "node"
		p.ContainerPort = 3000
		p.NginxConf = ""
	} else {
		p.WebServer = "nginx"
		p.NginxConf = GenerateNginxConf(p)
	}
	p.Dockerfile = GenerateDockerfile(p)

	if p.Framework == "Unknown" || p.AppType == TypeUnknown {
		p.Warnings = append(p.Warnings, "framework not recognized — check the repository files")
	}
	return p, nil
}

func detectFromPackage(pkg string) (framework, library, buildScript string) {
	if pkg == "" {
		return "", "", ""
	}
	lower := strings.ToLower(pkg)
	has := func(s string) bool { return strings.Contains(lower, s) }
	scripts := extractScripts(pkg)
	buildScript = scripts["build"]

	// SSR first — must never be deployed as static
	switch {
	case has(`"next"`):
		return "Next.js", "React", buildScript
	case has(`"nuxt"`):
		return "Nuxt", "Vue", buildScript
	case has(`"@nestjs/core"`) || has(`"@nestjs/common"`):
		return "NestJS", "Node.js", buildScript
	case has(`"@remix-run/react"`):
		return "Remix", "React", buildScript
	case has(`"@sveltejs/kit"`):
		return "SvelteKit", "Svelte", buildScript
	case has(`"@tanstack/react-start"`) || has(`"@tanstack/router"`) || has(`"@tanstack/react-router"`):
		return "TanStack Start", "React", buildScript
	case has(`"fresh"`):
		return "Fresh", "Preact", buildScript
	}

	switch {
	case has(`"astro"`):
		return "Astro", "Astro", buildScript
	case has(`"@docusaurus/core"`):
		return "Docusaurus", "React", buildScript
	case has(`"gatsby"`):
		return "Gatsby", "React", buildScript
	case has(`"vuepress"`):
		return "VuePress", "Vue", buildScript
	case has(`"vitepress"`):
		return "VitePress", "Vue", buildScript
	case has(`"@11ty/eleventy"`):
		return "Eleventy", "Eleventy", buildScript
	}

	switch {
	case has(`"react"`):
		if has(`"vite"`) {
			return "Vite", "React", buildScript
		}
		return "React (CRA/Webpack)", "React", buildScript
	case has(`"vue"`):
		if has(`"vite"`) {
			return "Vite", "Vue", buildScript
		}
		return "Vue", "Vue", buildScript
	case has(`"svelte"`):
		return "Svelte", "Svelte", buildScript
	case has(`"solid-js"`):
		return "Solid", "Solid", buildScript
	case has(`"@angular/core"`):
		return "Angular", "Angular", buildScript
	case has(`"lit"`):
		return "Lit", "Lit", buildScript
	case has(`"preact"`):
		return "Preact", "Preact", buildScript
	case has(`"qwik"`):
		return "Qwik", "Qwik", buildScript
	case has(`"vite"`):
		return "Vite", "Vanilla", buildScript
	case has(`"webpack"`):
		return "Webpack", "", buildScript
	}
	return "", "", buildScript
}

func detectFromFiles(srcDir string, read func(string) string, p *Plan) {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(srcDir, name))
		return err == nil
	}
	switch {
	case has("vite.config.ts") || has("vite.config.js") || has("vite.config.mjs"):
		p.Framework = "Vite"
	case has("astro.config.mjs") || has("astro.config.js"):
		p.Framework = "Astro"
	case has("next.config.js") || has("next.config.ts") || has("next.config.mjs"):
		p.Framework = "Next.js"
	case has("nuxt.config.ts") || has("nuxt.config.js"):
		p.Framework = "Nuxt"
	case has("angular.json"):
		p.Framework = "Angular"
	case has("svelte.config.js"):
		p.Framework = "SvelteKit"
	case has("gatsby-config.js"):
		p.Framework = "Gatsby"
	case has("vitepress.config.ts") || has("vitepress.config.mjs"):
		p.Framework = "VitePress"
	case has("webpack.config.js"):
		p.Framework = "Webpack"
	case has("rspack.config.js"):
		p.Framework = "Rspack"
	}
	if p.Framework == "" {
		switch {
		case has("src/main.tsx") || has("src/main.jsx"):
			p.Framework = "Vite"
		case has("pages") && has("public"):
			p.Framework = "Next.js"
		case has("app") && has("package.json"):
			p.Framework = "Unknown (App Router)"
		}
	}
}

func extractScripts(pkg string) map[string]string {
	out := map[string]string{}
	i := strings.Index(pkg, `"scripts"`)
	if i < 0 {
		return out
	}
	seg := pkg[i:]
	for _, m := range regexQuotedPairs(seg) {
		out[m[0]] = m[1]
		if len(out) > 24 {
			break
		}
	}
	return out
}

func detectBuildCommand(p *Plan, buildScript string) string {
	pm := p.PackageManager
	// buildScript é o VALOR do script package.json (ex: "vite build");
	// o comando correto é sempre "<pm> run <nome do script>".
	run := func(name string) string { return pm + " run " + name }
	if buildScript != "" && buildScript != "tsc" {
		return run("build")
	}
	switch p.Framework {
	case "Vite":
		return run("build")
	case "Astro":
		return run("build")
	case "Angular":
		return run("build")
	case "Next.js":
		return run("build")
	case "Nuxt":
		return run("build")
	case "NestJS":
		return run("build")
	case "SvelteKit":
		return run("build")
	case "Gatsby":
		return run("build")
	case "Docusaurus":
		return run("build")
	case "VitePress":
		return run("docs:build")
	case "Eleventy":
		return run("build")
	}
	if buildScript != "" {
		return run("build")
	}
	return ""
}
