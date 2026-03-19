# Implementation Strategy

## Purpose

This skill forces a code agent to understand a repository before editing it.
It is the planning layer for maintenance work.

## What it does

- reads policy and repo structure
- maps likely impacted files and dependencies
- lists the right verification commands
- records risks and approval boundaries

## Recommended runtime permissions

- read repository files
- read policy files such as `AGENTS.md`
- no write access required to use this skill

## Deterministic assets

- `config/repo-intake-checklist.yaml`

## Output

The output should be a short, reviewable strategy with explicit file and
command references.
