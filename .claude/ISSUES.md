# Issue tracker

Bugs found during review or test-writing, tracked separately from
`.claude/worklog/` (which is a commit-scoped work journal — see
`.claude/skills/worklog/SKILL.md`). Populated and maintained via the
`/issue` skill. IDs are sequential and permanent: a fix never renumbers an
entry, and a regression gets a new ID that references the old one.

## Open

### ISSUE-022 — `scanner_test.go` still calls the renamed `GetLexme`
- **Location:** `internals/lexer/scanner_test.go:119,525,630,631,638,639,660,661,690`
- **Found:** 2026-08-22, `/worklog` evidence-gathering (`go vet ./...`) for
  the pending panic-mode-error-recovery commit
- **Detail:** `internals/lexer/token.go`'s `Token.GetLexme` was renamed to
  `Token.Lexeme` in the current diff, but the 9 call sites above weren't
  updated (test file, not part of the diff). Fails `go vet`/`go test` for
  the whole `internals/lexer` package.

### ISSUE-021 — `errors.Error` passed by value where `Error()` needs a pointer
- **Location:** `internals/compiler/compiler.go:76` (call site),
  `internals/errors/errors.go:21-28` (`NewError`, returns `Error` not
  `*Error`), `errors.go:30` (`Error()` has a pointer receiver)
- **Found:** 2026-08-22, `/worklog` evidence-gathering (`go vet ./...`) for
  the pending panic-mode-error-recovery commit — first time this gets a
  tracked id; the 2026-08-21 worklog entry's "Stuck points" described the
  same defect in prose only.
- **Detail:** `errors.NewError(...)` returns an `Error` value.
  `(e *Error) Error() string` has a pointer receiver, so a bare `Error`
  value's method set does not include `Error()` — it does not satisfy the
  `error`/`fmt.Stringer` interfaces. `compiler.go:76`'s
  `fmt.Fprintf(os.Stderr, "%s", errors.NewError(...))` therefore never
  invokes `Error()`; `%s` falls back to Go's default struct formatting.
  `go vet` flags this directly: `fmt.Fprintf format %s has arg
  errors.NewError(...) of wrong type go-bytecode-interpreter/internals/errors.Error`.
  Separately, `errors.go:43`'s `fmt.Sprintf(errorString.String())` — found
  in the same file during the same vet run — is a non-constant format
  string (should just be `errorString.String()`, no `Sprintf`).

### ISSUE-018 — `checkKeyword`/`MatchString` misclassify identifiers as keywords by length alone
- **Location:** `internals/lexer/scanner.go:240-246` (`checkKeyword`),
  `scanner.go:264-266` (`MatchString`, formerly `memCompare`)
- **Found:** 2026-08-16, code reading while writing the pending
  scanner-keyword-fix worklog entry (lexer tests can't run to confirm by
  test — see ISSUE-017)
- **Detail:** `MatchString(basePtr, rest, length)` hardcodes
  `unsafe.Add(basePtr, 1)` as its read start, ignoring the `start` offset
  `checkKeyword` was given — for any keyword with `start != 1` (e.g. `FALSE`,
  `FOR`, `FUN`, `THIS`, `TRUE`, all called with `start=2` at
  `scanner.go:205-209,227,229`) it compares the wrong bytes. Separately,
  `checkKeyword`'s condition uses `||` between the pointer-length check and
  the `MatchString(...) == 0` check (`scanner.go:241-242`), where clox's
  original is effectively `&&` (length matches *and* bytes match) — with
  `||`, a total-length match alone is enough to return the keyword
  `TokenType` regardless of the identifier's actual bytes, so e.g. any
  3-byte identifier starting with `f` (not just `for`) would scan as `FOR`.
  Supersedes the "unfinished condition" half of ISSUE-016, whose literal
  `|| (1)` fragment is gone but whose replacement is still wrong.

### ISSUE-002 — VM dispatch has no case for `OP_CONST_LONG`
- **Location:** `internals/vm/vm.go`, `Run()` (the `switch c.OpCode(instruction)`)
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** The compiler and disassembler both know `OP_CONST_LONG` (3-byte
  operand form), but `Run()`'s switch has no `case c.OP_CONST_LONG`. If the
  compiler ever emits it, the VM falls through the switch with no case
  matched, and the instruction stream desyncs (the 3 operand bytes get read
  as fresh opcodes on the next loop iteration).

### ISSUE-004 — `WriteConstant` truncates the constant-pool index to one byte for `OP_CONST`
- **Location:** `internals/compiler/chunks.go:45-54`
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** `WriteConstant` writes `idx&0xff` for the `OP_CONST` case
  unconditionally — if the caller chose `OP_CONST` (1-byte operand) but the
  constant pool already has ≥256 entries, the index silently truncates
  instead of erroring or requiring `OP_CONST_LONG`.

