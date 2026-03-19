# Cybersecurity Skill Pack

## Purpose

This pack gives agents a defensive layer for untrusted content, dependency
review, secret hygiene, and runtime risk assessment.

## Included Skills

- `security/handle-untrusted-content`
- `security/dependency-supply-chain-audit`
- `security/secrets-and-credential-hygiene`
- `security/environment-risk-assessment`

## Capability Boundaries

- Analysis, halt, and escalation come before action.
- No skill in this pack should claim trust in third-party MCP servers by
  default.
- No destructive remediation is performed automatically in this release.

## Recommended Permissions

For Codex and Claude Code:

- read manifests, lockfiles, configs, and logs
- run approved scanners in sandboxed mode
- redact sensitive values in outputs

For generic runtimes:

- isolate scanner tools from production credentials
- restrict outbound network destinations where possible

## Human Approval Requirements

- credential rotation
- dependency replacement that changes runtime behavior
- edits to secret stores, CI secrets, or deployment credentials
- any action affecting production or shared environments

## Minimal Context Rule

Reference only the current security policy, trusted tool list, and approval
rules in `AGENTS.md` or `AGENTS-SECURITY.md`. Keep the rest in dedicated
docs to reduce drift and hallucinated policy.
