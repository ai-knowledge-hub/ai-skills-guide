# AI Output Eval Scorecard

Use this skill to score AI-generated outputs with a repeatable quality,
compliance, and creative operating system rubric.

## What this skill does

- Scores accuracy, clarity, brand fit, compliance, and actionability.
- Adds optional Creative Operating System dimensions for Cannes-style creative review: utility, product-as-media potential, cultural timing, human meaning, brand memory, distinctiveness, and creator fit.
- Flags high-severity findings with evidence.
- Returns pass/revise/fail verdict.

## Before you start

1. Provide task context and prompt used.
2. Provide generated output.
3. Provide brand and policy rules.

## Install

```bash
./bin/skills-hub install \
  marketing/ai-output-eval-scorecard@latest \
  --runtime codex
```

## First run prompt

```text
Use AI Output Eval Scorecard.
Task type: campaign concept
Score this output for quality, compliance, utility, distinctiveness,
product-as-media potential, cultural timing, and brand memory.
Return overall score, verdict, critical findings,
and rewrite recommendations.
```

## What good output looks like

- Dimension scores are clear.
- Critical findings include concrete evidence.
- Verdict is actionable.
- Creative work is not rewarded for polish alone; usefulness, evidence, and brand permission are scored explicitly.

## Beginner safety checklist

- Do not infer missing policy rules.
- Do not fabricate compliance certainty.
- Keep scoring criteria consistent across runs.
