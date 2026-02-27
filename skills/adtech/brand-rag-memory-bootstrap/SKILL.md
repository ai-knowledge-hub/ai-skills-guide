---
name: brand-rag-memory-bootstrap
description: Bootstrap private brand memory and retrieval structure so AI skills answer from approved campaign and policy context.
---

# Brand RAG Memory Bootstrap

## When to use
Use when setting up or refreshing a private knowledge base that agent skills must query before drafting strategy or creative.

## Inputs required
- source_docs
- brand_taxonomy
- policy_documents
- campaign_history
- chunking_strategy
- retrieval_requirements

## Workflow
1. Audit and classify source documents.
2. Normalize metadata (product, audience, channel, date, policy tags).
3. Define chunking and citation rules.
4. Build retrieval index structure and quality checks.
5. Return ingestion and refresh plan.

## Output format
- source_inventory
- metadata_schema
- chunking_plan
- retrieval_eval_plan
- refresh_schedule

## Guardrails
- Restrict memory to approved private sources.
- Require citation paths for generated claims.
- Flag stale or conflicting sources before publishing guidance.
