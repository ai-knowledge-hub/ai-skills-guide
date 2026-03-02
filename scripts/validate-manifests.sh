#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_SCHEMA="$ROOT/shared/schemas/skill.schema.json"
AGENT_SCHEMA="$ROOT/shared/schemas/agent.schema.json"
TOOL_SCHEMA="$ROOT/shared/schemas/tool.schema.json"
REGISTRY_SCHEMA="$ROOT/shared/schemas/registry-index.schema.json"

if ! command -v check-jsonschema >/dev/null 2>&1; then
  echo "[ERROR] check-jsonschema is required. Install via: pip install check-jsonschema"
  exit 1
fi

validate_module_manifests() {
  local module_dir="$1"
  local manifest_name="$2"
  local schema_path="$3"
  local label="$4"

  if [[ ! -d "$module_dir" ]]; then
    echo "[info] ${module_dir#$ROOT/}/ not found, skipping $label manifest validation."
    return 0
  fi

  mapfile -t manifests < <(find "$module_dir" -mindepth 3 -maxdepth 3 -name "$manifest_name" | sort)
  if [[ "${#manifests[@]}" -eq 0 ]]; then
    echo "[WARN] No $manifest_name manifests found under ${module_dir#$ROOT/}/. Skipping $label manifest validation."
    return 0
  fi

  echo "[check] validating ${#manifests[@]} $label manifest(s)"
  check-jsonschema --schemafile "$schema_path" "${manifests[@]}"
}

validate_module_manifests "$ROOT/skills" "skill.yaml" "$SKILL_SCHEMA" "skill"
validate_module_manifests "$ROOT/agents" "agent.yaml" "$AGENT_SCHEMA" "agent"
validate_module_manifests "$ROOT/tools-mcp" "tool.yaml" "$TOOL_SCHEMA" "tool"

for index_path in \
  "$ROOT/registry/index.json" \
  "$ROOT/registry/skills-index.json" \
  "$ROOT/registry/agents-index.json" \
  "$ROOT/registry/tools-index.json"
do
  if [[ -f "$index_path" ]]; then
    echo "[check] validating ${index_path#$ROOT/}"
    check-jsonschema --schemafile "$REGISTRY_SCHEMA" "$index_path"
  fi
done

echo "Manifest schema validation completed."
