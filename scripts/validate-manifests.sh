#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_SCHEMA="$ROOT/shared/schemas/skill.schema.json"
AGENT_SCHEMA="$ROOT/shared/schemas/agent.schema.json"
TOOL_SCHEMA="$ROOT/shared/schemas/tool.schema.json"
PLUGIN_SCHEMA="$ROOT/shared/schemas/plugin.schema.json"
REGISTRY_SCHEMA="$ROOT/shared/schemas/registry-index.schema.json"
CONTRACT_SCHEMA="$ROOT/shared/schemas/manifest-contract-v2.schema.json"
CONTRACT_FIXTURES="$ROOT/shared/schemas/fixtures/manifest-v2"
CONTRACT_MATRIX_SCHEMA="$CONTRACT_FIXTURES/contract-matrix.schema.json"

CHECK_JSONSCHEMA=""
if command -v check-jsonschema >/dev/null 2>&1; then
  CHECK_JSONSCHEMA="$(command -v check-jsonschema)"
elif [[ -x "$ROOT/.venv/bin/check-jsonschema" ]]; then
  CHECK_JSONSCHEMA="$ROOT/.venv/bin/check-jsonschema"
fi

if [[ -z "$CHECK_JSONSCHEMA" ]]; then
  echo "[ERROR] check-jsonschema is required. Install via: python3 -m venv .venv && .venv/bin/pip install check-jsonschema"
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

  manifests=()
  while IFS= read -r manifest_path; do
    manifests+=("$manifest_path")
  done < <(find "$module_dir" -mindepth 3 -maxdepth 3 -name "$manifest_name" | sort)
  if [[ "${#manifests[@]}" -eq 0 ]]; then
    echo "[WARN] No $manifest_name manifests found under ${module_dir#$ROOT/}/. Skipping $label manifest validation."
    return 0
  fi

  echo "[check] validating ${#manifests[@]} $label manifest(s)"
  "$CHECK_JSONSCHEMA" --schemafile "$schema_path" "${manifests[@]}"
}

validate_module_manifests "$ROOT/skills" "skill.yaml" "$SKILL_SCHEMA" "skill"
validate_module_manifests "$ROOT/agents" "agent.yaml" "$AGENT_SCHEMA" "agent"
validate_module_manifests "$ROOT/tools-mcp" "tool.yaml" "$TOOL_SCHEMA" "tool"
validate_module_manifests "$ROOT/plugins" "plugin.yaml" "$PLUGIN_SCHEMA" "plugin"

echo "[check] validating semantic manifest invariants"
go run ./cmd/manifest-validator "$ROOT"

echo "[check] validating manifest contract v2 golden fixtures"
"$CHECK_JSONSCHEMA" --check-metaschema "$CONTRACT_SCHEMA" "$CONTRACT_MATRIX_SCHEMA"
"$CHECK_JSONSCHEMA" --schemafile "$CONTRACT_MATRIX_SCHEMA" "$CONTRACT_FIXTURES/contract-matrix.valid.json"
"$CHECK_JSONSCHEMA" --schemafile "$SKILL_SCHEMA" "$CONTRACT_FIXTURES/skill.valid.yaml"
"$CHECK_JSONSCHEMA" --schemafile "$AGENT_SCHEMA" "$CONTRACT_FIXTURES/agent.valid.yaml"
"$CHECK_JSONSCHEMA" --schemafile "$TOOL_SCHEMA" "$CONTRACT_FIXTURES/tool.valid.yaml"
"$CHECK_JSONSCHEMA" --schemafile "$TOOL_SCHEMA" "$CONTRACT_FIXTURES/tool-flow-sequences.valid.yaml"
"$CHECK_JSONSCHEMA" --schemafile "$TOOL_SCHEMA" "$CONTRACT_FIXTURES/tool-secret-in-allowed-field.semantic-invalid.yaml"
"$CHECK_JSONSCHEMA" --schemafile "$PLUGIN_SCHEMA" "$CONTRACT_FIXTURES/plugin.valid.yaml"

for fixture_path in "$CONTRACT_FIXTURES"/*.invalid.json; do
  if "$CHECK_JSONSCHEMA" --schemafile "$CONTRACT_MATRIX_SCHEMA" "$fixture_path" >/dev/null 2>&1; then
    echo "[ERROR] Expected schema rejection: ${fixture_path#$ROOT/}"
    exit 1
  fi
done

if "$CHECK_JSONSCHEMA" --schemafile "$SKILL_SCHEMA" "$CONTRACT_FIXTURES/skill-partial.invalid.yaml" >/dev/null 2>&1; then
  echo "[ERROR] Expected schema rejection: shared/schemas/fixtures/manifest-v2/skill-partial.invalid.yaml"
  exit 1
fi

if "$CHECK_JSONSCHEMA" --schemafile "$SKILL_SCHEMA" "$CONTRACT_FIXTURES/skill-unversioned-v2.invalid.yaml" >/dev/null 2>&1; then
  echo "[ERROR] Expected schema rejection: shared/schemas/fixtures/manifest-v2/skill-unversioned-v2.invalid.yaml"
  exit 1
fi

for index_path in \
  "$ROOT/registry/index.json" \
  "$ROOT/registry/skills-index.json" \
  "$ROOT/registry/agents-index.json" \
  "$ROOT/registry/tools-index.json" \
  "$ROOT/registry/plugins-index.json"
do
  if [[ -f "$index_path" ]]; then
    echo "[check] validating ${index_path#$ROOT/}"
    "$CHECK_JSONSCHEMA" --schemafile "$REGISTRY_SCHEMA" "$index_path"
  fi
done

echo "Manifest schema validation completed."
