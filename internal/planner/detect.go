package planner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var quotedPairRe = regexp.MustCompile(`"([a-zA-Z0-9_:@/.-]+)"\s*:\s*"([^"]*)"`)

func regexQuotedPairs(s string) [][]string {
	ms := quotedPairRe.FindAllStringSubmatch(s, -1)
	out := [][]string{}
	for _, m := range ms {
		out = append(out, []string{m[1], m[2]})
	}
	return out
}

// detectOutputDir resolves the build output directory, respecting custom
// build.outDir in vite.config when present.
func detectOutputDir(srcDir string, p *Plan, read func(string) string) string {
	// custom outDir from vite.config
	for _, name := range []string{"vite.config.ts", "vite.config.js", "vite.config.mjs"} {
		cfg := read(name)
		if cfg != "" {
			m := regexp.MustCompile(`outDir\s*:\s*["']([^"']+)["']`).FindStringSubmatch(cfg)
			if len(m) > 1 {
				return strings.TrimPrefix(m[1], "./")
			}
		}
	}
	switch p.Framework {
	case "Astro":
		return "dist"
	case "Vite", "Solid", "Preact", "Lit", "Qwik":
		return "dist"
	case "React (CRA/Webpack)":
		return "build"
	case "Angular":
		// angular.json outputPath (best effort)
		cfg := read("angular.json")
		m := regexp.MustCompile(`"outputPath"\s*:\s*"([^"]+)"`).FindStringSubmatch(cfg)
		if len(m) > 1 {
			return m[1]
		}
		return "dist"
	case "Next.js":
		if strings.Contains(read("next.config.js")+read("next.config.ts"), "output: 'export'") {
			return "out"
		}
		return "out"
	case "Nuxt":
		return ".output/public"
	case "Gatsby", "Docusaurus":
		return "public"
	case "VitePress":
		return ".vitepress/dist"
	case "Eleventy":
		return "_site"
	case "SvelteKit":
		return "build"
	}
	// structure sniff
	for _, d := range []string{"dist", "build", "out", "public", ".output/public"} {
		if fi, err := os.Stat(filepath.Join(srcDir, d)); err == nil && fi.IsDir() {
			return d
		}
	}
	return "dist"
}

// detectRouting classifies the app type.
func detectRouting(p *Plan) {
	ssr := p.Framework == "Next.js" || p.Framework == "Nuxt" || p.Framework == "Remix" ||
		p.Framework == "SvelteKit" || p.Framework == "TanStack Start" || p.Framework == "Fresh"
	if ssr {
		p.AppType = TypeSSR
		p.SPAFallback = false
		p.Warnings = append(p.Warnings, "Server Side Rendering detected — the Node server renders and routes all traffic")
		return
	}
	switch p.Framework {
	case "Astro", "Gatsby", "Docusaurus", "VuePress", "VitePress", "Eleventy":
		p.AppType = TypeSSG
		p.SPAFallback = false
	case "Vite", "React (CRA/Webpack)", "Vue", "Svelte", "Solid", "Preact", "Lit", "Qwik", "Angular", "Webpack", "Rspack":
		p.AppType = TypeSPA
		p.SPAFallback = true
	default:
		// check for index.html → static; else unknown
		p.AppType = TypeStatic
		p.SPAFallback = false
	}
}

func discoverIndex(srcDir string, p *Plan) {
	for _, d := range []string{p.OutputDir, "dist", "build", "out", "public", "."} {
		if fi, err := os.Stat(filepath.Join(srcDir, d)); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(srcDir, d, "index.html")); err == nil {
				p.IndexFile = "index.html"
				return
			}
			// Angular style dist/<project>
			entries, _ := os.ReadDir(filepath.Join(srcDir, d))
			for _, e := range entries {
				if e.IsDir() {
					if _, err := os.Stat(filepath.Join(srcDir, d, e.Name(), "index.html")); err == nil {
						p.IndexFile = "index.html"
						p.OutputDir = d + "/" + e.Name()
						return
					}
				}
			}
		}
	}
	if p.AppType == TypeSPA {
		p.Warnings = append(p.Warnings, "index.html not found no output esperado — verifique o output directory")
	}
}
