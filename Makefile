SHELL := /bin/bash

.PHONY: help validate test-scripts test ci-local

help:
	@echo "Targets:"
	@echo "  make validate      - Validate skill structure and standards"
	@echo "  make test-scripts  - Run deterministic script checks"
	@echo "  make test          - Run all local tests (validate + test-scripts)"
	@echo "  make ci-local      - Run local checks similar to CI"

validate:
	bash scripts/validate-skills.sh

test-scripts:
	bash scripts/test-skill-scripts.sh

test: validate test-scripts

ci-local: test
