# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

legmacs is a terminal text editor written in and scriptable with
[let-go](https://github.com/nooga/let-go) (a Clojure-dialect Lisp with a Go
VM). It's Emacs-flavored: chord-based keys (`C-x C-s`), a minibuffer, a
kill-ring, major modes. There is no separate plugin language — the built-in
commands and keymap are defined with the exact same API a user's own config
uses.

The sibling checkout `../let-go` is the language/runtime itself (not part of
this repo). If a builtin's behavior is unclear, its Go source is more
authoritative than guessing: `pkg/rt/lang.go` (builtins), `pkg/rt/core/*.lg`
(stdlib written in let-go), `pkg/compiler/eval.go` (`eval`/`read-string`/
`load-string`/`read-all-string`), `pkg/rt/term.go` (the `term` namespace —
raw mode, ANSI, key reading).

## Commands

Run it (from this directory, so namespace resolution works — see Gotchas):

```sh
lg main.lg [file...]
```

Or via the installed wrapper, which works from any cwd and picks up
`~/.config/legmacs/legmacs_init.lg`:

```sh
bin/legmacs [file...]
```

Run the full test suite:

```sh
lg test/run.lg
```

Run a single test namespace (faster iteration than the full suite):

```sh
lg -e "(require '[test :refer [run-tests]] '[test.region-test]) (run-tests 'test.region-test)"
```

(`run-tests` with no arguments tests only `*ns*`, like Clojure's, so name
the namespace -- a bare `(run-tests)` reports 0 tests and success.)

There's no separate build/lint step — `lg` interprets `.lg` source directly.
`test/run.lg` is a hand-maintained list of every test namespace; adding a
new `test/*_test.lg` file means adding its namespace to that `:require`
list or it silently won't run.

## Architecture

**The pipeline is a pure state machine.** Everything above `main.lg` is
`(fn [state] -> state)` or `(fn [state] -> string)` — no I/O, no atoms for
the document itself. `main.lg`'s `loop`/`recur` is the only place that
touches the terminal:

```
term/read-key --(legmacs.keys)--> chord string
  --(legmacs.dispatch, consulting legmacs.keymap + legmacs.modes)--> command
    --(legmacs.commands, built on legmacs.buffer)--> new state
      --(legmacs.render)--> one ANSI string --(term/write)--> screen
```

This is what makes the whole editor unit-testable without a TTY (see
`test/`) — buffer edits, key parsing, dispatch, even rendering are just pure
functions checked against expected data.

**Scripting = the same registries the built-ins use.** `legmacs.keymap`
holds two atoms: a command registry (name → fn) and the keymap tree
(chord-path → command). `defcommand` defines a normal function *and*
registers it; `bind-key!` attaches a chord sequence to a registered command.
`legmacs/bindings.lg` (the defaults) and a user's `legmacs-init.lg` call the
exact same functions — there's no privileged built-in path.

