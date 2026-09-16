# Content Repurposing Plugin

Portable bundle for content reuse workflows across channels and formats.

## Includes
- Content adaptation and creative rules skills
- Output evaluation support
- Example review-before-publish hook

## Install Behavior

This plugin is published as a self-contained, integrity-locked packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

The release archive carries the exact bundled skill closure plus a dependency
lock, checksum manifest, CycloneDX SBOM, and provenance record. Installation
materializes those bundled skills into their native runtime directory without
reading sibling repository source directories.

## Install Command

```bash
./bin/skills-hub install --source remote --module plugins \
  --entry marketing/content-repurposing-plugin@0.2.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `marketing/creative-workshop-pmax-reels`
- `marketing/ai-output-eval-scorecard`
- `marketing/dynamic-creative-rules-engine`

## Packaged Hooks

Hooks kept inside this plugin package:
- `content-review-before-publish`

## Use Case
Install when teams want a repeatable plugin for turning one source asset into multiple publish-ready variants with review controls.
