#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAIL=0

validate_module_structure() {
  local module_root="$1"
  local label="$2"
  local spec_file="$3"
  local manifest_file="$4"

  if [[ ! -d "$module_root" ]]; then
    echo "[info] Skipping $label validation; ${module_root#$ROOT/}/ does not exist."
    return 0
  fi

  while IFS= read -r -d '' entry_dir; do
    required=("$spec_file" "README.md" "$manifest_file" "tests/test-prompts.md")
    for f in "${required[@]}"; do
      if [[ ! -f "$entry_dir/$f" ]]; then
        echo "[ERROR] Missing $f in ${entry_dir#$ROOT/}"
        FAIL=1
      fi
    done

    if [[ ! -d "$entry_dir/examples" ]]; then
      echo "[ERROR] Missing examples/ in ${entry_dir#$ROOT/}"
      FAIL=1
    fi

    if [[ -f "$entry_dir/$spec_file" ]]; then
      if [[ "$spec_file" == "SKILL.md" ]] && ! grep -q "^---" "$entry_dir/$spec_file"; then
        echo "[ERROR] Missing frontmatter in ${entry_dir#$ROOT/}/$spec_file"
        FAIL=1
      fi
      if [[ "$spec_file" == "SKILL.md" ]] && ! grep -qi "## When to use" "$entry_dir/$spec_file"; then
        echo "[ERROR] Missing 'When to use' section in ${entry_dir#$ROOT/}/$spec_file"
        FAIL=1
      fi
      if ! grep -qi "## Guardrails" "$entry_dir/$spec_file"; then
        echo "[ERROR] Missing 'Guardrails' section in ${entry_dir#$ROOT/}/$spec_file"
        FAIL=1
      fi
    fi

    prompt_file="$entry_dir/tests/test-prompts.md"
    if [[ -f "$prompt_file" ]]; then
      prompt_count="$(grep -Ec '^[0-9]+\.' "$prompt_file" || true)"
      if [[ "$prompt_count" -lt 5 ]]; then
        echo "[ERROR] Need at least 5 numbered prompts in ${prompt_file#$ROOT/} (found $prompt_count)"
        FAIL=1
      fi
    fi
  done < <(find "$module_root" -mindepth 2 -maxdepth 2 -type d -print0)
}

validate_module_structure "$ROOT/skills" "skills" "SKILL.md" "skill.yaml"
validate_module_structure "$ROOT/agents" "agents" "AGENT.md" "agent.yaml"
validate_module_structure "$ROOT/tools-mcp" "tools-mcp" "TOOL.md" "tool.yaml"

if [[ "$FAIL" -eq 0 ]]; then
  echo "All module entries passed structural validation."
else
  exit 1
fi
