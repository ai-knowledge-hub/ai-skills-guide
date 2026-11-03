SHELL := /bin/bash

.PHONY: help doctor validate manifests registry release-assets-check test-scripts test ci-local cli-build cli-build-test cli-build-runtime-dev cli-runtime-test cli-test check-runtime-trust release-cli-artifacts web-dev web-build web-lint web-e2e release-cut

RUNTIME_TRUST_LDFLAGS = -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.codexModelAttestationPublicKey=$(CODEX_MODEL_ATTESTATION_PUBLIC_KEY) -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.claudeModelAttestationPublicKey=$(CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY) -X github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.genericModelAttestationPublicKey=$(GENERIC_MODEL_ATTESTATION_PUBLIC_KEY)

help:
	@echo "Targets:"
	@echo "  make doctor        - Verify local toolchain prerequisites"
	@echo "  make validate      - Validate module structure and standards"
	@echo "  make manifests     - Validate skill/agent/tool manifests and registry schemas"
	@echo "  make registry      - Generate skills, agents, tools, and compatibility indexes"
	@echo "  make release-assets-check - Verify immutable release artifacts are committed"
	@echo "  make test-scripts  - Run deterministic script checks"
	@echo "  make test          - Run all local tests (validate + test-scripts)"
	@echo "  make ci-local      - Run local checks similar to CI"
	@echo "  make cli-test      - Run Go unit tests for CLI packages"
	@echo "  make cli-build     - Build the skills-hub CLI binary"
	@echo "  make cli-runtime-test - Test the preserved readiness development profile"
	@echo "  make cli-build-runtime-dev - Build readiness development CLI with explicit model trust roots"
	@echo "  make cli-build-test - Build an unreleased candidate with the installer release profile"
	@echo "  make release-cli-artifacts VERSION=vX.Y.Z - Build installer release binaries and checksums"
	@echo "  make web-dev       - Run Next.js hub app in dev mode"
	@echo "  make web-build     - Build Next.js hub app"
	@echo "  make web-lint      - Lint Next.js hub app"
	@echo "  make web-e2e       - Run Playwright smoke tests for web app"
	@echo "  make release-cut VERSION=vX.Y.Z[-alpha.N] - Validate and push release tag from main"

doctor:
	@echo "[check] go"
	@command -v go >/dev/null 2>&1 || (echo "Missing go (>=1.22)." && exit 1)
	@go version
	@echo "[check] python3"
	@command -v python3 >/dev/null 2>&1 || (echo "Missing python3 (>=3.10)." && exit 1)
	@python3 --version
	@echo "[check] check-jsonschema"
	@command -v check-jsonschema >/dev/null 2>&1 || (echo "Missing check-jsonschema. Install with: python3 -m pip install check-jsonschema" && exit 1)
	@check-jsonschema --version
	@echo "Environment looks ready."

validate:
	bash scripts/validate-skills.sh

manifests:
	bash scripts/validate-manifests.sh

registry:
	go run ./cmd/registry-builder

release-assets-check:
	cd apps/web && node scripts/prepare-public-assets.mjs
	@test -z "$$(git status --short --untracked-files=all -- releases)" || (git status --short --untracked-files=all -- releases; echo "Release store changed; commit new versions or increment a colliding version."; exit 1)

test-scripts:
	bash scripts/test-skill-scripts.sh
	bash scripts/test-release-history.sh

test: validate test-scripts

ci-local: test registry manifests cli-test

cli-test:
	go test ./...

check-runtime-trust:
	@go run ./cmd/runtime-trust-validator \
		--codex "$(CODEX_MODEL_ATTESTATION_PUBLIC_KEY)" \
		--claude "$(CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY)" \
		--generic "$(GENERIC_MODEL_ATTESTATION_PUBLIC_KEY)"

cli-build:
	GOFLAGS= GOWORK=off go build -tags installer_release -o bin/skills-hub ./cmd/skills-hub

# Candidates exercise the same command surface as a released installer.
cli-build-test:
	GOFLAGS= GOWORK=off go build -tags installer_release -o bin/skills-hub-test ./cmd/skills-hub

# Explicit development-only profile; never used by release publication.
cli-build-runtime-dev: check-runtime-trust
	go build -tags runtime_readiness -ldflags "$(RUNTIME_TRUST_LDFLAGS)" -o bin/skills-hub-runtime-dev ./cmd/skills-hub

cli-runtime-test:
	go test -tags runtime_readiness ./cmd/skills-hub

release-cli-artifacts:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release-cli-artifacts VERSION=vX.Y.Z"; exit 1; fi
	bash scripts/build-governed-cli-release.sh "$(VERSION)" "dist"

web-dev:
	cd apps/web && pnpm dev

web-build:
	cd apps/web && pnpm build

web-lint:
	cd apps/web && pnpm lint

web-e2e:
	cd apps/web && pnpm test:e2e

release-cut:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release-cut VERSION=vX.Y.Z[-alpha.N]"; exit 1; fi
	bash scripts/release-cut.sh "$(VERSION)"
