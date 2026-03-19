# Code Maintenance Skill Pack

## Purpose

This pack provides reusable repo-maintenance skills for planning, verifying,
testing, coverage review, and PR drafting.

## Included Skills

- `engineering/implementation-strategy`
- `engineering/code-change-verification`
- `engineering/test-gap-analyzer`
- `engineering/coverage-gap-reporter`
- `engineering/pr-review-and-draft`

## Capability Boundaries

- Default to read-only repo inspection and local verification.
- Do not push, merge, deploy, or change CI settings by default.
- Keep edits scoped to the requested change.

## Recommended Permissions

For Codex:

- read repo
- run local build, lint, typecheck, and test commands
- write files only after approval

For Claude Code:

- keep default read-only permissions
- allow sandboxed test execution when needed
- review MCP server permissions separately

For generic runtimes:

- mount repo read-only first
- enable a writable working copy only for approved tasks

## Human Approval Requirements

- public API changes
- cross-cutting refactors
- dependency upgrades with security impact
- any push, merge, or release action

## Minimal Context Rule

Keep `AGENTS.md` focused on:

- repo purpose
- canonical commands
- approval boundaries
- required verification skills

Do not overload it with long tutorials or duplicated docs.
