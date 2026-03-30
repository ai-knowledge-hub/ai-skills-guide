# Page Speed & Technical SEO Plugin

Portable bundle for technical SEO and landing-page diagnostics.

## Includes
- Search and paid-search alignment skill guidance
- Browser-driven page review workflow
- Runtime environment risk checks before automated diagnostics
- GA4 connector reference for landing-page and engagement validation
- Example hook for Lighthouse-style review handoff

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- bundled tools are installed into the runtime `tools-mcp/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills and tools do not live inside the plugin directory itself. That
avoids duplication while still making the plugin install materially usable.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/page-speed-technical-seo-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `marketing/seo-paid-search-synergy`
- `adtech/playwright-agentic-e2e`
- `security/environment-risk-assessment`

Tools installed into `tools-mcp/...`:
- `analytics/ga4-mcp-connector`

## Packaged Hooks

Hooks kept inside this plugin package:
- `lighthouse-and-landing-page-review`

## Use Case
Install when a team needs a repeatable workflow for reviewing slow landing pages, technical SEO regressions, and page-experience issues before escalating changes to engineering.
