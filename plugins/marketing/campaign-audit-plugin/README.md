# Campaign Audit Plugin

Portable bundle for campaign QA and governance checks.

## Includes
- Compliance and policy-focused skills
- Campaign QA supervisor agent
- Analytics and ad connector references
- Example block-publish hook

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- bundled agents are installed into the runtime `agents/...` directory
- bundled tools are installed into the runtime `tools-mcp/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills, agents, and tools do not live inside the plugin directory
itself. That avoids duplication and preserves the plugin as a composition
package.

## Use Case
Install when media or governance teams need a repeatable package for pre-launch audits and major campaign-change reviews.
