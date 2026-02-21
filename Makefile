SHELL := /bin/bash

.PHONY: help validate test-scripts test ci-local cli-build cli-test

help:
	@echo "Targets:"
	@echo "  make validate      - Validate skill structure and standards"
	@echo "  make test-scripts  - Run deterministic script checks"
	@echo "  make test          - Run all local tests (validate + test-scripts)"
	@echo "  make ci-local      - Run local checks similar to CI"
	@echo "  make cli-test      - Run Go unit tests for CLI packages"
	@echo "  make cli-build     - Build the skills-hub CLI binary"

validate:
	bash scripts/validate-skills.sh

test-scripts:
	bash scripts/test-skill-scripts.sh

test: validate test-scripts

ci-local: test

cli-test:
	go test ./...

cli-build:
	go build -o bin/skills-hub ./cmd/skills-hub
