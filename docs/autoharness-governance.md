# Autoharness Governance

## Goal

Allow harness policies and skills to improve over time without giving
autonomous agents uncontrolled authority over the main codebase.

## Allowed Artifacts

- `AGENTS.md`
- `AGENTS-SECURITY.md`
- `skills/`
- harness-only test prompts, benchmark files, and config

## Protected Artifacts

- application runtime code
- deployment and CI configuration
- infrastructure definitions
- secret-bearing files and credential stores
- production connectors under `tools-mcp/`

## Review Flow

1. Reflection skill identifies a repeated failure pattern.
2. Proposal skill drafts a skill or policy update.
3. Regression evaluator tests the change against saved prompts.
4. Human reviewer approves before merge.

## Required Evidence

- benchmark or run-log references
- reason for the proposed change
- expected improvement
- known risks and rollback path

## Rollback Rule

Any harness change that reduces safety, weakens approval boundaries, or
regresses benchmark behavior must be reverted or held for manual review.