### ISSUE-006 — REPL swallows the file-read error
- **Location:** `internals/repl/repl.go:18`
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** `LoadProgramFromPath` builds an error with `fmt.Errorf(...)` on
  a failed `os.ReadFile` but never returns or logs it — the function
  continues to `compiler.Compile(file)` with a nil/empty `file` slice.

### ISSUE-007 — `main.go` indexes `os.Args` with no length check
- **Location:** `cmd/glox/main.go:17,21`
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** `os.Args[1]` (and `os.Args[2]` for `--path`) is indexed
  directly inside the `switch`. Running the binary with no arguments panics
  with an index-out-of-range instead of printing `cliHelp()`.

### ISSUE-008 — `NewValueArray` sets `Count: 0` despite pre-filling 8 zero values
- **Location:** `internals/value/value.go:15-21`
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** `NewValueArray` allocates `mem.AllocSlice[Value](a, 8)` (8
  zero-valued elements) but initializes `Count: 0`. Currently dead code
  (unused elsewhere), but the mismatch between allocated length and
  reported count is a latent bug for whichever `WriteValueArray`-style
  caller relies on `Count`.

### ISSUE-009 — `vm_bench_test.go` calls `v.Interpret()` with no arguments
- **Location:** `internals/vm/vm_bench_test.go:22`
- **Found:** 2026-07-25, initial scaffold review
- **Detail:** `Interpret` has the signature `Interpret(source string)`; the
  benchmark calls it with zero arguments. Still failing today
  (`not enough arguments in call to v.Interpret`), confirmed via
  `go build ./...` diagnostics this session.

