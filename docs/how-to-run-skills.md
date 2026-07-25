# How to Run Skills, Agents, and Tools

## 1. Read required inputs

Open the entry `README.md` and spec file:
- skills: `SKILL.md`
- agents: `AGENT.md`
- tools-mcp: `TOOL.md`

Run `skills-hub info` and inspect `usability` first. `template-only` entries need implementation; `setup-required` entries need their listed credentials, bindings, and policies.

## 2. Provide minimum context

Supply account IDs, date range, platforms, and objective as requested.

## 3. Run test prompts first

Use prompts from `tests/test-prompts.md` before production usage.

## 4. Validate output

Check for:
- required sections
- no fabricated numbers
- uncertainty flags where data is missing

## 5. Escalate to scripts when needed

If outputs drift, use deterministic scripts and schemas from `shared/`.

## Registry files

When testing hub output consistency, use module indexes in `registry/`:
- `skills-index.json`
- `agents-index.json`
- `tools-index.json`
