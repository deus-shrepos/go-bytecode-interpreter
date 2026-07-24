---
name: worklog
description: >
  Record the work behind a pending commit as a precise entry in
  .claude/worklog/ — what changed (diff-scoped), why it was done, how it was
  done, and the evidence (tests/benchmarks). Invoke before committing any
  change, or when the user asks to log their progress. Distinct from /log,
  which captures concepts for the Obsidian vault; /worklog captures work done.
---

# /worklog — commit-scoped work journal

Produce **one entry per intended commit** in `.claude/worklog/`, named
`YYYY-MM-DD-<short-slug>.md` (slug from the change's subject, kebab-case).
The entry is a factual engineering record written for the author's future
self: precise, diff-grounded, no filler.

## Procedure

1. **Scope from git, never from memory.** Run `git diff --stat` and
   `git diff` (staged if anything is staged — `git diff --cached` — otherwise
   the working tree). The diff defines what the entry may claim. If the diff
   is empty, say so and stop.
2. **Recover the why/how from the conversation first.** The reasoning usually
   already happened in-session (a review finding, a traced bug, a design
   discussion). Pull the *actual* motivation and approach from there. Only if
   the why is genuinely absent, ask the user one pointed question — never
   invent a rationale.
3. **Collect evidence.** If tests/benchmarks were run this session, quote the
   relevant result lines. If none were run and the change is testable, run
   `go test ./...` (and `go vet ./...`) now and record the outcome — including
   failures. An entry that says "tests: not run" is honest; an entry that
   implies passing tests without a run is not.
4. **Write the entry** from the template below.
5. **Attribute honestly.** The human wrote the code. Where the assistant
   contributed (found a bug in review, explained a concept, wrote a test on
   request), say so in one clause — the record should show who did what.

## Entry template

```markdown
---
date: YYYY-MM-DD
phase: lexer | parser | compiler | vm | memory | value | tooling
files: [internals/vm/vm.go, ...]        # from git diff --stat
commit: pending                          # replace with short SHA after commit
---

# <imperative one-line subject, like a commit subject>

## What
<Bullet per meaningful change, each anchored to file:line or a hunk.
Diff-scoped facts only — no claims about code the diff doesn't touch.>

## Why
<The problem or goal that motivated this. The trigger: a failing test, a
review finding, a phase milestone. 2–5 sentences.>

## How
<The approach taken and the key decision(s): what was considered, what was
rejected and why, which invariant the change preserves. This is the part
future-you will thank you for. 3–8 sentences.>

## Evidence
<Verbatim result lines: `go test ./...` outcome, benchmark deltas
(benchstat), vet output. Or "tests: not run — <reason>".>

## Stuck points (optional)
<Where progress stalled and what unblocked it — worth recording; these are
the learning moments. Omit the section if there were none.>
```

## Hard rules

- **Precision over prose.** Every claim in *What* must be visible in the
  diff; every number in *Evidence* must come from a command run.
- One entry = one intended commit. If the working tree mixes two unrelated
  changes, say so and suggest splitting; write one entry per split.
- Keep entries under ~60 lines. This is a journal, not documentation.
- Never rewrite past entries; after committing, update only the `commit:`
  field with the short SHA.
