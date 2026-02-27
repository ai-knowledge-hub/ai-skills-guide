#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"
RUN_E2E="${RUN_E2E:-0}"

if [[ -z "$VERSION" ]]; then
  echo "Usage: bash scripts/release-cut.sh vX.Y.Z[-alpha.N]"
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+\.[0-9]+)?$ ]]; then
  echo "[ERROR] Version must match vX.Y.Z or vX.Y.Z-alpha.N"
  exit 1
fi

cd "$ROOT"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "[ERROR] Working tree is not clean. Commit or stash changes first."
  exit 1
fi

current_branch="$(git branch --show-current)"
if [[ "$current_branch" != "main" ]]; then
  echo "[ERROR] Release cut must run from main. Current branch: $current_branch"
  exit 1
fi

echo "[check] syncing main"
git fetch origin
git pull --ff-only origin main

if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
  echo "[ERROR] Tag $VERSION already exists locally."
  exit 1
fi

if git ls-remote --tags origin | grep -q "refs/tags/$VERSION$"; then
  echo "[ERROR] Tag $VERSION already exists on origin."
  exit 1
fi

echo "[check] validate skills"
bash scripts/validate-skills.sh

echo "[check] rebuild registry"
go run ./cmd/registry-builder

if [[ -n "$(git status --porcelain)" ]]; then
  echo "[ERROR] Generated files changed (for example registry/index.json). Commit before release."
  git status --short
  exit 1
fi

echo "[check] web lint/build"
(
  cd apps/web
  pnpm install --frozen-lockfile
  pnpm lint
  pnpm build

  if [[ "$RUN_E2E" == "1" ]]; then
    pnpm test:e2e
  fi
)

echo "[release] creating and pushing tag $VERSION"
git tag "$VERSION"
git push origin "$VERSION"

echo "[done] Tag pushed: $VERSION"
echo "GitHub Actions will publish the release via .github/workflows/release-on-tag.yml"
