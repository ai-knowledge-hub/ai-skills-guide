# Competitive Intelligence Plugin

Portable bundle for evidence-based competitor monitoring and synthesis.

## Includes
- Brand memory bootstrap for structured evidence capture
- Output evaluation guidance for ranking claims and insight quality
- Untrusted-content handling to avoid taking instructions from scraped sources
- Example digest hook for weekly competitive reporting

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills do not live inside the plugin directory itself. That keeps the
plugin lightweight and avoids duplicating the source component packages.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/competitive-intelligence-plugin@0.1.0 --runtime claude
```

## Installed Dependencies

Skills installed into `skills/...`:
- `adtech/brand-rag-memory-bootstrap`
- `marketing/ai-output-eval-scorecard`
- `security/handle-untrusted-content`

## Packaged Hooks

Hooks kept inside this plugin package:
- `weekly-competitor-signal-digest`

## Use Case
Install when a team needs a repeatable, safer workflow for collecting competitor signals, structuring observations, and sharing evidence-backed summaries without treating external content as trusted instructions.
