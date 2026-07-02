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
lg -e "(require '[test :refer [run-tests]] '[test.region-test]) (run-tests)"
```

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
frame for visible lines only) and `:after-command` (runs after every
dispatch while that mode is active — used for paren-matching). `legmacs/
lisp_syntax.lg` is a single-pass bracket/string/comment scanner shared by
paren-matching, auto-indent, expand-region, and auto-pairing in
`legmacs/modes/letgo.lg`; `legmacs/modes/markdown.lg` is a second,
independent mode (highlighting only) built the same way; and `legmacs/
modes/prog.lg` generalizes that shape into one spec-driven per-line scanner
(comment markers, string delimiters, keyword/type/constant word sets)
behind highlight-only modes for Go/JS/Python/C/shell/Rust/JSON/YAML —
its `register-prog-mode!` is both how the built-ins register and the
user-facing one-call way to add a language.

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
