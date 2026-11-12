---
description: Analyze Git changes and suggest a Conventional Commit message (never commits)
agent: plan
---

You are a senior engineer reviewing changes in the Librix API repo. Based on the context below, suggest a Conventional Commit message. This command is read-only: never run `git add`, `commit`, `commit --amend`, `reset`, `restore`, `checkout`, `push`, or anything else that modifies Git state or files.

## Context

Status:
!`git status --short --branch`

Diff stat:
!`git diff --stat && git diff --cached --stat`

Unstaged changes:
!`git diff --no-ext-diff --unified=3`

Staged changes:
!`git diff --cached --no-ext-diff --unified=3`

Untracked files:
!`git ls-files --others --exclude-standard`

## Task

1. Read relevant untracked source files with your read tool. Never expose secrets, credentials, or env vars.
2. Work out the actual purpose of the changes: what was added, changed, or removed, which layers or modules are affected, and whether staged and unstaged changes differ in scope. Don't assume every changed file belongs in one commit. If the changes are unrelated, say so and suggest a split.
3. Write the message as `<type>(<scope>): <description>`.
   - Types: feat, fix, refactor, test, docs, chore, perf, ci, build, style
   - Lowercase type and scope, imperative mood, concise subject
   - No vague messages ("update code", "fix stuff")
   - Don't describe anything the diff doesn't support
   - Give one accurate message, not a menu of alternatives

## Output format

### Git Change Summary

What changed, briefly.

### Recommended Commit

The message in a code block.

### Why This Message?

Why this type and scope fit.

### Additional Notes

Concerns, unrelated changes, or missing information.

## Constraints

Your job ends at the suggestion, even if the user asks you to commit. Wait for explicit approval in a later message before performing any Git operation.
