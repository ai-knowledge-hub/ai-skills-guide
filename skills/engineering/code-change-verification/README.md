# Code Change Verification

## Purpose

This skill standardizes post-change verification for code agents.

## What it does

- chooses commands by stack and changed paths
- runs the right verification checks
- reports pass, fail, skip, or flaky outcomes
- highlights remaining risk when verification is incomplete

## Recommended runtime permissions

- read repository files
- run local lint, typecheck, test, and build commands
- no Git push or deploy permissions

## Deterministic assets

- `config/command-matrix.yaml`

## Notes

This skill should be mandatory for runtime code changes before an agent marks
work complete.
