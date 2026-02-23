#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CODEX_SKILLS_DIR="${CODEX_HOME:-$HOME/.codex}/skills"
TARGET="$CODEX_SKILLS_DIR/playwright-vscode-loop-codex"

mkdir -p "$CODEX_SKILLS_DIR"
rm -rf "$TARGET"
cp -R "$ROOT" "$TARGET"

echo "Installed skill to $TARGET"
