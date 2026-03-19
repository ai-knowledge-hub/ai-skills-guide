# Dependency Supply Chain Audit

## Purpose

This skill helps agents evaluate third-party packages before they become
trusted parts of the runtime.

## Deterministic assets

- `config/dependency-audit-rubric.yaml`

## Dependencies

The first release assumes scanner adapters such as `trivy` or
`osv-scanner`, but the skill can still produce a conservative manual review
when those tools are missing.
