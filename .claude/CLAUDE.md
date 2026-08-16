# Project contract

Compiler-development learning project (typed functional language on a
clox-style bytecode VM, in Go). The human writes all implementation code; the
assistant reviews and teaches, in the main loop and in every subagent —
**never** writes, patches, or scaffolds the implementation itself. Full
contract (role, rules, per-phase review focus): `.claude/output-styles/mentor.md`,
active as this project's output style.

# Work log (required before every commit)

Every change the human intends to commit gets a work-log entry in
`.claude/worklog/`, written via the `/worklog` skill. The entry records
**what changed, why, and how** — precisely, diff-scoped, with evidence
(test/bench output). When the human says they're about to commit, or asks for
a commit message, run `/worklog` first if no entry exists for the change.
Distinct from `/log` (Obsidian concept notes): `/worklog` records *work done*;
`/log` records *ideas learned*.

Bugs found during review or test-writing are tracked separately in
`.claude/ISSUES.md` via the `/issue` skill — run it whenever review or a test
run surfaces a real defect, the same trigger point as `/worklog` but for open
bug state instead of committed work. When a `/worklog` entry fixes a bug that
has an open `ISSUES.md` entry, cite the `ISSUE-NNN` id and close it in the
same session.

# Model routing & token efficiency

The main conversation runs on whatever `/model` selects. Delegated work routes
to the cheapest tier that can complete it, via the agents in `.claude/agents/`.

## Routing table

| Task | Tier | Delegate to |
|---|---|---|
| Find/locate/summarize, file inventories, git-history questions | low | `scout` (haiku) |
| Review the working diff, run tests/benchmarks, routine explanations | medium | `diff-review` (sonnet 4.6) |
| unsafe/arena/GC, VM invariants, architecture, profiled perf work | high | `deep-review` (opus) |
| Anything answerable from context already in the conversation | none | Answer directly — spawning any agent costs more than answering. |

Escalation is one-way and justified: start at the cheapest tier that can
complete the task; escalate only when the cheaper tier reports it is out of
depth (`diff-review` is instructed to say "escalate to deep-review: <reason>").
Don't delegate work the main agent can do in 1–2 tool calls.

## Token-efficiency rules (apply in the main loop AND inside agents)

- **Diff-scope all reviews.** Use `git diff` / `git diff --stat` to find what
  changed; never re-read whole files to discover the change.
- **Search before reading.** Grep/Glob to locate, then Read only the relevant
  line range (offset/limit). No full-file reads of files > ~200 lines unless
  the task genuinely needs the whole file.
- **Never re-read a file just edited to "verify"** — Edit/Write error on
  failure; re-reading is pure token waste.
- **Prefer tool output over re-reading:** `go vet`, `go build`, and test
  failures already name the `file:line`.
- **No perf claim without a measurement** (`go test -bench . -benchmem`,
  `benchstat`, `pprof`) — this is also a correctness rule for this project.
- **Calibrated output length.** Answers should be complete enough that the
  human doesn't need a follow-up, and no longer: short answer + one reason for
  simple questions, the full teaching arc only for genuinely new concepts.
  Never re-explain something already covered this session.
