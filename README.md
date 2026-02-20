# AI Skills Guide

Practical, reusable AI skills for marketing practitioners and ad-tech software engineers.

## What this repo is

This repository is the executable companion to our written guide. It contains production-oriented `SKILL.md` packages, deterministic scripts, test prompts, and contribution standards.

- Guide article: [The Agent Architect’s Playbook: Building AI Skills for Marketing & Ad Tech](https://ai-news-hub.performics-labs.com/news/agent-architect-playbook-building-ai-skills-marketing-adtech)
- Community references: [VoltAgent awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills)

## v0.1 Scope (6 practical skills)

1. `meta-google-weekly-performance-review` (Beginner)
2. `creative-workshop-pmax-reels` (Intermediate)
3. `lifecycle-experiment-planner` (Intermediate)
4. `policy-brand-compliance-checker` (Intermediate)
5. `seo-paid-search-synergy` (Advanced)
6. `analyst-copilot-bigquery-redshift` (Advanced)

## Definition of done for each skill

- Has `SKILL.md` with clear routing intent and guardrails
- Has `tests/test-prompts.md` (>= 5 realistic prompts)
- Has `examples/` with sample input/output shape
- Documents runtime assumptions and dependencies
- Uses scripts/config for deterministic logic where relevant

## Repository layout

```text
skills/
  marketing/
  adtech/
shared/
  metrics/
  policies/
  schemas/
  naming/
docs/
scripts/
.github/
```

## Provider and framework examples to explore

Cross-runtime examples are cataloged in [awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills). Useful sections include skills for:
- OpenAI Codex / Agent Skills
- Claude-style skills
- Gemini CLI patterns
- GitHub Copilot / VS Code patterns
- Vercel AI SDK agent resources

## Quickstart

1. Pick a skill folder under `skills/`.
2. Read `README.md` + `SKILL.md` for required inputs.
3. Run prompts in `tests/test-prompts.md`.
4. Verify structure with `bash scripts/validate-skills.sh`.
5. Submit improvements via PR.

## Contributing

See `CONTRIBUTING.md` and `docs/how-to-contribute-a-skill.md`.
