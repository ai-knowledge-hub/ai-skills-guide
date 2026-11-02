#!/usr/bin/env bash
set -euo pipefail

release_tag="${1:-}"
expected_commit="${2:-$(git rev-parse HEAD)}"
if [[ ! "$release_tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "Refusing CLI release: an exact vX.Y.Z[-prerelease] tag is required" >&2
  exit 1
fi

tagged_commit="$(git rev-parse --verify "refs/tags/${release_tag}^{commit}")"
head_commit="$(git rev-parse HEAD)"
if [[ "$tagged_commit" != "$expected_commit" || "$head_commit" != "$tagged_commit" ]]; then
  echo "Refusing CLI release: tag, expected source and HEAD must identify the same commit" >&2
  exit 1
fi
if ! git merge-base --is-ancestor "$tagged_commit" refs/remotes/origin/main; then
  echo "Refusing CLI release: tagged source must already be integrated into origin/main" >&2
  exit 1
fi
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "Refusing CLI release: tagged checkout is not clean" >&2
  exit 1
fi
echo "CLI release source verified: $release_tag at $tagged_commit"
