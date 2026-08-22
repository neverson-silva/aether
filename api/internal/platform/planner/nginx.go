package planner

import "strings"

// GenerateNginxConf produces an optimized nginx.conf for the plan.
func GenerateNginxConf(p *Plan) string {
	root := "/usr/share/nginx/html"
	index := p.IndexFile
	if index == "" {
		index = "index.html"
	}
	var sb strings.Builder
	sb.WriteString("server {\n")
	sb.WriteString("  listen 80;\n")
	sb.WriteString("  server_name _;\n")
	sb.WriteString("  root " + root + ";\n")
	sb.WriteString("  index " + index + ";\n\n")

	sb.WriteString("  gzip on;\n")
	sb.WriteString("  gzip_vary on;\n")
	sb.WriteString("  gzip_min_length 1024;\n")
	sb.WriteString("  gzip_types text/plain text/css application/json application/javascript application/xml image/svg+xml;\n\n")

	sb.WriteString("  location ~* \\.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?|ttf|eot)$ {\n")
	sb.WriteString("    expires 1y;\n")
	sb.WriteString("    add_header Cache-Control \"public, immutable\";\n")
	sb.WriteString("    try_files $uri =404;\n")
	sb.WriteString("  }\n\n")

	// nunca servir .env* nem dotfiles
	sb.WriteString("  location ~ /\\.env { deny all; }\n")
	sb.WriteString("  location ~ /\\.(?!well-known) { deny all; }\n\n")

	sb.WriteString("  add_header X-Content-Type-Options nosniff;\n")
	sb.WriteString("  add_header X-Frame-Options SAMEORIGIN;\n")
	sb.WriteString("  add_header Referrer-Policy strict-origin-when-cross-origin;\n")
	sb.WriteString("  etag on;\n\n")

	if p.SPAFallback {
		sb.WriteString("  location / {\n")
		sb.WriteString("    try_files $uri $uri/ /" + index + ";\n")
		sb.WriteString("  }\n")
	} else {
		sb.WriteString("  location / {\n")
		sb.WriteString("    try_files $uri $uri/ =404;\n")
		sb.WriteString("  }\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "80"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		return "0"
	}
	return s
}
