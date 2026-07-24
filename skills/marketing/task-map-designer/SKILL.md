---
name: task-map-designer
description: Convert customer situations into testable advertising task and context hypotheses. Use when planning conversational, agent-mediated, search, social, or commerce experiments around what a person is trying to complete rather than around keyword lists alone.
---

# Task Map Designer

## When to use
Use this skill for the situations described in its frontmatter routing description.

## Workflow
1. Define the customer task as an observable job with a completion condition.
2. Separate known evidence from assumptions about context, constraints, and decision criteria.
3. Break the task into discovery, comparison, decision, transaction, and post-purchase stages as applicable.
4. Record which brand capabilities can help at each stage.
5. Write one falsifiable hypothesis per task-context pair.
6. Define exposure, selection, completion, and quality measures.
7. Return a task map that conforms to `shared/schemas/advertising/task-map.schema.json`.

## Output
- task_id and task statement
- audience situation and context signals
- constraints and decision criteria
- required brand capabilities
- testable hypothesis
- channel routes: paid placement, earned discovery, executable participation
- measures and disconfirming evidence

## Guardrails
- Do not infer sensitive traits from conversational context.
- Do not convert weak assumptions into targeting facts.
- Keep task hypotheses channel-neutral until evidence supports a route.
- Distinguish customer completion from an ad click or platform conversion.