**Modes shadow the global keymap key-by-key, not sequence-by-sequence.**
`legmacs.modes` lets a mode add a leaf under an existing global prefix
(e.g. let-go-mode's `C-x C-e`) without hiding the rest of that prefix's
global bindings — `legmacs.dispatch` walks *all* applicable keymaps in
parallel at each chord, not one keymap chosen up front. A mode can also
supply `:highlighter` (pure per-line syntax coloring, called fresh every
frame for visible lines only) or, for a language with constructs that
cross line boundaries, `:highlight-line` + `:highlight-carry` (see the
highlighting paragraph below), and `:after-command` (runs after every
dispatch while that mode is active — used for paren-matching). `legmacs/
lisp_syntax.lg` is a single-pass bracket/string/comment scanner shared by
paren-matching, auto-indent, expand-region, and auto-pairing in
`legmacs/modes/letgo.lg`; `legmacs/modes/markdown.lg` is a second,
independent mode (highlighting only) built the same way; and `legmacs/
modes/prog.lg` generalizes that shape into one spec-driven per-line scanner
(comment markers, string delimiters, keyword/type/constant word sets)
behind major modes for Go/JS/Python/C/shell/Rust/JSON/YAML/TOML/Lua/Ruby/
SQL/Dockerfile/CSS/HTML/Zig/Java/Kotlin/Swift/C#/PHP/Makefile — its
`register-prog-mode!` is both how the built-ins
register and the user-facing one-call way to add a language. `:block-comment`
is checked before `:line-comments` in both this scanner and
`legmacs/prog_syntax.lg` below, since a block-open marker can be a
strict-prefix superstring of the line-comment marker (Lua: `--[[` vs `--`).

**Paren-matching, auto-pairing, and auto-indent are reusable minor modes,
not let-go-only.** `legmacs/prog_syntax.lg` is a sibling of
`legmacs/lisp_syntax.lg` — the same full-buffer bracket/string/comment scan,
but driven by a language `:spec` map (the exact one `register-prog-mode!`
already builds a highlighter from: `:line-comments`, `:block-comment`,
`:string-delims`, plus purely-structural `:brackets`/`:indent-width`
knobs) instead of Lisp's fixed rules. `legmacs/modes/structural.lg` builds
three ordinary minor modes on top of it — `:paren-match`, `:electric-pair`,
`:auto-indent` — that look up `(modes/mode-syntax-spec (:mode state))` at
call time and degrade to plain behavior (self-insert, plain newline, clear
the match) when the active buffer's major mode has none. `register-prog-mode!`
stores its spec as the mode's `:syntax-spec`, and `legmacs.modes/switch-to-mode`
auto-enables all three whenever the mode being switched to has one — so
every language in the pack gets all three "for free" the same way `.lg`
files already had them, with zero extra wiring per language. `:let-go`
deliberately doesn't declare a `:syntax-spec` and keeps its own
hand-written versions in `legmacs/modes/letgo.lg` instead, since a real
Lisp reader has structure (character literals like `\(`) this generic
scanner doesn't model — merging the two would either lose that or bloat
the generic scanner with Lisp-only rules every other language pays for.

**Major vs minor modes are one registry, distinguished only by which buffer
slot holds them.** A buffer's `:mode` is its single major (content) mode;
`:minor-modes` is an ordered vector layering *over* it (minor shadows major
shadows global). `legmacs.modes/active-modes` returns that order (minors,
then major); `dispatch`'s keymap layering, its `:after-command` chain, and
`render`'s highlighter/`:status` compositing all iterate it — so adding a
mode capability means extending those iterations, not special-casing major
vs minor. Two hooks exist for modal editing (CRUTCH, `legmacs/modes/
crutch.lg`): a mode's `:keymap` may be a `(fn [state] -> keymap)` (bindings
that depend on buffer-local state — CRUTCH's normal/insert/visual keymaps
off `:crutch-state`), and `:suppress-self-insert?` (bool or fn of state)
makes unbound printable keys inert instead of self-inserting. CRUTCH's three
vi states are ONE minor mode carrying a `:crutch-state` field, not three
registered modes — modeling mutually-exclusive phases as separate modes
would re-introduce a by-convention "exactly one active" invariant that the
single-field approach makes structural. `dispatch` also stamps each pressed
chord onto the threaded state as `:last-chord` (so a mode can observe the
raw key, not just resulting state), and modes can supply `:status-right` (a
right-aligned mode-line lighter, composed by `render` alongside the
left-aligned `:status`). `legmacs/modes/keycast.lg` is a pure *observer*
minor mode built on exactly those two: empty keymap, `:after-command` logs
`:last-chord`, `:status-right` renders the log — a good template for any
"watch keystrokes / show HUD" feature.

