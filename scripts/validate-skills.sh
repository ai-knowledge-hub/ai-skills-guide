#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAIL=0

while IFS= read -r -d '' skill_dir; do
  required=("SKILL.md" "README.md" "tests/test-prompts.md")
  for f in "${required[@]}"; do
    if [[ ! -f "$skill_dir/$f" ]]; then
      echo "[ERROR] Missing $f in ${skill_dir#$ROOT/}"
      FAIL=1
    fi
  done

  if [[ -f "$skill_dir/SKILL.md" ]]; then
    if ! grep -q "^---" "$skill_dir/SKILL.md"; then
      echo "[ERROR] Missing frontmatter in ${skill_dir#$ROOT/}/SKILL.md"
      FAIL=1
    fi
    if ! grep -qi "## When to use" "$skill_dir/SKILL.md"; then
      echo "[ERROR] Missing 'When to use' section in ${skill_dir#$ROOT/}/SKILL.md"
      FAIL=1
    fi
    if ! grep -qi "## Guardrails" "$skill_dir/SKILL.md"; then
      echo "[ERROR] Missing 'Guardrails' section in ${skill_dir#$ROOT/}/SKILL.md"
      FAIL=1
    fi
  fi

done < <(find "$ROOT/skills" -mindepth 2 -maxdepth 2 -type d -print0)

if [[ "$FAIL" -eq 0 ]]; then
  echo "All skills passed structural validation."
else
  exit 1
fi
