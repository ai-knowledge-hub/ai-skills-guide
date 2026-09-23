# Creative Operating System Supervisor

This agent coordinates the Creative Operating System workflow described in the documentation pack. It is designed for marketing teams that want AI-assisted work to retain memory, usefulness, timing, governance, and distinctiveness.

## What this agent does

- Audits whether the team has the machinery to use AI creatively without producing generic work.
- Routes the right skill at the right stage: audit, product-as-media, utility concept, cultural timing, creator brief, evaluation.
- Produces a launch-readiness summary with approval requirements.

## Runtime assumptions

- Claude registers this agent natively from the plugin package.
- Codex and generic runtimes install the definition as advisory guidance because
  they do not currently expose a native plugin-agent registration contract.
- Brand memory, campaign archive, trend, and approval integrations are optional local environment bindings, not cataloged tool dependencies in this repo.
- Example binding shapes are documented in `config/tool-bindings.example.json`.

## When to use it

Use it for:

- Cannes-style effectiveness reviews.
- New campaign concept development.
- Creator-led campaign planning.
- Product-led marketing opportunities.
- Cultural moment response planning.
- Anti-slop review of AI-generated campaign ideas.

## Install

Preferred for first-time users:

```bash
./bin/skills-hub install --module plugins --entry marketing/creative-operating-system-plugin@0.2.0 \
  --runtime codex --execution-runtime node22
```

Agent-only install if you already have the surrounding workflow and templates:

```bash
./bin/skills-hub install marketing/creative-operating-system-supervisor@latest \
  --module agents --runtime codex
```

## First run prompt

```text
Use Creative Operating System Supervisor.
We are planning a new product launch campaign.
Audit our creative operating system, identify the biggest gaps,
then propose a workflow using the available skills before we generate ideas.
```

## Good output looks like

- It starts with memory and evidence, not immediate copy generation.
- It distinguishes concept quality from execution readiness.
- It identifies approvals before risky work moves forward.
- It connects utility, product surfaces, creators, and evaluation into one process.
