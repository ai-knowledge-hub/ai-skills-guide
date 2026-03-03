# Security Baseline (Draft v0.1)

This document defines the minimum security expectations for this repository.
It covers `skills`, `agents`, and `tools-mcp` packages.

Status: experimental and evolving.

## Scope and Trust Model

- The registry is a discovery layer, not a security approval system.
- A listed package is not automatically safe for production use.
- Every adopting team is responsible for runtime security controls.
- Treat all package content as review-required before client deployment.

## Security Principles

- Least privilege by default.
- Read-only by default for data access and integrations.
- Explicit human approval for high-risk or write actions.
- Full auditability for tool calls and critical decisions.
- Defense in depth against prompt injection and data exfiltration.

## Threats to Address

- Prompt injection from untrusted content (docs, URLs, chat, tickets).
- Tool misuse by over-permissioned connectors.
- Data exfiltration to unapproved destinations.
- Silent failure modes that hide data quality or policy violations.
- Supply chain risk from unreviewed external contributions.

## Minimum Runtime Controls (Required)

### 1) Identity and Access

- Use service accounts, not personal credentials, for integrations.
- Scope API permissions to the smallest required set.
- Prefer short-lived credentials and regular credential rotation.
- Separate dev, staging, and production credentials.

### 2) Tool Execution Guardrails

- Apply per-agent tool allowlists.
- Validate parameters against strict schemas before execution.
- Block dangerous operations unless explicitly approved.
- Enforce output contracts for deterministic downstream handling.

### 3) Approval Gates

- Auto-approve read/reporting tasks only.
- Require human approval for write or irreversible actions.
- Require explicit approval for budget or targeting changes.

### 4) Data Handling

- Classify data sensitivity before enabling connectors.
- Mask or redact sensitive values in logs and outputs.
- Restrict outbound network targets where possible.
- Do not persist sensitive payloads beyond required retention.

### 5) Audit and Observability

- Log: requested action, selected tool, parameters, result, timestamp.
- Log policy/guardrail decisions and approval events.
- Preserve immutable audit records for incident review.
- Alert on repeated failures, policy violations, or unusual usage spikes.

### 6) Prompt Injection Defenses

- Treat all external content as untrusted input.
- Separate instruction channels from data channels.
- Reject tool calls that depend on untrusted instruction overrides.
- Require citation/evidence for critical recommendations.

## Contribution Security Requirements

All PRs that add or change `agents` or `tools-mcp` should include:

- permissions required (read/write scopes)
- data sources touched
- expected logging behavior
- failure modes and safe fallback behavior
- approval requirements for risky actions

If this information is missing, maintainers may request updates before merge.

## Production Adoption Checklist

Before production rollout, confirm:

- security review completed by your local engineering/security owners
- IAM scopes validated and least privilege enforced
- runtime guardrails and approval gates enabled
- audit logging wired to internal monitoring
- incident response owner and rollback path defined

## Current Repository Position

- This repo contains templates, manifests, and implementation guidance.
- Some tool and MCP entries are intentionally starter specifications.
- Adopters should harden connectors and runtime policies in their own
  environment before handling client data.

## Planned Next Steps

- Add trust labels (`experimental`, `reviewed`, `production-candidate`).
- Add security checklist to PR template.
- Add policy examples for prompt injection and exfiltration controls.
- Add a governance document for deprecation and incident handling.

## Feedback

Use GitHub Discussions to propose controls, threat scenarios, or
compliance requirements:

[GitHub Discussions](https://github.com/ai-knowledge-hub/ai-skills-guide/discussions)
