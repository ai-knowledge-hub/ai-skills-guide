# Runtime Safety Plugin

Portable bundle for conservative runtime and repository safety review.

## Includes
- Untrusted-content handling
- Dependency and supply-chain review
- Secret and credential hygiene checks
- Environment risk assessment
- Example quarantine hook for suspicious instructions

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
./bin/skills-hub install --module plugins --entry security/runtime-safety-plugin@0.2.0 \
  --runtime codex --execution-runtime node22
```

## Installed Dependencies

Skills installed into `skills/...`:
- `security/handle-untrusted-content`
- `security/dependency-supply-chain-audit`
- `security/secrets-and-credential-hygiene`
- `security/environment-risk-assessment`

## Packaged Hooks

Hooks kept inside this plugin package:
- `quarantine-suspicious-instructions`

This Markdown hook is compiled as visible advisory guidance unless a runtime
provides an equivalent native enforcement mapping.

## First Use

```bash
node scripts/first_use.mjs examples/first-use-input.json
```

The command produces a conservative decision and never performs remediation.

## Use Case

Install when a runtime should default to halt, escalate, and recommend for
suspicious content, risky dependencies, leaked secrets, and unsafe execution
environments before any live action is attempted.
