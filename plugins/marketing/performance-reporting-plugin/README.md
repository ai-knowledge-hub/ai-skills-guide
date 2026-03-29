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

## Use Case
Install when a team needs one shared reporting package instead of manually wiring separate skills, tools, and review steps.
