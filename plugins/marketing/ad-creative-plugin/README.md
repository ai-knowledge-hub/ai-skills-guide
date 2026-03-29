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

## Use Case
Install when a team wants one reusable package for developing ad concepts, evaluating them against rules and hypotheses, and handing approved variants into production workflows.
