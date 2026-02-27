SHELL := /bin/bash

.PHONY: help doctor validate manifests registry test-scripts test ci-local cli-build cli-test web-dev web-build web-lint web-e2e release-cut

help:
	@echo "Targets:"
	@echo "  make doctor        - Verify local toolchain prerequisites"
	@echo "  make validate      - Validate skill structure and standards"
	@echo "  make manifests     - Validate skill.yaml and registry index schemas"
	@echo "  make registry      - Generate registry/index.json from skill manifests"
	@echo "  make test-scripts  - Run deterministic script checks"
	@echo "  make test          - Run all local tests (validate + test-scripts)"
	@echo "  make ci-local      - Run local checks similar to CI"
	@echo "  make cli-test      - Run Go unit tests for CLI packages"
	@echo "  make cli-build     - Build the skills-hub CLI binary"
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

test-scripts:
	bash scripts/test-skill-scripts.sh

test: validate test-scripts

ci-local: test registry manifests

cli-test:
	go test ./...

cli-build:
	go build -o bin/skills-hub ./cmd/skills-hub

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
