# Autoharnessing Skill Pack

## Purpose

This pack supports controlled evolution of harness artifacts such as
`AGENTS.md`, `AGENTS-SECURITY.md`, and repository-local skills.

## Included Skills

- `agentops/harness-run-reflection`
- `agentops/harness-skill-proposal`
- `agentops/harness-regression-evaluator`

## Capability Boundaries

- Harness skills may only propose or evaluate changes to harness-owned
  artifacts by default.
- They must not directly modify app runtime code, infra, or deployment
  config without human approval.
- Model weights are not changed; only scaffold and policy artifacts evolve.

## Recommended Permissions

For Codex and Claude Code:

- read run logs, benchmark prompts, and harness policies
- write to draft branches or staging directories only
- require approval before committing accepted harness updates

For generic runtimes:

- isolate harness work in a dedicated workspace
- keep production code paths read-only to the harness agent

## Human Approval Requirements

- expansion of writable scope outside harness-owned paths
- adoption of new benchmark gates
- changes to rollback, incident, or security policy wording

## Minimal Context Rule

Keep `AGENTS.md` short and decision-oriented. Put benchmark data,
red-team prompts, and long-form rationale in separate harness docs.
