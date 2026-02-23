#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_SCHEMA="$ROOT/shared/schemas/skill.schema.json"
REGISTRY_SCHEMA="$ROOT/shared/schemas/registry-index.schema.json"

if ! command -v check-jsonschema >/dev/null 2>&1; then
  echo "[ERROR] check-jsonschema is required. Install via: pip install check-jsonschema"
  exit 1
fi

mapfile -t manifests < <(find "$ROOT/skills" -mindepth 3 -maxdepth 3 -name "skill.yaml" | sort)
if [[ "${#manifests[@]}" -eq 0 ]]; then
  echo "[WARN] No skill.yaml manifests found under skills/. Skipping skill manifest validation."
else
  echo "[check] validating ${#manifests[@]} skill manifest(s)"
  check-jsonschema --schemafile "$SKILL_SCHEMA" "${manifests[@]}"
fi

if [[ -f "$ROOT/registry/index.json" ]]; then
  echo "[check] validating registry/index.json"
  check-jsonschema --schemafile "$REGISTRY_SCHEMA" "$ROOT/registry/index.json"
else
  echo "[info] registry/index.json not found, skipping registry index validation."
fi

echo "Manifest schema validation completed."
