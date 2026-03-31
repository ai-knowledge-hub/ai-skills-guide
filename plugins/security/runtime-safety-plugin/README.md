# Runtime Safety Plugin

Portable bundle for conservative runtime and repository safety review.

## Includes
- Untrusted-content handling
- Dependency and supply-chain review
- Secret and credential hygiene checks
- Environment risk assessment
- Example quarantine hook for suspicious instructions

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- no agents are installed for this plugin
- no tools are installed for this plugin
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled security skills do not live inside the plugin directory itself. That
avoids duplication while keeping the plugin usable as a single review lane.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry security/runtime-safety-plugin@0.1.0 --runtime codex
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

## Use Case

Install when a runtime should default to halt, escalate, and recommend for
suspicious content, risky dependencies, leaked secrets, and unsafe execution
environments before any live action is attempted.
