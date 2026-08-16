---
name: issue
description: >
  Record a bug found during code review or test-writing as a tracked entry
  in .claude/ISSUES.md, or close an entry once a commit fixes it. Invoke
  whenever review or a test run surfaces a real defect, or when the user
  asks to log/close/list issues. Distinct from /worklog, which journals
  work done per commit; /issue tracks open-vs-fixed bug state across
  sessions.
---

# /issue — bug tracker

`.claude/ISSUES.md` is a standing list of known bugs, split into `## Open`
and `## Fixed`. Unlike `.claude/worklog/`, entries are long-lived and get
their status updated in place — but their *content* (ID, location, found
date/context, detail) is never rewritten once written, matching the
worklog's "never rewrite past entries" spirit applied to the parts that
should stay historical.

## Procedure

### Recording a new issue

1. **Ground it in something concrete.** A bug must trace to a specific
   `file:line` (or function) and a one-sentence mechanism — "this looks
   wrong" is not an entry; "case '\t' has no advance(), so the loop never
   terminates" is.
2. **Assign the next ID.** `grep -o 'ISSUE-[0-9]*' .claude/ISSUES.md | sort
   -V | tail -1` and increment. IDs are assigned in discovery order and
   never reused or renumbered.
3. **Write the entry** under `## Open`, using the template below.
4. **Note supersession, if any.** If this issue replaces or refines a
   previous one (e.g. a bug found while fixing another bug), say so in
   Detail and add a matching note to the old entry pointing forward.

### Closing an issue

1. Find the entry under `## Open` by ID.
2. Move it to `## Fixed`, appending a line: `**Fixed:** <date>,
   <commit-sha>` (or `<worklog-slug>` if the fix is written but not yet
   committed).
3. Leave Location/Found/Detail exactly as written — only the section and
   the appended Fixed line change.
4. If the fix is partial (only some of the Detail's claims no longer hold),
   don't mark it Fixed — split it: close what's actually fixed, open a new
   ID for the remainder, and cross-reference both in Detail.

### Cross-linking with `/worklog`

When a worklog entry's `## What` fixes a bug that has an open `ISSUES.md`
entry, cite the `ISSUE-NNN` id in that entry's `## Why` or `## How`, and
close the issue via this skill in the same session — don't let the fix and
the tracker drift apart.

## Entry template

```markdown
### ISSUE-NNN — <one-line summary>
- **Location:** `path/to/file.go:LINE` (or function name)
- **Found:** YYYY-MM-DD, <what surfaced it — review, a failing test, a
  worklog entry slug>
- **Detail:** <the mechanism — what the code does and why it's wrong, 1-4
  sentences. Concrete enough that fixing it doesn't require re-deriving the
  bug.>
```

Closed entries add one more line directly after Detail:

```markdown
- **Fixed:** YYYY-MM-DD, <commit-sha-or-worklog-slug>
```

## Hard rules

- One entry per distinct bug. Don't bundle two unrelated defects into one
  ID even if they're in the same function — track them independently so
  one can close without the other.
- Never delete an entry, and never renumber IDs. A regression or a
  follow-on bug gets a fresh ID that references the old one.
- Don't mark something Fixed without checking the current code (or a
  passing test) — a commit message claiming a fix is a hypothesis, not
  verification.
- Keep Detail factual and diff/code-grounded, same standard as `/worklog`'s
  "every claim must be visible in the diff."