### ISSUE-010 — `makeToken` never sets `Token.Line`
- **Location:** `internals/lexer/scanner.go`, `makeToken`
- **Found:** 2026-08-10, while writing `scanner_test.go` line-tracking tests
- **Detail:** Only `errorToken` populates `Line` (from `s.line`); `makeToken`
  returns `Token{Type, Start, Length}` with no `Line` field, so every
  ordinary token (parens, operators, strings) reports `Line == 0` regardless
  of its actual source line. Confirmed by `TestScanToken_LineTracking`.
  Continuation of the "line never set" half of ISSUE-005 (the "can't advance
  past byte 0" half of that issue is fixed).

### ISSUE-011 — `Compile()` has an unconditional infinite scan loop plus unreachable code
- **Location:** `internals/compiler/compiler.go:17-23`
- **Found:** 2026-08-10, while running `go test ./internals/lexer/...` this session
- **Detail:** `for { token := scanner.ScanToken() }` has no exit condition
  and shadows `token` every iteration, which is also why it fails to
  compile (`declared and not used: token`). The `token := scanner.ScanToken()`
  and the `lexer.ERROR` check written below the loop are unreachable.
  Supersedes ISSUE-001 (the original "`scanner` declared unused" bug is
  gone now that `scanner` is used — but the code that uses it doesn't work).

### ISSUE-012 — stale module import path breaks the build after the `go.mod` rename
- **Location:** `internals/repl/repl.go:5`, `internals/value/value.go:5`,
  `internals/vm/vm_bench_test.go:4-5`
- **Found:** 2026-08-10, while running `go test ./internals/lexer/...` this session
- **Detail:** `go.mod`'s module line was changed (uncommitted) from
  `go-bytecode-interpter` (missing the second "e") to
  `go-bytecode-interpreter`, but these 3 files still import the old typo'd
  path, so `go build ./...` fails with `could not import
  go-bytecode-interpter/...`. `internals/lexer/scanner_test.go` had the same
  issue and was fixed as part of updating it this session.

### ISSUE-013 — `skipWhiteSpace` infinite-loops on tab or `\r`
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace`
- **Found:** 2026-08-10, while writing `scanner_test.go` whitespace tests
- **Detail:** `case '\t':` and `case '\r':` have empty bodies. Go's `switch`
  does not fall through by default, so matching either case exits the
  switch without ever calling `advance()`; the enclosing `for` loop then
  re-`peek()`s the same byte forever. Confirmed by hand — two `go.exe`
  processes had to be killed after `go test` hung; the test
  (`TestScanToken_SkipWhitespace/tabs` and `/carriage_return`) is now
  wrapped in a 2s timeout so it fails instead of hanging the suite.

### ISSUE-014 — `skipWhiteSpace` only skips one whitespace/newline byte per call
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace`
- **Found:** 2026-08-10, while writing `scanner_test.go` whitespace tests
- **Detail:** The `' '` and `'\n'` cases do call `advance()`, but then
  `break LOOP` exits the function immediately after one byte instead of
  continuing to loop — so `"+  +"` scans as `PLUS`, `ERROR` (unexpected
  second space), not two `PLUS` tokens. Confirmed by
  `TestScanToken_SkipWhitespace/spaces` and `/mixed_with_newline`.

## Fixed

### ISSUE-020 — `skipWhiteSpace` comment case no longer skips the trailing newline
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace()` `case '/':`
  (~line 136-146 in the current working tree)
- **Found:** 2026-08-20, `TestScanToken_SkipComment` (pre-existing test)
  failing after running the full `internals/lexer` suite while adding new
  f-string scanner tests (`TestScanToken_FString_*`, which pass)
- **Detail:** The uncommitted working-tree diff (rider fix alongside the
  f-string scanner feature) moved `return` from inside the `case '/':`
  `else` branch to unconditionally after the whole `if/else`. For a real
  `//` comment, the inner loop stops at the trailing `'\n'` (not consumed
  by the loop, which checks `peek() != '\n'`); previously `return` fired
  only on the non-comment (`else`) branch, so the comment path fell through
  to `skipWhiteSpace`'s enclosing `for {}` and looped back to consume the
  `'\n'` via `case '\n':`. Now `return` fires immediately after the comment
  loop, leaving `'\n'` unconsumed. Back in `ScanToken()`, the bare `'\n'`
  matches no `switch` case (only `skipWhiteSpace` handles `'\n'`) and falls
  to `default: errorToken(...)`, producing an `ERROR` token instead of
  resuming normal scanning.
- **Fixed:** 2026-08-20, current working-tree diff. `case '/':`'s `else`
  branch now `return`s directly (not after the whole `if/else`), so the
  comment loop falls through to the enclosing `for` and `case '\n':`
  consumes the trailing newline on the next iteration. Verified:
  `go test ./internals/lexer/... -run TestScanToken_SkipComment` passes.

### ISSUE-019 — `vm_bench_test.go` imports the pre-rename module path
- **Location:** `internals/vm/vm_bench_test.go:4-5`
- **Found:** 2026-08-16, `go test ./...` run before committing the pending
  scanner/opcode-stringer changes
- **Detail:** Imports `"go-bytecode-interpter/internals/compiler"` and
  `"go-bytecode-interpter/internals/memory"` (missing the "e" in
  "interpreter"), left over from the module rename in commit `344b2b0`
  (`module go-bytecode-interpreter` in `go.mod`). Breaks `go test ./...`
  with `package go-bytecode-interpter/internals/compiler is not in std`,
  failing only `internals/vm`; `go build ./...` and every other package's
  tests are unaffected. Not part of the current diff (scanner/lexer +
  compiler opcode-stringer changes) — pure pre-existing breakage.
- **Fixed:** 2026-08-20, current working-tree diff. Both imports now read
  `go-bytecode-interpreter/...`. Verified: `go build ./...` no longer
  reports the missing-package error for this file (the package still fails
  separately on ISSUE-009, an unrelated call-signature bug in the same
  file).

### ISSUE-017 — `Token.String()` passes an `unsafe.Pointer` to `%g`
- **Location:** `internals/lexer/token.go:84`
- **Found:** 2026-08-16, `go vet ./...` run while writing the pending
  scanner-keyword-fix worklog entry
- **Detail:** `fmt.Sprintf("...start=%g...", ..., t.Start, ...)` passes
  `t.Start` (`unsafe.Pointer`) to the `%g` (float) verb. `go vet` fails on
  this, and because `go test` runs vet as part of its build step, this
  single line blocks every test in `internals/lexer` from running at all
  (`FAIL go-bytecode-interpreter/internals/lexer [build failed]`) — `go
  build ./...` itself still succeeds since vet isn't part of a plain build.
- **Fixed:** 2026-08-16, pending commit (worklog TBD). `%g` changed to
  `%p` in `internals/lexer/token.go:84`, matching `t.Start`'s
  `unsafe.Pointer` type; confirmed via `go vet ./internals/lexer/...`
  producing no output.

### ISSUE-016 — `checkKeyword`/`identifierType` broken, blocks `go build ./...`
- **Location:** `internals/lexer/scanner.go:168` (call site),
  `internals/lexer/scanner.go:210-227` (`identifierType`/`checkKeyword`)
- **Found:** 2026-08-16, `/worklog` review of the pending scanner-keyword diff
- **Detail:** `identifierType()` is defined as a method
  `(s *Scanner) identifierType()` but called as a free function at
  `scanner.go:168` (`return s.makeToken(identifierType())` — missing `s.`
  receiver). `checkKeyword` (`scanner.go:222-227`) has an unfinished
  condition — `calcPtrDiff(s.start, s.current) == (start + length) || (1)`
  — where the literal `(1)` is an untyped int in a bool position (a
  leftover fragment from translating clox's C `memcmp`-truthy idiom), the
  `if` body is empty, and the function falls off the end with no `return`.
  `memCompare()` is already defined in the same file but unused — the fix
  wires it in and compares its result against `0`. `go build ./...` fails
  with all three errors simultaneously.
- **Fixed:** 2026-08-16, pending worklog entry (uncommitted). The `s.`
  receiver and the missing `return`s are in place and `go build ./...`
  succeeds. The replacement condition is still wrong, tracked separately as
  ISSUE-018.

### ISSUE-001 — `scanner` declared and not used, blocked `go build ./...`
- **Location:** `internals/compiler/compiler.go:14` (original)
- **Found:** 2026-07-25, initial scaffold review
- **Fixed:** 2026-08-10 → superseded by ISSUE-011. `scanner` is used now
  (inside the infinite loop added since), so this exact error is gone, but
  the code that replaced it is its own open bug.

### ISSUE-003 — `LoadLongConst` uint8 shift overflow
- **Location:** `internals/compiler/chunks.go:76-78`
- **Found:** 2026-07-25, initial scaffold review (flagged directly by `go vet`)
- **Fixed:** 2026-07-25, commit `56b83ec` ("fix uint8 overflow"). Verified:
  `LoadLongConst` now casts each operand to `uint32` before shifting
  (`uint32(b) << 8`, `uint32(c) << 16`), so the shift no longer happens
  inside 8-bit arithmetic.

### ISSUE-010 — `makeToken` never sets `Token.Line`
- **Location:** `internals/lexer/scanner.go`, `makeToken`
- **Found:** 2026-08-10, while writing `scanner_test.go` line-tracking tests
- **Fixed:** 2026-08-16, `2026-08-16-scanner-keywords-and-module-rename`
  worklog entry (uncommitted). `makeToken` now sets `Line: s.line`.
  Verified by reading the current diff; not yet build-verified as the
  file has an unrelated open defect, ISSUE-016.

### ISSUE-011 — `Compile()` has an unconditional infinite scan loop plus unreachable code
- **Location:** `internals/compiler/compiler.go:17-23`
- **Found:** 2026-08-10, while running `go test ./internals/lexer/...` this session
- **Fixed:** 2026-08-16, `2026-08-16-scanner-keywords-and-module-rename`
  worklog entry (uncommitted). `Compile` now loops `ScanToken()` and
  returns on `lexer.EOF`; the `lexer.ERROR` check is reachable inside the
  loop and no longer shadows `token`.

### ISSUE-013 — `skipWhiteSpace` infinite-loops on tab or `\r`
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace`
- **Found:** 2026-08-10, while writing `scanner_test.go` whitespace tests
- **Fixed:** 2026-08-16, `2026-08-16-scanner-keywords-and-module-rename`
  worklog entry (uncommitted). `case '\t', '\r', ' '` now calls
  `s.advance()`.

### ISSUE-014 — `skipWhiteSpace` only skips one whitespace/newline byte per call
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace`
- **Found:** 2026-08-10, while writing `scanner_test.go` whitespace tests
- **Fixed:** 2026-08-16, `2026-08-16-scanner-keywords-and-module-rename`
  worklog entry (uncommitted). The whitespace/newline cases advance and
  fall back to the top of the enclosing `for` loop instead of breaking out
  after one byte.

### ISSUE-015 — comment skip doesn't consume its trailing newline
- **Location:** `internals/lexer/scanner.go`, `skipWhiteSpace` `case '/'`
- **Found:** 2026-08-10, while writing `scanner_test.go` comment-skip test
- **Fixed:** 2026-08-16, `2026-08-16-scanner-keywords-and-module-rename`
  worklog entry (uncommitted). The comment branch still stops at `\n`
  without consuming it, but since `skipWhiteSpace` now loops (ISSUE-014's
  fix), control returns to the top of the `for` and the `case '\n'` arm
  consumes it on the next iteration.

### ISSUE-005 — scanner couldn't advance past the first byte
- **Location:** `internals/lexer/scanner.go`
- **Found:** 2026-07-25, initial scaffold review
- **Fixed:** 2026-07-27, commit `cad8f7f` ("scanner work"). `start`/`current`
  are `unsafe.Pointer` into the source now, with `advance()` moving the
  pointer forward — confirmed scanning multi-character sources works via
  `TestScanToken_MultiTokenSequence`. The "`Token.line` is never set" half of
  this issue is still open — see ISSUE-010.
