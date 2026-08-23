package planner

import "strings"

// GenerateDockerfile produces a multi-stage Dockerfile.
// For SSR apps it produces a Node runtime; otherwise a static nginx runtime.
func GenerateDockerfile(p *Plan) string {
	if p.AppType == TypeSSR {
		return generateNodeDockerfile(p)
	}
	buildImage := "node:22-alpine"
	if p.PackageManager == "bun" {
		buildImage = "oven/bun:1"
	}
	install := installCommand(p)
	output := p.OutputDir
	if output == "" {
		output = "dist"
	}
	build := p.BuildCommand
	if build == "" {
		build = p.PackageManager + " run build"
	}

	var sb strings.Builder
	sb.WriteString("# syntax=docker/dockerfile:1\n")
	sb.WriteString("FROM " + buildImage + " AS build\n")
	sb.WriteString("WORKDIR /app\n")
	if p.PackageManager == "bun" {
		sb.WriteString("COPY package.json bun.lock* ./\n")
	} else {
		if p.HasLockfile {
			sb.WriteString("COPY package.json package-lock.json ./\n")
		} else {
			sb.WriteString("COPY package*.json ./\n")
		}
	}
	sb.WriteString("RUN " + install + "\n")
	sb.WriteString("COPY . .\n")
	sb.WriteString("RUN " + build + "\n")
	sb.WriteString("\nFROM nginx:alpine\n")
	sb.WriteString("COPY nginx.conf /etc/nginx/conf.d/default.conf\n")
	sb.WriteString("COPY --from=build /app/" + strings.TrimPrefix(output, "./") + " /usr/share/nginx/html\n")
	sb.WriteString("EXPOSE 80\n")
	sb.WriteString("CMD [\"nginx\", \"-g\", \"daemon off;\"]\n")
	return sb.String()
}

func generateNodeDockerfile(p *Plan) string {
	buildImage := "node:22-alpine"
	if p.PackageManager == "bun" {
		buildImage = "oven/bun:1"
	}
	install := installCommand(p)
	build := p.BuildCommand
	if build == "" {
		build = p.PackageManager + " run build"
	}
	start := startCommand(p)
	port := p.ContainerPort
	if port == 0 {
		port = 3000
	}

	var sb strings.Builder
	sb.WriteString("# syntax=docker/dockerfile:1\n")
	sb.WriteString("FROM " + buildImage + " AS build\n")
	sb.WriteString("WORKDIR /app\n")
	if p.PackageManager == "bun" {
		sb.WriteString("COPY package.json bun.lock* ./\n")
	} else if p.PackageManager == "pnpm" {
		sb.WriteString("COPY package.json pnpm-lock.yaml ./\n")
	} else if p.PackageManager == "yarn" {
		sb.WriteString("COPY package.json yarn.lock ./\n")
	} else {
		sb.WriteString("COPY package*.json ./\n")
	}
	sb.WriteString("RUN " + install + "\n")
	sb.WriteString("COPY . .\n")
	sb.WriteString("RUN " + build + "\n")
	sb.WriteString("\nFROM " + buildImage + " AS runtime\n")
	sb.WriteString("WORKDIR /app\n")
	sb.WriteString("ENV NODE_ENV=production\n")
	sb.WriteString("ENV PORT=" + itoa(port) + "\n")
	sb.WriteString("COPY --from=build /app/package*.json ./\n")
	sb.WriteString("RUN " + productionInstall(p) + "\n")
	sb.WriteString("COPY --from=build /app/ ./\n")
	sb.WriteString("EXPOSE " + itoa(port) + "\n")
	sb.WriteString("CMD [\"sh\", \"-c\", " + shellQuote(start) + "]\n")
	return sb.String()
}

func startCommand(p *Plan) string {
	switch p.Framework {
	case "Next.js":
		return "next start"
	case "Nuxt":
		return "node .output/server/index.mjs"
	case "Analog.js":
		return "node dist/server/index.mjs"
	case "NestJS":
		return "npm run start:prod"
	case "Remix":
		return "npm run start"
	case "SvelteKit":
		return "node build"
	case "TanStack Start":
		return "node dist/server/server.js"
	case "Fresh":
		return "deno run -A main.ts"
	default:
		return "npm start"
	}
}

func productionInstall(p *Plan) string {
	switch p.PackageManager {
	case "bun":
		return "bun install --production"
	case "pnpm":
		return "pnpm install --prod"
	case "yarn":
		return "yarn install --production"
	default:
		return "npm install --omit=dev --no-audit --no-fund || npm install --omit=dev --legacy-peer-deps --no-audit --no-fund"
	}
}
func shellQuote(s string) string {
	return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
}

func installCommand(p *Plan) string {
	switch p.PackageManager {
	case "bun":
		return "bun install"
	case "pnpm":
		return "pnpm install --frozen-lockfile"
	case "yarn":
		return "yarn install --frozen-lockfile"
	default:
		if p.HasLockfile {
			return "npm ci"
		}
		// sem lockfile: fallback --legacy-peer-deps para projetos com deps
		// com peer conflicts (ex: "latest" + sem lockfile)
		return "npm install --no-audit --no-fund || npm install --legacy-peer-deps --no-audit --no-fund"
	}
}
