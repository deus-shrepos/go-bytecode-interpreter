---
name: anki
description: >
  Turn a significant change, a debugging session that surfaced a non-obvious
  concept, or an explicit chat request into atomic Anki flashcards in
  .learning/deck.tsv. Invoke after a /worklog entry lands, after a
  debugging thread resolves a real misunderstanding, or on request. Distinct
  from /worklog (work done) and /issue (bug state) — /anki captures durable
  conceptual knowledge, not project state.
---

# /anki — flashcard capture

Turns hard-won understanding into flashcards for spaced repetition: one
card, one atomic, testable fact. Skip mechanical/syntax trivia a reference
doc already covers well — capture the things that took a debugging session,
a wrong first guess, or a review comment to actually understand: invariants,
gotchas, the "why" behind a design, root causes of bugs.

## When this fires

- Proactively suggested (not auto-run) right after a `/worklog` entry is
  written, when a debugging session lands on a root cause that wasn't
  obvious going in, or when an `/issue` entry is closed and its root cause
  is conceptually reusable beyond this one bug.
- On explicit request — "make this a card", "let's Anki this", "I want to
  remember that."
- Always as a proposal step: draft candidate cards and show them to the
  human before appending. A bad card compounds review cost over months; no
  card should reach the deck unreviewed.

## Procedure

1. **Identify the source** — a worklog slug, an ISSUE id, or the immediate
   conversation/debugging transcript. Ground every card in that source; no
   fact goes on a card unless it traces to the diff, the transcript, or a
   canonical text (see `mentor.md`'s source list: Crafting Interpreters,
   Engineering a Compiler, Dragon Book, TAPL, Software Foundations, Drepper,
   Agner Fog, Systems Performance).
2. **Split into atomic candidates** — one concept per card. Reject
   candidates that are pure syntax/mechanical recall (e.g. "what keyword
   declares a variable"). Cards build understanding of invariants and
   mechanisms, not rote API memorization — same "review not author" spirit
   as the rest of this project's assistant contract.
3. **Check for duplicates** — if `.learning/deck.tsv` doesn't exist yet,
   skip this step. Otherwise grep its Front column for the concept; if
   already covered, skip, or if the old card was wrong/incomplete, tag its
   row `superseded` and add a new one rather than editing in place.
4. **Draft Front and Back** — Front is a question, not a statement. Back is
   the answer plus the one sentence of "why" that makes it stick. Cite a
   canonical source only when the transcript/diff/worklog actually names or
   quotes one (e.g. a comment referencing clox, a worklog line citing a
   text) — don't infer a citation from general project context (e.g. "this
   is a clox port, so cite Crafting Interpreters") when nothing in the
   source material actually said so. No citation is better than a guessed
   one.
5. **Show drafts to the human** for approval or edits before writing
   anything. If running non-interactively with no human available to
   approve synchronously, treat the drafts as pending: report them in full
   and do not append to `.learning/deck.tsv`.
6. **Append approved cards** to `.learning/deck.tsv`. If the file doesn't
   exist yet, create it with the Anki header directives first (see below).

## Card template

First-time file creation — header directives Anki's plain-text importer
requires:

```
#separator:tab
#html:false
#tags column:3
Front	Back	Tags
```

Each card is one tab-separated row appended after the header:

```
Why does the arena allocator bump-pointer instead of freelist per object?	Bump allocation is O(1) and cache-friendly because it never walks a free list; the arena is freed in bulk when the enclosing scope ends. (Engineering a Compiler ch. on memory management)	memory concept worklog:2026-07-25-initial-scaffold
```

Tags are space-separated within the third column: phase
(`lexer`/`parser`/`compiler`/`vm`/`memory`/`value`/`tooling`), card type
(`concept`/`debug`/`gotcha`), and an optional source cross-reference
(`worklog:<slug>` or `issue:ISSUE-NNN`).

## Hard rules

- One testable fact per card. If the draft needs "and," it's two cards.
- Never invent an answer — every Back must be traceable to code, a
  debugging transcript, or a cited source.
- Never silently write cards — always show drafts for approval first.
- Append-only. Corrections become new rows tagged `superseded` on the old
  one, mirroring `/issue`'s "never delete or renumber" rule — Anki tracks
  review history against note content, so editing a live card's Front/Back
  after it's been reviewed corrupts its scheduling history.
- Skip trivia a reference doc already covers well; prioritize the
  non-obvious.
