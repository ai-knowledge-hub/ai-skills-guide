# Performance Reporting Plugin

Portable bundle for weekly reporting workflows.

## Includes
- Weekly performance reporting skills
- Dashboard generation and QA
- Reporting supervisor agent
- Connector references for analytics, ads, and warehouse data
- Example post-analysis Slack hook

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- bundled agents are installed into the runtime `agents/...` directory
- bundled tools are installed into the runtime `tools-mcp/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills, agents, and tools do not live inside the plugin directory
itself. That avoids duplication and reflects the plugin's role as an
installable composition package.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/performance-reporting-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `marketing/meta-google-weekly-performance-review`
- `adtech/dashboard-generator`
- `adtech/dashboard-qa-checker`
- `adtech/executive-narrative-writer`

Agents installed into `agents/...`:
- `marketing/weekly-performance-supervisor`

Tools installed into `tools-mcp/...`:
- `analytics/ga4-mcp-connector`
- `ads/meta-ads-mcp-connector`
- `warehouse/bigquery-mcp-query-runner`

## Packaged Hooks

Hooks kept inside this plugin package:
- `post-analysis-slack-summary`

## Use Case
Install when a team needs one shared reporting package instead of manually wiring separate skills, tools, and review steps.
