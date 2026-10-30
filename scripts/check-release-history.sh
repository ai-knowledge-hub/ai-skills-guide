#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: bash scripts/check-release-history.sh <base-sha> <head-sha>"
  exit 2
fi

base_sha="$1"
head_sha="$2"
zero_sha="0000000000000000000000000000000000000000"

git cat-file -e "${head_sha}^{commit}"
if [[ "$base_sha" == "$zero_sha" ]]; then
  # The first push of a branch has no preceding commit. Comparing with the
  # empty tree correctly treats every release in that push as an addition.
  base_sha="$(git hash-object -t tree /dev/null)"
else
  git cat-file -e "${base_sha}^{commit}"
fi

changes="$(git diff --name-status --no-renames "$base_sha" "$head_sha" -- \
  releases/skills releases/agents releases/tools-mcp releases/plugins)"
violations="$(printf '%s\n' "$changes" | awk 'NF > 0 && $1 != "A"')"

if [[ -n "$violations" ]]; then
  echo "Published release files are append-only; create a new version."
  printf '%s\n' "$violations"
  exit 1
fi
