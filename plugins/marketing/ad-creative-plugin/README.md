# Ad Creative Plugin

Portable bundle for paid-media creative ideation, QA, and experiment planning.

## Includes
- Creative ideation workflow for PMax, reels, and paid-social variants
- Rules-based creative review guidance
- A/B testing and evaluation support
- Meta Ads connector reference for campaign-context setup
- Example pre-publish creative review hook

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- bundled tools are installed into the runtime `tools-mcp/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills and tools do not live inside the plugin directory itself. That
avoids duplication and keeps each component in its native runtime location.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/ad-creative-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `marketing/creative-workshop-pmax-reels`
- `marketing/dynamic-creative-rules-engine`
- `marketing/ab-test-planner-analyzer`
- `marketing/ai-output-eval-scorecard`

Tools installed into `tools-mcp/...`:
- `ads/meta-ads-mcp-connector`

## Packaged Hooks

Hooks kept inside this plugin package:
- `creative-review-before-publish`

## Use Case
Install when a team wants one reusable package for developing ad concepts, evaluating them against rules and hypotheses, and handing approved variants into production workflows.
