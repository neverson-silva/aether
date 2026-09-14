#!/usr/bin/env bash
set -euo pipefail

failed=0

if rg -n 'image:[[:space:]]*["'"'"']?[^[:space:]"'"'"']*:latest(["'"'"'[:space:]]|$)|"image":[[:space:]]*"[^"]*:latest(["'"'"'[:space:]]|$)' api/db/migrations infra api/internal/modules/templates/infra/catalogdata install-dev.sh dev.sh .env.example --glob '*.sql' --glob '*.yml' --glob '*.yaml' --glob '*.sh' --glob '*.env' 2>/dev/null; then
  printf '%s\n' 'mutable :latest image references are forbidden' >&2
  failed=1
fi

for lockfile in api/go.sum frontend/web/package-lock.json frontend/aether_ds/package-lock.json; do
  if [[ ! -s "$lockfile" ]]; then
    printf 'required dependency lockfile is missing or empty: %s\n' "$lockfile" >&2
    failed=1
  fi
done

if [[ ! -s api/go.mod ]]; then
  printf '%s\n' 'Go module manifest is missing or empty' >&2
  failed=1
fi

if ! rg -q 'COPY --from=build /out/bin/cosign /usr/local/bin/cosign' infra/Dockerfile; then
  printf '%s\n' 'production image must contain the pinned cosign verifier' >&2
  failed=1
fi

if rg -n 'RUN[[:space:]]+npm[[:space:]]+install([[:space:]]|$)' infra --glob '*Dockerfile' >/dev/null 2>&1; then
  printf '%s\n' 'Dockerfile dependency installation must use a lockfile-aware command' >&2
  failed=1
fi

if [[ "${AETHER_REQUIRE_IMAGE_SIGNATURES:-false}" == "true" ]]; then
  if ! command -v cosign >/dev/null 2>&1; then
    printf '%s\n' 'cosign is required when AETHER_REQUIRE_IMAGE_SIGNATURES=true' >&2
    failed=1
  fi
  if [[ -z "${AETHER_COSIGN_PUBLIC_KEY:-}" || ! -f "${AETHER_COSIGN_PUBLIC_KEY}" ]]; then
    printf '%s\n' 'AETHER_COSIGN_PUBLIC_KEY must point to a readable verification key' >&2
    failed=1
  fi
fi

if [[ "${AETHER_REQUIRE_IMAGE_DIGESTS:-false}" == "true" ]] && ! rg -q 'AETHER_REQUIRE_IMAGE_DIGESTS' api/internal/platform/worker/docker_runtime.go; then
  printf '%s\n' 'digest enforcement is required when AETHER_REQUIRE_IMAGE_DIGESTS=true' >&2
  failed=1
fi

if (( failed != 0 )); then
  exit 1
fi

printf '%s\n' 'Supply-chain validation passed'
