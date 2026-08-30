#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
legacy_runtime="pod""man"
legacy_compose="pod""man-compose"
legacy_socket="pod""man.sock"
legacy_machine="gv""proxy"
active_paths=(api infra install.sh install-dev.sh dev.sh uninstall.sh README.md docs)
failed=0

for term in "$legacy_runtime" "$legacy_compose" "$legacy_socket" "$legacy_machine"; do
	if matches=$(rg -n -i "$term" "${active_paths[@]}" --glob '!specs/**' --glob '!AGENTS.md' --glob '!scripts/validate-docker-cutover.sh' 2>/dev/null); then
		printf 'forbidden legacy runtime reference (%s):\n%s\n' "$term" "$matches" >&2
		failed=1
	fi
done

if matches=$(rg -n 'os/exec|exec\.Command' api/internal/modules api/cmd --glob '*.go' --glob '!**/*_test.go' 2>/dev/null); then
	printf 'direct process execution in feature packages:\n%s\n' "$matches" >&2
	failed=1
fi

if ! command -v docker >/dev/null 2>&1; then
	printf '%s\n' 'Docker CLI is required' >&2
	failed=1
fi

if command -v docker >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
	printf '%s\n' 'Docker Compose is required' >&2
	failed=1
fi

if (( failed != 0 )); then
	exit 1
fi

printf '%s\n' 'Docker cutover validation passed'
