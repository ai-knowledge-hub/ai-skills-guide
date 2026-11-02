#!/usr/bin/env bash
set -euo pipefail

release_tag="${1:-}"
script_dir="$(cd "$(dirname "$0")" && pwd)"
bash "$script_dir/verify-cli-release-source.sh" "$release_tag"

# An explicit dispatch is required after a GITHUB_TOKEN-authenticated tag push.
# Keep the workflow execution on the tag so its attestation binds refs/tags/….
gh workflow run release-on-tag.yml \
  --repo "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}" \
  --ref "$release_tag"
