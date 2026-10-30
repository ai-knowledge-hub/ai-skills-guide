# Competitive Intelligence Plugin

Portable bundle for evidence-based competitor monitoring and synthesis.

## Includes
- Brand memory bootstrap for structured evidence capture
- Output evaluation guidance for ranking claims and insight quality
- Untrusted-content handling to avoid taking instructions from scraped sources
- Example digest hook for weekly competitive reporting

## Install Behavior

This plugin is published as a self-contained, integrity-locked packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

The release archive carries the pinned skill closure, dependency lock,
checksums, SBOM, and provenance. Installation never depends on sibling source
directories.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/competitive-intelligence-plugin@0.2.0 \
  --runtime claude --execution-runtime node22
```

## Installed Dependencies

Skills installed into `skills/...`:
- `adtech/brand-rag-memory-bootstrap`
- `marketing/ai-output-eval-scorecard`
- `security/handle-untrusted-content`

## Packaged Hooks

Hooks kept inside this plugin package:
- `weekly-competitor-signal-digest`

This Markdown hook is compiled as visible advisory guidance unless a runtime
provides an equivalent native enforcement mapping.

## First Use

```bash
node scripts/first_use.mjs examples/first-use-input.json
```

The command labels every external signal as untrusted and requiring human
review before publication.

## Use Case
Install when a team needs a repeatable, safer workflow for collecting competitor signals, structuring observations, and sharing evidence-backed summaries without treating external content as trusted instructions.
