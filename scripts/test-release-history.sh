#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECKER="$ROOT/scripts/check-release-history.sh"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT

configure_repository() {
  local repository="$1"
  git -C "$repository" init -q
  git -C "$repository" config user.name "Release History Test"
  git -C "$repository" config user.email "release-history-test@example.invalid"
}

mutation_repository="$TEST_ROOT/mutation"
mkdir -p "$mutation_repository/releases/skills/example/package/1.0.0"
configure_repository "$mutation_repository"
printf 'original\n' > "$mutation_repository/releases/skills/example/package/1.0.0/SKILL.md"
git -C "$mutation_repository" add .
git -C "$mutation_repository" commit -qm "publish release"
mutation_base="$(git -C "$mutation_repository" rev-parse HEAD)"

printf 'mutated\n' > "$mutation_repository/releases/skills/example/package/1.0.0/SKILL.md"
git -C "$mutation_repository" commit -qam "mutate release"
printf 'unrelated\n' > "$mutation_repository/README.md"
git -C "$mutation_repository" add README.md
git -C "$mutation_repository" commit -qm "unrelated final commit"

if (cd "$mutation_repository" && bash "$CHECKER" "$mutation_base" HEAD >/dev/null); then
  echo "[ERROR] full-range check accepted an earlier release mutation"
  exit 1
fi

addition_repository="$TEST_ROOT/addition"
mkdir -p "$addition_repository"
configure_repository "$addition_repository"
printf 'catalog\n' > "$addition_repository/README.md"
git -C "$addition_repository" add README.md
git -C "$addition_repository" commit -qm "initial catalog"
addition_base="$(git -C "$addition_repository" rev-parse HEAD)"
mkdir -p "$addition_repository/releases/skills/example/package/1.0.0"
printf 'new release\n' > "$addition_repository/releases/skills/example/package/1.0.0/SKILL.md"
git -C "$addition_repository" add .
git -C "$addition_repository" commit -qm "append release"
(cd "$addition_repository" && bash "$CHECKER" "$addition_base" HEAD)

echo "Release history range checks passed."
