# Brand RAG Memory Bootstrap

Use this skill to set up a private brand memory system that other skills
can query before generating outputs.

## What this skill does

- Audits and classifies brand/policy/campaign documents.
- Defines metadata schema and chunking strategy.
- Produces retrieval quality and refresh plans.

## Before you start

1. Collect approved source documents.
2. Define taxonomy labels (brand, channel, audience, date).
3. Define refresh cadence and ownership.

## Install

```bash
./bin/skills-hub install \
  adtech/brand-rag-memory-bootstrap@latest \
  --runtime codex
```

## First run prompt

```text
Use Brand RAG Memory Bootstrap.
Sources: brand guidelines, policy docs, campaign history.
Return source inventory, metadata schema, chunking plan,
retrieval eval plan, and refresh schedule.
```

## What good output looks like

- Source inventory is complete and categorized.
- Metadata keys are consistent.
- Retrieval plan includes evaluation checks.

## Beginner safety checklist

- Use only approved private documents.
- Require citation paths for key claims.
- Flag stale sources before use.
