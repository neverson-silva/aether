#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
catalog_directory="${repository_root}/api/internal/modules/templates/infra/catalogdata"
blueprint_directory="${catalog_directory}/blueprints"
metadata_url="https://templates.dokploy.com/meta.json"

command -v curl >/dev/null 2>&1 || { printf '%s\n' 'curl is required' >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { printf '%s\n' 'jq is required' >&2; exit 1; }

mkdir -p "${blueprint_directory}"
curl --fail --location --silent --show-error --max-time 30 "${metadata_url}" -o "${catalog_directory}/meta.json.tmp"
jq -e 'type == "array" and length > 0' "${catalog_directory}/meta.json.tmp" >/dev/null
mv "${catalog_directory}/meta.json.tmp" "${catalog_directory}/meta.json"

jq -r '.[] | [.id, .logo] | @tsv' "${catalog_directory}/meta.json" | while IFS=$'\t' read -r template_id logo; do
  case "${template_id}" in
    ""|*[!A-Za-z0-9._-]*) continue ;;
  esac
  curl --fail --location --silent --show-error --max-time 30 "https://templates.dokploy.com/blueprints/${template_id}/docker-compose.yml" -o "${blueprint_directory}/${template_id}.docker-compose.yml.tmp"
  test -s "${blueprint_directory}/${template_id}.docker-compose.yml.tmp"
  mv "${blueprint_directory}/${template_id}.docker-compose.yml.tmp" "${blueprint_directory}/${template_id}.docker-compose.yml"
  curl --fail --location --silent --show-error --max-time 30 "https://templates.dokploy.com/blueprints/${template_id}/template.toml" -o "${blueprint_directory}/${template_id}.template.toml.tmp"
  test -s "${blueprint_directory}/${template_id}.template.toml.tmp"
  mv "${blueprint_directory}/${template_id}.template.toml.tmp" "${blueprint_directory}/${template_id}.template.toml"
  case "${logo}" in
    ""|*/*|*\\*) continue ;;
  esac
  curl --fail --location --silent --show-error --max-time 30 "https://templates.dokploy.com/blueprints/${template_id}/${logo}" -o "${blueprint_directory}/${template_id}-logo.tmp"
  test -s "${blueprint_directory}/${template_id}-logo.tmp"
  mv "${blueprint_directory}/${template_id}-logo.tmp" "${blueprint_directory}/${template_id}-logo"
done

find "${catalog_directory}" -type f -name '*.tmp' -delete
printf '%s\n' 'Dokploy templates synchronized into the repository'
