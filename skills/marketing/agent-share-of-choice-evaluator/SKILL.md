---
name: agent-share-of-choice-evaluator
description: Evaluate whether a brand is retrieved, understood, considered, recommended, selected, transacted with, and used successfully across a stable panel of agent-mediated customer tasks. Use for AI visibility, conversational advertising, agentic commerce, or paid-versus-earned discovery experiments.
---

# Agent Share of Choice Evaluator

## When to use
Use this skill for the situations described in its frontmatter routing description.

## Workflow
1. Load a versioned task panel and freeze the evaluation conditions.
2. Run each task consistently across the selected agent surfaces.
3. Capture evidence for retrieval, understanding, consideration, recommendation, paid response, selection, transaction, completion, and reuse.
4. Label observed, synthetic, platform-reported, and independently verified evidence separately.
5. Calculate stage rates only where denominators and samples are explicit.
6. Compare paid, earned, and executable routes without collapsing them into one rank.
7. Report failures, uncertainty, and the next experiment.

## Output
- task panel version and run conditions
- stage evidence by task
- stage rates with sample sizes
- route comparison
- evidence quality and confidence
- failure modes and next tests

## Guardrails
- Do not claim a universal agent ranking.
- Do not treat one model, prompt, geography, or session as representative.
- Do not merge paid exposure with earned recommendation.
- Do not call platform conversion data independently verified.
