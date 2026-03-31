# Harness Governance Plugin

Portable bundle for governed harness improvement.

## Includes
- Harness run reflection
- Harness skill proposal drafting
- Harness regression evaluation
- Example hook enforcing human approval before adoption

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- no agents are installed for this plugin
- no tools are installed for this plugin
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled harness skills do not live inside the plugin directory itself. That
avoids duplication and keeps the plugin focused on governance rather than
self-contained policy forks.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry agentops/harness-governance-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `agentops/harness-run-reflection`
- `agentops/harness-skill-proposal`
- `agentops/harness-regression-evaluator`

## Packaged Hooks

Hooks kept inside this plugin package:
- `require-human-approval-for-harness-adoption`

## Use Case

Install when a runtime should be able to inspect, propose, and evaluate
harness changes, but must never adopt them automatically without human
approval.
