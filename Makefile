SHELL := /bin/bash

.PHONY: help validate manifests registry test-scripts test ci-local cli-build cli-test

help:
	@echo "Targets:"
	@echo "  make validate      - Validate skill structure and standards"
	@echo "  make manifests     - Validate skill.yaml and registry index schemas"
	@echo "  make registry      - Generate registry/index.json from skill manifests"
	@echo "  make test-scripts  - Run deterministic script checks"
	@echo "  make test          - Run all local tests (validate + test-scripts)"
	@echo "  make ci-local      - Run local checks similar to CI"
	@echo "  make cli-test      - Run Go unit tests for CLI packages"
	@echo "  make cli-build     - Build the skills-hub CLI binary"

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
