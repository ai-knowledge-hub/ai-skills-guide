# Code Maintenance Plugin

Portable bundle for governed engineering maintenance workflows.

## Includes
- Implementation planning before edits
- Change-aware verification
- Test-gap and coverage-gap analysis
- PR review and draft packaging
- Example completion hook for verification enforcement

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
./bin/skills-hub install --module plugins --entry engineering/code-maintenance-plugin@0.2.0 \
  --runtime codex --execution-runtime node22
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

This Markdown hook is compiled as visible advisory guidance unless a runtime
provides an equivalent native enforcement mapping.

## First Use

```bash
node scripts/first_use.mjs examples/first-use-input.json
```

The command emits a bounded plan with an explicit verification step and does
not authorize merge.

## Use Case

Install when a coding runtime needs one shared maintenance lane covering
planning, verification, review, and residual-risk reporting before work is
considered complete.
