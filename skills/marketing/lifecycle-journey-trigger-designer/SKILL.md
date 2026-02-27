---
name: lifecycle-journey-trigger-designer
description: Design lifecycle journey triggers, channel sequencing, and measurement plans from behavior and audience signals.
---

# Lifecycle Journey Trigger Designer

## When to use
Use when creating or refactoring lifecycle programs across email, push, and in-app channels with event-based triggers.

## Inputs required
- journey_goal
- audience_definition
- trigger_events
- channel_constraints
- frequency_caps
- success_metrics

## Workflow
1. Validate journey objective and audience boundary.
2. Map entry, progression, and exit triggers.
3. Define channel sequence and timing windows.
4. Add suppression rules and frequency controls.
5. Return rollout plan with success metrics.

## Output format
- journey_blueprint
- trigger_rules
- channel_sequence
- suppression_rules
- measurement_plan

## Guardrails
- Avoid overlapping trigger logic that can double-send.
- Respect channel frequency and legal communication constraints.
- Mark dependencies on missing event instrumentation.
