# Harness Governance Plugin

Portable bundle for governed harness improvement.

## Includes
- Harness run reflection
- Harness skill proposal drafting
- Harness regression evaluation
- Example hook enforcing human approval before adoption

## Install Behavior

This plugin is published as a self-contained, integrity-locked packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- no agents are installed for this plugin
- no tools are installed for this plugin
- packaged hooks remain inside this plugin's `hooks/` directory

The release archive carries the pinned skill closure, dependency lock,
checksums, SBOM, and provenance. Installation never depends on sibling source
directories.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry agentops/harness-governance-plugin@0.2.0 \
  --runtime codex --execution-runtime node22
```

## Installed Dependencies

Skills installed into `skills/...`:
- `agentops/harness-run-reflection`
- `agentops/harness-skill-proposal`
- `agentops/harness-regression-evaluator`

## Packaged Hooks

Hooks kept inside this plugin package:
- `require-human-approval-for-harness-adoption`

This Markdown hook is compiled as visible advisory guidance unless a runtime
provides an equivalent native enforcement mapping.

## First Use

```bash
node scripts/first_use.mjs examples/first-use-input.json
```

The command emits a bounded decision record and never authorizes adoption.

## Use Case

Install when a runtime should be able to inspect, propose, and evaluate
harness changes, but must never adopt them automatically without human
approval.
