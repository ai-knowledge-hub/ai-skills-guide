#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
output_dir="${2:-dist}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Usage: $0 vX.Y.Z[-prerelease] [output-directory]" >&2
  exit 1
fi

tagged_commit="$(git rev-list -n 1 "$version" 2>/dev/null || true)"
head_commit="$(git rev-parse HEAD)"
if [[ -z "$tagged_commit" || "$tagged_commit" != "$head_commit" ]]; then
  echo "Refusing release build: HEAD is not the exact $version commit" >&2
  exit 1
fi
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "Refusing release build: the tagged checkout is not clean" >&2
  exit 1
fi

go run ./cmd/runtime-trust-validator \
  --codex "${CODEX_MODEL_ATTESTATION_PUBLIC_KEY:-}" \
  --claude "${CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY:-}" \
  --generic "${GENERIC_MODEL_ATTESTATION_PUBLIC_KEY:-}"

if [[ -e "$output_dir" ]] && [[ -n "$(find "$output_dir" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  echo "Refusing to overwrite non-empty output directory: $output_dir" >&2
  exit 1
fi
mkdir -p "$output_dir"

ldflags="-s -w -buildid= -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.codexModelAttestationPublicKey=${CODEX_MODEL_ATTESTATION_PUBLIC_KEY} -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.claudeModelAttestationPublicKey=${CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY} -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.genericModelAttestationPublicKey=${GENERIC_MODEL_ATTESTATION_PUBLIC_KEY}"
targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64)

for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  suffix=""
  if [[ "$os" == "windows" ]]; then
    suffix=".exe"
  fi
  artifact="$output_dir/skills-hub_${version}_${os}_${arch}${suffix}"
  echo "Building $artifact"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$ldflags" -o "$artifact" ./cmd/skills-hub
done

(
  cd "$output_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum skills-hub_* > SHA256SUMS
  else
    shasum -a 256 skills-hub_* > SHA256SUMS
  fi
)
