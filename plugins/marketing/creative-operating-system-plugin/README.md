# Creative Operating System Plugin

Installable package for teams that want to move from AI-generated campaign ideas to a governed creative operating system with memory, utility, product surfaces, creator collaboration, timing, and evaluation.

## Includes
- Brand-memory bootstrap and operating-system audit skills
- Utility-led concepting and product-as-media mapping
- Cultural timing triage and creator briefing
- Final evaluation scorecard
- Creative Operating System supervisor agent
- Packaged hooks for memory gating and launch-gate review
- Starter templates and JSON schemas for first-run setup

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- bundled agents are installed into the runtime `agents/...` directory
- packaged hooks, templates, and schemas remain inside this plugin's directory

Bundled skills and agents do not live inside the plugin directory itself. That avoids duplication while still making the package usable immediately.

## Install Command

```bash
./bin/skills-hub install --module plugins --entry marketing/creative-operating-system-plugin@0.1.0 --runtime codex
```

## Installed Dependencies

Skills installed into `skills/...`:
- `adtech/brand-rag-memory-bootstrap`
- `marketing/creative-operating-system-audit`
- `marketing/utility-campaign-concept-designer`
- `marketing/product-as-media-mapper`
- `marketing/cultural-timing-signal-triage`
- `marketing/creator-strategy-brief`
- `marketing/ai-output-eval-scorecard`

Agents installed into `agents/...`:
- `marketing/creative-operating-system-supervisor`

## Packaged Hooks

Hooks kept inside this plugin package:
- `require-memory-before-ideation`
- `require-launch-gate-review`

## First-Run Path

If the team has no existing systems yet, run the package in this order:

1. Fill `templates/brand-memory-intake.md`
2. Fill `templates/campaign-archive-intake.md`
3. Fill `templates/approval-matrix.md`
4. Run `adtech/brand-rag-memory-bootstrap`
5. Run `marketing/creative-operating-system-audit`
6. Use the supervisor agent to plan the next workflow steps
7. Use `templates/trend-signal-triage.md`, `templates/creator-brief-intake.md`, and `templates/launch-gate-checklist.md` as needed

## What This Package Makes Immediately Easier

- starting without an existing memory structure
- capturing reusable campaign evidence
- running a manual approval model before formal workflow tooling exists
- moving from concepting into launch review without inventing the process from scratch

## What Still Needs Local Adaptation

This plugin is operationally useful out of the box, but teams still need to adapt:

- brand evidence and memory inputs
- campaign archive sources
- approval owners and sign-off rules
- creator disclosure requirements
- product, legal, and engineering feasibility checks

Those are packaged here as starter templates and schemas rather than pretending the repo can infer them automatically.
