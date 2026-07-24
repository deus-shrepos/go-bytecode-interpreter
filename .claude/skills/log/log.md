---
name: log
description: Capture the concept, decision, or gotcha from the current work as a single atomic Obsidian note — categorized via frontmatter properties and wired into the concept graph with wikilinks and domain MOCs. Invoke after a meaningful change or insight.
---

# /log — capture work as an interlinked Obsidian note

When invoked, turn the idea currently under discussion into **one atomic note** in the
knowledge vault: categorized by properties, linked to related concepts, and attached to a
domain map. The goal is notes that import into Obsidian with zero cleanup and join the graph
automatically.

## Where notes live

Write under `$VAULT` (default `docs/knowledge/`; if the user has pointed Claude Code at an
Obsidian vault via `additionalDirectories`, use that path instead):

```
$VAULT/
  notes/        # atomic notes — one concept each
  maps/         # MOC hub notes — one per domain
  _schema.md    # the property schema; single source of truth, never deviate from it
```

## Procedure (every invocation)

1. **Classify.** Choose exactly one `type` (`concept` | `decision` | `gotcha` | `pattern`)
   and exactly one primary `domain` from the taxonomy below. Add finer-grained `tags`.
2. **Find connections.** Grep/Glob `notes/` by title and tag for concepts this relates to.
   Express each as a `[[wikilink]]`. If a *structurally central* concept is referenced but has
   no note yet, create a one-line **stub** (`status: seedling`) so the link resolves and the
   graph stays connected. Stub load-bearing concepts only — never trivia.
3. **Write the note** from the template. Filename = the concept title in sentence case;
   Obsidian resolves wikilinks by title, so the title *is* the address
   (e.g. `notes/Operator precedence in a Pratt parser.md`).
4. **Attach to a map.** Set `up: ["[[<Domain> MOC]]"]`. If that MOC doesn't exist, create it
   from the MOC template. The MOC's membership is a query, so you never hand-edit a list of links.
5. **Keep it atomic.** One idea per note, body under ~200 words. A `seedling` is fine and
   expected — do not polish, do not pad. Two ideas means two notes, linked.

## Taxonomy — `domain` (pick one primary)

- `compiler/frontend` — scanning, parsing, error recovery
- `compiler/semantics` — type checking, scope resolution, gradual typing
- `compiler/backend` — bytecode emission, codegen
- `runtime/vm` — dispatch loop, stack discipline, calling convention
- `runtime/memory` — value representation, GC, arenas
- `performance` — profiling method, optimization, mechanical sympathy
- `plt/theory` — type theory, Curry–Howard, intuitionistic logic, semantics
- `lang/go` · `lang/c` — language-specific lessons

`type` is orthogonal to `domain`: a single bug can be a `gotcha` in `runtime/vm`. `tags`
carry the fine detail (`pratt`, `nan-boxing`, `escape-analysis`) for cross-cutting queries.

## Atomic note template

````markdown
---
title: Operator precedence in a Pratt parser
type: concept
domain: compiler/frontend
tags: [lin, pratt, precedence]
status: seedling          # seedling → budding → evergreen
created: 2026-06-24
source: "Crafting Interpreters §17.6"
up: ["[[Parsing MOC]]"]
related: ["[[Recursive descent]]", "[[Binding power]]"]
---

## Idea
<the concept in 2–4 plain sentences>

## Where it bit / why it matters
<the concrete trigger: the bug, the design force, the decision and its rationale>

## Connections
- [[Recursive descent]] — Pratt is the expression-level refinement of it
- [[Binding power]] — the numeric mechanism that encodes precedence/associativity

## Open question
<optional — what's unresolved; keeps a seedling honest>
````

**Why wikilinks appear twice** (frontmatter `related:` *and* the Connections body section):
this is deliberate, not redundant. Bases reads only frontmatter, so `related:` makes the links
clickable in a Base. The graph view and Dataview read both, and the body version stays readable
as prose. Put the same links in both places.

## MOC template (`maps/<Domain> MOC.md`)

The map does not contain a hand-curated list of links. It contains a query that gathers every
note tagged with its domain, so the map grows itself as you `/log`:

````markdown
---
title: Parsing MOC
type: moc
domain: compiler/frontend
tags: [moc, lin]
created: 2026-06-24
---

# Parsing — Map of Content

> Auto-assembled from note properties. Nothing below is hand-maintained.

```dataview
TABLE status, tags, source
FROM "docs/knowledge/notes"
WHERE domain = this.domain
SORT status DESC, file.name ASC
```
````

**Bases alternative** (same data, GUI, no query language): `Bases: New base`, filter
`domain is compiler/frontend`, add columns `status`, `tags`, `source`. The `.base` file stores
only the view; the notes remain the source of truth.

## Hard rules

- **One idea per note.** Atomicity is what makes the graph meaningful.
- **All queryable data in frontmatter.** Bases ignores inline `key:: value` fields and body text.
- **Never invent properties outside `_schema.md`.** Five consistent fields beat twenty ad-hoc
  ones; inconsistent keys produce empty queries.
- **ISO dates; `tags` as a YAML list (plural).** Match Obsidian 1.9+ property conventions.
- The system earns its keep only as a byproduct of real work. If maintaining it ever costs more
  than the work it documents, the schema is too big — cut it, don't feed it.