**Indentation is one pure function plus data.** `legmacs.indent/
indent-column` answers "what column should this line start at" for a
language spec, and everything electric (RET, TAB, typing a closer, opening
a bracket pair out into a block -- all in `legmacs.modes.structural`) is a
call to it plus `set-line-indent`. Rules live in the language's own spec
under `:indent`: `:style :absolute` counts bracket depth (C-likes, and
self-correcting), `:style :relative` reads the previous non-blank line and
adjusts by keyword rules (shell/Ruby/Lua, and Python's offside rule via
`:open-suffix [":"]`), `:fn` is the escape hatch. Adding a language's
indentation should mean adding data to its spec, not code here.
let-go-mode deliberately doesn't go through the rule engine -- Lisp aligns
under the enclosing form's first argument (`legmacs.modes.letgo/
indent-column-for`), which is a different shape of answer, not a different
rule set -- but it does share `legmacs.indent`'s edit half, so TAB behaves
identically everywhere.

**Highlighting carries state across lines, and render is what threads
it.** A mode's `:highlight-line` is `(fn [carry line] -> {:spans :carry})`;
the carry is opaque to render and private to the mode (prog: the block
comment or multi-line string still open; let-go: `:string`; markdown:
`:fenced`). `render/rows-highlight-spans` folds the carry from line 0 down
to the top of the viewport, then styles the visible rows in order -- which
is why highlight spans are computed once per window per frame in
`window-frame` and handed to `text-row`, instead of each row deriving its
own. Folding the prefix is only affordable because of `:highlight-carry`,
the state-only half: it answers "could this line change anything?" with a
native substring search (`legmacs.modes.prog/make-carry-advance`, and
`legmacs.lisp-syntax/ends-in-string?` for let-go, which is the cheap
scan-free version of the same question `scan` answers) and hands the carry
straight through when not. Skipping that -- deriving the carry by running
the full scanner over every line above the viewport -- costs ~10x and
shows up as lag scrolling deep into a large file. A plain stateless
`:highlighter` still works and is auto-lifted into the same shape by
`legmacs.modes/line-highlighter`, so a mode with nothing to carry (help,
repl) needs no changes.

**`legmacs.render` is a per-column fg/bg span compositor**, not a
single-highlight hack — region selection, paren-match, and syntax colors
are independent span lists merged into one style array per row, so they can
all show up on the same line without stepping on each other.

**Multiple buffers wrap the flat state without changing it.**
`legmacs.dispatch`/`render`/`commands` still only ever see one flat state —
that was true before multi-buffer support existed and still is.
`legmacs.buffers` holds a "workspace" (every buffer, plus what's genuinely
global rather than per-buffer: echo message, minibuffer/overlay, pending
prefix key, the one shared clipboard). `bufs/current` merges the active
buffer with those into the flat shape dispatch expects; `bufs/put-current`
splits the result back apart. The one thing a normal command can't express
this way — "switch to a different buffer" — is just an ordinary
`:buffer-command` field on the state it returns (e.g. `{:op :switch :id
2}`); `put-current` reads it, applies it to the workspace, and clears it.
When folding shared fields back, `put-current` must `get` each one
explicitly rather than `select-keys` + `merge` — a command that `dissoc`s a
shared key (instead of `assoc`-ing it to nil) would otherwise silently fail
to clear it on the workspace (this was a real, previously-shipped bug).
The same trap applies to the per-window view slice: `legmacs.windows/
view-of` uses explicit `get` for the same reason (`scroll-to-fit`
*dissoc*es `:recenter`/`:scroll-anchor`).

**Blocking work is two shared fields, not a per-feature hook.** Main-thread
work that must paint first uses `:pending-task` — a fn or vector of fns,
`bufs/arm-task` to append, `bufs/run-pending-task` to run the head *in
place of reading a key*. I/O that can run off-thread uses `:async-jobs`:
`bufs/spawn-task` starts a `(future …)` immediately with an `io` thunk
(no editor state) and a `:then` `(fn [state outcome] -> state)` applied
later by `bufs/drain-jobs` on the main thread *before* the next frame is
painted — drain-then-paint, not the other way around, or a finished job
sits on screen until the next keystroke. `bufs/loop-wait` is the idle
decision (`:quit` / `:pending-task` / `:poll` / `:read-key`) so that
order is testable without a TTY. Goroutines never touch buffers. Outcome
is `{:ok v}` / `{:err e}` because let-go's future delivers nil on throw.
While jobs are in flight the loop peeks `term/key-pending?` and sleeps
20ms instead of parking in `read-chord`, so typing continues. Idle is
still a blocking read. C-g (`bufs/discard-jobs`) drops results without
killing HTTP/`os/sh`. Named functions, not inline `fn` literals, when
building the closure inside a conditional — see the AOT-hoisting gotcha
below.

**Windows (splits) are the same wrapping trick one level up.** The
workspace holds a window tree (`legmacs.windows` — pure data + layout
geometry, knows nothing of buffer contents or rendering): leaves show a
buffer through a per-window `:view` (cursor/scroll — Emacs window-point,
so two windows on one buffer diverge), split nodes stack `:below` or sit
`:beside` with proportional integer weights (a terminal resize re-flows
the same shape). `bufs/current` flattens the *active window's*
buffer+view; window commands are just more `:buffer-command` ops
(`:split`, `:other-window`, `:delete-window`, `:delete-other-windows`),
the buffer-switching ops retarget the active window, and `:kill`
re-points any window showing the dead buffer. `:current` (the active
buffer id) is re-derived from the active window after every fold — never
edited independently (`main.lg`'s `build-workspace` focuses buffer 0 via
a real `:switch` op for exactly this reason). Rendering is the one layer
that sees all windows per frame: `render/prepare-workspace` runs the
scroll pass per window (full scroll-to-fit only for the active one;
inactive views are just clamped — a window you're not in mustn't scroll
on its own), then `render/workspace-frame` draws each rectangle with its
own mode line (dimmed when inactive) and fills `│` divider columns
between side-by-side windows. `windows/layout` partitions its area
*exactly* (children plus dividers = full width/height), preserving the
every-cell-written-no-clear frame invariant. The flat single-state
`render/frame` still exists and must stay byte-identical to a one-window
workspace frame — there's a test asserting exactly that.

**Generic eval-and-replace (`C-c C-v`) is an `:async-jobs` client.**
`legmacs.modes.letgo/eval-replace-sexp` discovers the region or structural
form at point, highlights it, snapshots its source, and evaluates it through
`bufs/spawn-task`. Completion verifies that the source span is unchanged
before replacing it. Ordinary values use `pr-str`; functions that need
source-aware application return `replacement-result` with a registered
result-kind handler. The same seam carries every other eval command: all four build one
request (`:op` `:eval`/`:load`, `:source`, `:forms`, `:state`, `:backend`,
and filename/directory/line/column/namespace `:context`) and hand it to
the buffer's backend. Which backend is `letgo/eval-backend-for`: the
first registered resolver that claims the buffer, else the global
`current-eval-backend` -- resolvers run every frame (the lighter), so they
must stay string checks, no filesystem. With `:in-process`,
`C-x C-e`/`C-j`/`C-c C-e` still eval synchronously on the main thread (so
editor-scripting forms take effect before the next frame); any other
backend makes them `spawn-task` jobs too. Remote backends return
`printed-result` (text the server printed, plus captured output) since
the value never existed in this VM. Completion
(`legmacs.modes.letgo-complete`, TAB/M-TAB) routes the same way through a
per-backend `completers` registry, and so do eldoc, doc and `M-.`
(`legmacs.modes.letgo-lookup`, a `lookups` registry of `(fn [request] ->
info-map)`). let-go-mode's single `:after-command` now runs
paren-matching and then `letgo/after-command-hooks` (eldoc is one), each
in a try so a failing hook can't break editing. eldoc runs every key, so
it scans only back to the enclosing top-level form (a `(` in column 0)
and reads the ns form from the buffer's first 80 lines; remote answers
come from a short-TTL cache, fetched by a quiet `spawn-task`. Goto uses
a `:visit` buffer-command (show buffer by `:id`/`:filename`/`:name`, or
open `:state`, at `:row`/`:col`) and a shared `:xref-stack` for `M-,`.

`legmacs.nrepl` is the nREPL backend: one connection per project root
(buffers with a `:filename` under it resolve to it; fileless buffers like
`*scratch*` never do), one blocking bencode round trip per request,
serialized by a channel-token lock, inside the job's future -- so no
long-lived reader goroutine exists for `loop-wait` to keep polling.
`legmacs.jack-in` starts servers from data launch configs. let-go
interop can't reach `cmd.Process`, so a server runs under `sh -c` that
writes a pid file, redirects output to a log (an unread pipe would fill
and block it) and `exec`s the command; the editor holds its stdin pipe
open (`lg -n` exits on EOF) and stops it with SIGTERM + closing stdin.
Liveness is `ps -o stat=`, not `kill -0`: an exited child is a zombie
until Waited, and `kill -0` says a zombie is alive. `bufs/add-exit-hook!`
hooks run after `main.lg` restores the terminal; both namespaces use it
so no server outlives the editor. The editor picks the port
(`os/free-port`) rather than reading `.nrepl-port`, which Babashka
doesn't write for port 0. Backends never touch live buffer state.

`legmacs.vibe/vibe` uses that protocol rather than owning a special key
command. Under the dynamic `letgo/*eval-replace-context*` it derives its
own buffer marker context and returns a tagged `:vibe` result; outside it,
vibe remains an ordinary blocking string-returning function. The user
message includes a live catalog of canonical let-go namespaces
(`string/` not `clojure.string/`) and the file's `:require` aliases; the
model can also call `list-namespaces` / `ns-publics` / `var-doc` against
the running VM (OpenAI goes through `/v1/responses` so tools can sit
next to `:reasoning-effort`, default `:medium`; a `chat/completions` URL
selects the older shape). Its registered handler rewrites JVM-Clojure
prefixes, splices the string, and inserts missing `(:require ...)` entries
into the ns form in one undo step. The system prompt also requires a
docstring on every `defn` / `defn-` / `defcommand`.

Full per-file breakdown and the complete default keymap are in
[README.md](README.md); read it before making structural changes.

## Gotchas specific to this codebase / let-go

- **No forward references at the top level.** A `defn` body referencing a
  symbol not yet defined earlier in the same namespace fails to compile,
  even though it's a Lisp — order top-level `def`/`defn` bottom-up, or use
  `(declare foo)` above the first use for genuine mutual recursion.
  Self-recursion inside one `defn` is fine.
- **Namespace resolution is cwd-relative, not script-relative.** `lg
  main.lg` only resolves `(require 'legmacs.buffer)` etc. if run from this
  directory (or with `LG_SOURCE_PATHS` including it) — that's exactly what
  `bin/legmacs` sets up so it works from anywhere.
- **`(require 'legmacs.main)` runs the editor.** `main.lg` calls `(main)`
  as its last top-level form, so requiring that namespace from a test or
  scratch script starts the real interactive loop. To drive the pipeline
  headlessly (e.g. to test something end-to-end without a real terminal),
  write a throwaway script that requires `legmacs.buffers`/`dispatch`/
  `render`/`bindings`/`commands`/`modes.letgo` directly and reimplements
  the small `run`/`build-workspace` loop inline — `term/read-key` and
  `term/write` work fine on a plain pipe, no pty required; only
  `term/size` and `raw-mode!` need a real terminal, and `render/frame`
  takes explicit cols/rows so you can sidestep that too.
- **`load-string`ing text that opens with `(ns foo)` relocates the current
  namespace permanently — including from inside a function call.** Since
  legmacs evaluates in its own live namespace (`legmacs.main`), one
  `eval-buffer` on any ordinary `.lg` file would otherwise leave `buf/`,
  `km/`, `dispatch/`, `render/` and `vibe` unresolvable for the rest of the
  session — the next `C-x C-e` fails with "Can't resolve ... in this
  context". `legmacs.modes.letgo/preserving-ns` (`(ns-name *ns*)` before,
  `in-ns` after, via the reify-outcome pattern so it survives a throw)
  wraps `eval-buffer` for exactly this. `C-x C-e`/`C-j` and the `*repl*`
  buffer are deliberately left unwrapped: an explicit `(in-ns 'foo)` there
  *should* move you.
- **`(def x (load-string "..."))` as a script's own top-level form breaks
  resolving the *next* top-level form.** Reproducible with plain `lg -e`,
  nothing legmacs-specific, and doesn't affect legmacs itself since
  `eval-buffer` only ever calls `load-string` from inside an
  already-compiled, already-running function — never as a top-level form.
  Only bites you experimenting at a REPL.
- **Be careful writing control-character escapes into `.lg` source through
  file-editing tool calls.** A four-hex-digit unicode string escape for a
  control code (backslash, letter u, then the hex digits) can get decoded
  into the literal raw byte before it ever reaches disk, leaving an actual
  invisible control character in the source file instead of the escape
  text you meant to write (confirmed firsthand while drafting this very
  file — `od -c path | head` will show a stray octal escape like `\177` if
  it happens). `legmacs/keys.lg` sidesteps this entirely by building
  control-byte strings with `(char code)` at runtime rather than writing
  the escape literally in source; follow that pattern rather than typing
  a unicode string escape for a control character directly into a file.
- **Don't return an `fn` literal from a conditional branch if it captures a
  local.** let-go's AOT Go-lowering (see `build/`) hoists such a closure out
  of the `if`/`when`/`cond` and emits it as the enclosing function's
  unconditional `return`, leaving an empty `if {} else {}` behind -- every
  branch then returns that one closure, sometimes over a variable the taken
  branch never set. It is silent: `go build` only catches the cases where a
  dead temporary survives ("declared and not used"), and the CI native-build
  smoke step is the only thing that looks. Upstream bug:
  nooga/let-go#766. Non-capturing closures are fine (they get lifted to
  top-level fns). Until it's fixed, build each closure in its own small
  named function so the branches contain *calls*, not `fn` forms -- see
  `legmacs.modes/lifted-highlighter` and
  `legmacs.modes.prog/marker-gated-advance`. To check a suspicious lowering:
  `./build.sh`, then look for `if vm.IsTruthy(...) {\n} else {\n}` followed
  by `return rt.BoxNativeFn(` in the generated file (the same pattern with a
  `v = vm.NIL` after it is just a `cond`'s `:else nil`, and is fine).
- **`count` and `subs` on a *string* are O(n), and `string/index-of` from
  an offset is too.** let-go strings are indexed by rune, so all three walk
  the string from the beginning; on a 96KB buffer `(count text)` measured
  ~35us. A scanner that calls any of them once per character is therefore
  quadratic in file size, which is not a subtle slowdown: one RET in a
  commented 4,000-line Go file took **61 seconds** before
  `legmacs.prog-syntax` was moved onto a char vector (`(vec (seq text))`
  once, `nth` after that, markers pre-converted to char vectors, and the
  next newline found by walking rather than by `string/index-of`). The same
  keystroke is ~65ms now, ~10ms in a 700-line file. Any new whole-buffer
  scan must follow that shape: convert once, pass the length in, never
  reach for a string operation inside the loop. Per-*line* code can be
  relaxed about it (lines are short), but that's the only exception.
- **A per-keystroke pass shouldn't build what nobody reads.** `scan` in
  both syntax namespaces returns a bracket-pair map and a span vector; the
  things that run on every key press want one integer or one stack, so
  they have their own lean passes instead (`prog-syntax/open-stack-at` and
  `depth-at`, `lisp-syntax/open-stack-at*`, `match-for-closer` and
  `point-in-string-or-comment-at?` in both, `lisp-syntax/ends-in-string?`
  for the highlighter carry). They're pinned to `scan`'s answers by tests
  in `test/prog_syntax_test.lg` -- keep it that way when changing either
  side, since two scanners that disagree about where a string ends is
  exactly the class of bug that produces "auto-pairing works except in
  this one file."
- **Reducing over an empty `(map f (concat ...))` calls the reducing fn
  once with nil.** let-go laziness bug: `(reduce f init (map g (concat
  [] [])))` invokes `f` with a nil element even though `seq`/`count`/`vec`
  all agree the sequence is empty — `(map g [])` and `(concat [] [])` on
  their own reduce fine; it's specifically map-over-concat. Materialize
  with `mapv`/`vec` before reducing (see the span conversion in
  `legmacs.render`'s text-row, which hit this in production shape:
  `style-array` reduces over converted span lists that are usually empty).
- **Buffer text must never reach the frame raw.** A literal tab byte
  written to the terminal doesn't paint cells — it *jumps* the cursor to
  the next tab stop, so whatever the previous frame had there shows
  through (looks like a refresh/scrolling bug; really a broken
  every-cell-painted invariant). Other control bytes are worse (a raw ESC
  in a file would execute as ANSI mid-frame). `legmacs.render/line-display`
  is the chokepoint: it expands tabs to tab stops and control bytes to
  caret notation *with a char-index → display-column map*, and everything
  positional (highlight spans, cursor placement, `:scroll-col` — which is
  a display column) must go through that mapping, not raw char indices.
- **`finally` can't be used for cleanup that has to survive an exception.**
  A let-go `try` whose only trailing clause is `finally` *swallows* the
  in-flight exception and returns it as an ordinary value (so the caller
  sees a successful-looking `#error` instead of a raised one), and a
  `catch` that rethrows skips the `finally` entirely. Neither shape both
  propagates and cleans up. Reify the outcome instead — `(let [outcome
  (try {:ok (f)} (catch e {:err e}))] (cleanup!) (if (contains? outcome
  :err) (throw (:err outcome)) (:ok outcome)))` — as `legmacs.vibe/
  with-context` does. `main.lg`'s `(try (run ws) (finally
  (shutdown-terminal!)))` is fine only because it doesn't need the
  exception afterwards.
- **`json/write-json` mangles string map keys.** `{"a" 1}` serializes as
  `{"\"a\"": 1}` — the key gets quoted twice. Keyword keys are correct, so
  build request bodies with keywords (`{:max_tokens 5}`, underscores and
  all); `legmacs.vibe`'s provider `:body` functions do exactly that.
  `json/read-json` is fine either way; it takes `{:keywords? true}` (not
  `:keywordize`) if you want keyword keys back.
- **A thrown plain string has no `ex-message`.** `(throw "boom")` is caught
  as the bare string, where `(ex-message e)` is nil and `(str e)` is the
  message; a real error object is the other way round. Use `(or (ex-message
  e) (str e))` for anything that catches both.
- Ctrl-Space is sent as a NUL byte (0x00) by terminals, and BEL (0x07,
  also literally Ctrl-G) doubles as `term/read-key`'s SIGWINCH wake-up —
  `legmacs.keys` has to special-case both; see its docstrings before
  changing control-byte handling.
- **`legmacs.lisp-syntax/in-string-or-comment?` answers a different
  question than "is *point* inside an in-progress string."** It's a
  half-open-span (`:end` exclusive) check meant for an already-typed
  character at a fixed offset (what `update-paren-match` uses it for). At
  the exact offset where an *unterminated* string's span currently ends,
  that check reads as "just past it," not "inside it" — and that offset is
  precisely where point sits right after typing an opening `"` with no
  close yet, which is the common case auto-pairing (`legmacs/modes/
  letgo.lg`) actually needs to get right. Worse, a string closed exactly at
  that same offset produces a *structurally identical* trailing span, so
  no amount of post-processing `:spans` alone can tell the two apart.
  `scan`'s `:in-string?` field (scan truncated to the offset in question)
  exists to answer this correctly instead — use it, not
  `in-string-or-comment?`, for any "is point inside a string right now"
  check. Comments don't share this ambiguity (nothing closes a comment
  early the way a closing quote does), so a plain end-of-span check is
  fine for those.

## Not part of this repo's normal workflow

`build/` is a generated Go project (let-go's Go-lowering/`gogen` tooling
transpiling the `.lg` sources under `build/src/` into native Go packages,
per `build/main.go`'s own header comment) — a performance experiment, not
something edited by hand or required for normal development. There's no
in-repo script that regenerates it; treat it as a build artifact.
