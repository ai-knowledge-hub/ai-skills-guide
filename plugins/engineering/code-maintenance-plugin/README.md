# Code Maintenance Plugin

Portable bundle for governed engineering maintenance workflows.

## Includes
- Implementation planning before edits
- Change-aware verification
- Test-gap and coverage-gap analysis
- PR review and draft packaging
- Example completion hook for verification enforcement

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- no agents are installed for this plugin
- no tools are installed for this plugin
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills do not live inside the plugin directory itself. That avoids
source duplication and keeps the plugin focused on orchestration and policy.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry engineering/code-maintenance-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `engineering/implementation-strategy`
- `engineering/code-change-verification`
- `engineering/test-gap-analyzer`
- `engineering/coverage-gap-reporter`
- `engineering/pr-review-and-draft`

## Packaged Hooks

Hooks kept inside this plugin package:
- `verification-before-complete`

## Use Case

Install when a coding runtime needs one shared maintenance lane covering
planning, verification, review, and residual-risk reporting before work is
considered complete.
