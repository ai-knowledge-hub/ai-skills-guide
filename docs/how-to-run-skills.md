# How to Run Skills

## 1. Read required inputs

Open the skill's `README.md` and `SKILL.md`.

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
