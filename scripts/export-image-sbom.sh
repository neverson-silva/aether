#!/usr/bin/env bash
set -euo pipefail

command -v docker >/dev/null 2>&1 || { printf '%s\n' 'docker is required' >&2; exit 1; }
docker sbom --help 2>&1 | rg -q -- '--format' || { printf '%s\n' 'Docker SBOM plugin with SPDX output is required' >&2; exit 1; }

if (( $# == 0 )); then
  printf '%s\n' 'at least one image reference is required' >&2
  exit 1
fi

output_directory="${AETHER_SBOM_DIR:-artifacts/sbom}"
mkdir -p "${output_directory}"

for image in "$@"; do
  name="$(printf '%s' "${image}" | tr '/:@' '___')"
  docker sbom --format spdx-json "${image}" > "${output_directory}/${name}.spdx.json"
done

printf '%s\n' "SBOM artifacts exported to ${output_directory}"
