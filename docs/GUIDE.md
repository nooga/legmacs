# legmacs guide

The full reference: running and building, the complete keymap, the mode
system, the scripting API, and how the editor is put together. For a quick
intro and install instructions see the [README](../README.md).

## Running and building

```sh
lg main.lg [file...]
```

Run from the repo directory, or install the `bin/legmacs` wrapper somewhere
on your `PATH` (or symlink to it) to run it from anywhere:

```sh
ln -s "$(pwd)/bin/legmacs" /usr/local/bin/legmacs
legmacs [file...]
```

Multiple file arguments each open in their own buffer (the first one
focused). The wrapper just sets `lg`'s namespace search path so `legmacs.*`
resolves regardless of your current directory, and so an optional
`~/.config/legmacs/legmacs_init.lg` gets picked up (see Scripting, below).
Running `lg main.lg` directly works fine too; you just won't get user-config
loading unless `LG_SOURCE_PATHS` already includes this directory.

`./build.sh` AOT-compiles the whole editor into one standalone native binary
at `build/bin/legmacs`. It needs a [let-go](https://github.com/nooga/let-go)
source checkout beside this repo (override with `LETGO_DIR`). This is the
build the release workflow ships to Homebrew and the releases page.

## Using it

legmacs uses Emacs-style chords: `C-x` means hold Control and press `x`;
`M-x` means Escape then `x` (or Alt+x, which most terminals send as the same
two bytes). Sequences like `C-x C-s` are pressed one chord after another,
not held together.

| Keys | Action |
|---|---|
| `C-f` / `→` | forward-char |
| `C-b` / `←` | backward-char |
| `C-n` / `↓` | next-line |
| `C-p` / `↑` | previous-line |
| `C-a` / Home | beginning of line |
| `C-e` / End | end of line |
| `M-f` / `M-b` (or `C-→` / `C-←`) | forward/backward word |
| `M-}` / `M-{` (or `C-↓` / `C-↑`) | forward/backward paragraph |
| `M-<` / `M->` (or `C-Home` / `C-End`) | start/end of buffer |
| `C-v` / `M-v` | page down/up |
| `C-l` | recenter: scroll point to the middle of the window |
| `M-g g` | go to a line by number |
| `RET` | newline |
| `DEL` | delete backward |
| `C-d` / Delete | delete forward |
| `C-k` | kill to end of line |
| `M-k` | kill whole line |
| `M-d` / `M-DEL` (or `C-Delete`) | kill word forward / backward |
| `C-y` | yank (paste) |
| `C-SPC` | set the mark (start a region) |
| `C-w` | kill (cut) the region |
| `M-w` | copy the region, without deleting it |
| `C-x C-x` | swap point and mark |
| `C-_` / `C-x u` | undo |
| `M-_` | redo |
| `C-s` | incremental regexp search (swiper-style); `C-s`/`C-r` step through matches |
| `M-%` | query-replace: replace occurrences from point on, asking at each one (`y`/`n`/`!`/`q`) |
| `M-;` | comment-dwim: toggle a comment on the current line, or every line the region spans |
| `M-/` | dabbrev-expand: complete the word before point from one already in the buffer; press again to cycle |
| `C-x C-SPC` | pop-mark: jump back to before the last long-range jump (`goto-line`, landing on a swiper match, `beginning`/`end-of-buffer`) |
| `C-x (` / `C-x )` | start / stop recording a keyboard macro |
| `C-x e` | replay the last recorded keyboard macro |
| `M-!` | run a shell command, output in the echo area |
| `C-x C-s` | save |
| `C-x C-w` | write file (always prompts) |
| `C-x C-f` | open file in a new buffer (or switch to it, if already open) |
| `C-x b` | switch to a buffer by name (`TAB` completes; a new name creates one) |
| `C-x C-b` | list open buffers |
| `C-x k` | kill (close) the current buffer |
| `C-x <right>` / `C-x <left>` | cycle to the next / previous buffer |
| `C-c C-z` | open (or switch to) the `*repl*` buffer |
| `C-c C-v` | in let-go-mode, asynchronously evaluate the region/form at point and replace it with its result |
| `C-x 2` | split the current window in two, one above the other |
| `C-x 3` | split the current window in two, side by side |
| `C-x o` | move to the next window (cycles top-to-bottom, left-to-right) |
| `C-x 0` | close the current window (the buffer stays open) |
| `C-x 1` | make the current window the only one |
| `C-x C-c` | quit (prompts if the current buffer has unsaved changes) |
| `C-g` | cancel / quit-out-of-here |
| `M-x` | run a command by name (`TAB` completes); the live list shows each command's current key bindings. `load-theme` switches the color palette |
| `C-x C-q` | toggle the current buffer read-only |
| `C-h b` | show all key bindings (mode maps, then global) in a read-only `*Help*` buffer |
| `C-h k` | describe the command bound to the next key sequence |
| `C-h f` | describe a command by name (docstring + keys) |
| `C-h m` | describe the current major and minor modes |
| `C-h C-h` | list the `C-h` help prefix itself |

Pasting into a terminal that supports bracketed paste lands as one atomic
insert, not a burst of individual keystrokes -- so a multi-line paste can't
trip auto-pairing or auto-indent into mangling it the way replaying it one
character at a time would.

Files indented with real tabs (Go, Makefiles) display with each tab
expanded to the next 4-column stop — `(reset! legmacs.render/tab-width 8)`
in your init changes that — and any other control character in a file
shows as caret notation (`^[`, `^M`, ...) rather than reaching the
terminal raw. This is about displaying what's already in the file; TAB the
key inserts spaces (`tab-insert`) in an ordinary buffer, and re-indents the
current line in one that has a major mode with indent rules (see
[Indentation](#indentation)).

Completing prompts (`M-x`, `C-x C-f`, `C-x C-s`, `C-x C-w`, `C-x b`) show a
live, vertico-style vertical list that narrows as you type. Matching is
*fuzzy* (the characters you type just have to appear in order, so `nxtl`
finds `next-line`), best match highlighted at the top. `M-x` annotates each
command with the keys that would invoke it in the current buffer, drawn
muted after the name (`save-buffer` then `C-x C-s`); the selected row also
right-aligns that command's docstring when it has one. Matching is still on
the name, so `sbf` finds `save-buffer` and `RET` still runs it. (File prompts match fuzzily within the current directory
segment: `src/cmp` narrows the entries of `src/`.) `C-n`/`C-p` (or the
arrows) move the highlight, `TAB` pulls the highlighted candidate up into
the input line (so you can descend into a directory or refine further).
`M-x` and the buffer/mode pickers submit the
**highlighted** candidate on `RET`; the file prompts submit exactly the
**path you typed** on `RET` (so a brand-new filename is never silently
swapped for a lookalike), so `TAB` then `RET` to open a listed one, or
`M-RET` anywhere to force the raw text.

`C-s` is a live, ivy/swiper-style search built on the same growable,
selectable echo area: the input is treated as a regexp, every matching line
shows in a list that grows out of the echo area, and point previews the
selected match in the buffer as you type. `C-s`/`C-n` (and `C-r`/`C-p`) step
through matches, `RET` leaves point on the current one, `C-g` returns to
where you started. (The older one-shot `search-forward` is still there under
`M-x`.)

`C-h` is a help prefix. `C-h b` (`describe-bindings`) opens the key-binding
list — the current major mode, then any minor modes with bindings, then
global — in a real, read-only, syntax-highlighted `*Help*` buffer rather
than a one-shot overlay, so you can scroll it, search it with `C-s`, and
switch away and back like any other buffer. Each binding is followed by
that command's docstring, flattened to one muted line. `C-h k` (`describe-key`)
captures the next key sequence without running it and shows the command
plus its docstring; `C-h f` (`describe-function`) does the same from a
command name (the same fuzzy list as `M-x`); `C-h m` describes the buffer's
modes; `C-h C-h` lists the prefix itself. Read-only is a general buffer
property (`C-x C-q` toggles it on any buffer); the buffer mutation
primitives in [`legmacs/buffer.lg`](../legmacs/buffer.lg) refuse edits when
it's set, so no command (built-in or from your config) can modify a
read-only buffer, while movement, scrolling, and search keep working.

Multiple buffers: `lg main.lg a.txt b.txt` opens both (the first one
focused); `C-x C-f` on a new path opens another without touching whatever
you were already editing, and on a path that's already open just switches
to it rather than loading it twice. Buffer names are basenames (`a.txt`,
not the full path), deduplicated like Emacs's if two open files share one
(`a.txt`, `a.txt<2>`). Undo history, the mark, and paren-match are all
per-buffer; the echo-area message, minibuffer, and the one shared
clipboard are not, so they're exactly as visible switching buffers as they
are switching windows.

And multiple windows: `C-x 2` splits the current window into two stacked
ones, `C-x 3` into two side-by-side ones (separated by a `│` divider), in
any combination, resplit as deep as you like. `C-x o` cycles focus through
them (only the active window's mode line renders bright; the terminal
cursor is always in the active window), `C-x 0` closes the current one,
`C-x 1` goes back to a single window. Splitting shows the same buffer
twice; each window then keeps its own cursor and scroll position, Emacs
window-point style, so you can hold two places in one file — or use
`C-x b`/`C-x C-f` in one window to look at something else entirely.
Windows are viewports, not owners: `C-x 0` never closes a buffer, and
`C-x k` never closes a window (a window whose buffer was killed just
shows another one). Sizes are proportional, so resizing the terminal
re-flows the same split shape. Buffer switching (`C-x b`, `C-x <right>`,
find-file) always acts on the active window and leaves the others alone.

`C-SPC` sets the mark, then move point anywhere and the region between the
two is highlighted. `C-w` cuts it, `M-w` copies it (either way it shares
the same one clipboard slot `C-y` yanks from), and `C-g` drops the mark
without touching the buffer.

## Modes

A buffer can have a major mode: a keymap that shadows the global one while
it's active. Shadowing happens key-by-key, not for a whole sequence at once.
A mode can add a new leaf under an existing global prefix (let-go-mode adds
`C-x C-e` without hiding `C-x C-s`/`C-x C-c`/etc), and anything the mode
doesn't bind falls straight through to global. A mode can also plug into
rendering (`:highlighter`, a pure per-line syntax colorer) and into every
keystroke (`:after-command`, e.g. to keep a matched-paren highlight current);
see [`legmacs/modes.lg`](../legmacs/modes.lg)'s `register-mode!`.

A language with constructs that cross line boundaries declares
`:highlight-line` instead: `(fn [carry line] -> {:spans ... :carry ...})`,
where `carry` is whatever the previous line returned (`nil` at the top of the
buffer) and is entirely private to the mode -- `legmacs.render` only threads
it down the buffer. That's what keeps a block comment, a raw string or a
markdown code fence colored on every line it covers, including when the line
that opened it has scrolled off the top of the screen. Since the carry has to
be folded from the first line of the buffer to the top of the viewport, a
mode should also declare `:highlight-carry` -- the same answer without the
spans, and free to shortcut: "no backtick on this line, so nothing can have
changed" is one substring search, versus tokenizing a line nobody is looking
at.

A buffer can *also* have any number of **minor modes** layered on top of its
major mode. They're the same registry entry (`register-minor-mode!`), the
only difference being where the buffer holds them: the major mode lives in a
single `:mode` slot (at most one, it's the content-defining mode), minor
modes in an ordered `:minor-modes` vector (toggle with
`enable-`/`disable-`/`toggle-minor-mode`, or `M-x toggle-minor-mode`).
Precedence is fixed: minor modes shadow the major mode, which shadows global,
exactly what a modal-editing layer needs. Two extra hooks make that layer
possible: a mode's `:keymap` may be a `(fn [state] -> keymap)` (so its
active bindings can depend on buffer-local state), and `:suppress-self-insert?`
(a bool or fn of state) makes unbound printable keys inert instead of typing
themselves. `:highlighter`/`:after-command`/`:status`/`:status-right` (a
right-aligned mode-line lighter) all compose across every active mode.
`legmacs.dispatch` also stamps each pressed chord onto the state as
`:last-chord`, so a mode can observe the raw key, not just the resulting
state.

`*scratch*` starts in **let-go-mode**, and any file ending in `.lg` opens
into it automatically -- as do `.clj`/`.cljc`/`.cljs`/`.bb`/`.edn` files,
since let-go is a Clojure dialect close enough in reader syntax
(parens/brackets/braces, `;` comments, `"`-strings) that real
Clojure/ClojureScript/Babashka source and EDN data get correct
highlighting, paren-matching, auto-pairing, and auto-indent as-is; only
in-process eval (`C-x C-e` etc.) is let-go-specific and won't succeed on
forms that aren't also valid let-go:

| Keys | Action |
|---|---|
| `C-x C-e` | evaluate the s-expression before point, show the result in the echo area |
| `C-j` | evaluate the s-expression before point, insert the result right there |
| `C-c C-v` | asynchronously evaluate the region or form at point and replace it with the result |
| `C-c C-e` | evaluate the whole buffer |
| `C-c M-j` | jack in: start the project's nREPL server (lgx, Clojure CLI, Babashka or `lg`) and connect (see [nREPL](#nrepl)) |
| `C-c M-c` | connect to an nREPL server that is already running |
| `C-c C-q` | disconnect this project from nREPL (stopping a jacked-in server), back to in-process eval |
| `RET` | newline, aligned under the form it's inside |
| `TAB` | re-indent the current line the same way; if that changes nothing and point is after a symbol, complete it |
| `M-TAB` | complete the symbol at point (`C-M-i` sends this) |
| `M-.` | jump to the definition of the symbol at point (into a jar, read-only, for Clojure libraries) |
| `M-,` | jump back to where the last `M-.` came from |
| `C-c C-d d` | show the symbol at point's arglists, doc and source location in `*Help*` (also `C-c C-d C-d`) |
| `C-c e` | expand the selection outward (word → sexp → next sexp up → ...) |
| `C-c u` | contract the selection back in, undoing the last expand |
| `(` / `[` / `{` | insert the matching closer too, point left between them |
| `)` / `]` / `}` | move over a closer already sitting under point instead of doubling it |
| `DEL` | also deletes an adjacent empty pair (`(\|)`) in one step |

Auto-pairing stands down inside a string or comment (typing `(` while
writing a docstring that itself contains code doesn't try to balance it),
using `legmacs.lisp-syntax`'s scanner to tell "point is inside an
in-progress string" apart from "point sits right after a string that just
closed", since the two look identical from the buffer's text alone until you
track whether that trailing quote was ever seen.

Point does not need to sit precisely after a close delimiter: the eval
commands use the shared Lisp scanner to find the innermost form containing
point, also accepting point immediately after its close. An active region
wins and may contain several forms. The evaluated span flashes for ordinary
`C-x C-e`/`C-j` eval and stays highlighted while an asynchronous `C-c C-v`
job is in flight; the echo area and mode line also say `evaluating...`.

Eval isn't a sandboxed toy evaluator: it runs in-process, in the same
namespace the editor itself is running in (`legmacs.main`), using let-go's
own `eval`/`read-string`/`load-string` (`pkg/compiler/eval.go`). Code typed
into `*scratch*` sees every alias `main.lg` set up (`buf/`, `keys/`, `km/`,
`dispatch/`, `render/`) plus all of `clojure.core`, live. For example
`(km/bind-key! "C-c z" :keyboard-quit)` typed into `*scratch*` and run with
`C-x C-e` rebinds a key in the *running* editor, no restart. `C-j` repeated
down the buffer is what makes `*scratch*` feel like a REPL transcript, the
same way Emacs's does. Eval errors (bad syntax, a runtime exception,
whatever) are caught and shown in the echo area rather than crashing the
editor.

`C-c C-e` puts the current namespace back when it's done. It has to:
evaluating a buffer that opens with `(ns foo)` -- which every real
`.lg` file does -- relocates the running namespace permanently, and since
that namespace *is* the editor's own, one `C-c C-e` on an ordinary file
would otherwise leave `buf/`, `km/`, `dispatch/` and `vibe` unresolvable
for the rest of the session. `C-x C-e` and `C-j` are deliberately left
alone, as is the `*repl*` buffer: an `(in-ns 'foo)` you typed and evaluated
on purpose should still move you. A buffer with no ns form (*scratch*)
evals in the editor ns, so lg-isms like `(now)` resolve the same as
`C-x C-e`.

`C-c C-z` opens (or switches to) `*repl*` -- a dedicated, persistent REPL
buffer rather than a transcript you build by hand with `C-j`. It's plain
let-go-mode (so everything above -- highlighting, auto-pairing, paren
matching, expand-region -- comes along for free) plus one extra minor
mode that reinterprets `RET`: type an expression after the `=> ` prompt
and press it. A complete top-level form gets evaluated in place (same
`load-string`, same running-editor namespace as `C-x C-e`), its result
printed right after it, and a fresh prompt opened below; an incomplete one
(an open bracket) just inserts a newline and auto-indents so a multi-line
`defn` can keep being typed across several `RET` presses, the same way it
would in an ordinary `.lg` buffer. `RET` anywhere but the last line is an
ordinary let-go-mode `RET` -- editing earlier transcript text doesn't try
to re-run it. [`legmacs/modes/repl.lg`](../legmacs/modes/repl.lg) has the
details, including the one real wrinkle: because auto-pairing keeps
closing every bracket you open immediately, "is the input complete" has to
be judged from the text *before point*, not the whole line -- otherwise
the auto-inserted closer sitting after point would make every half-typed
form look finished the moment its first bracket goes in.

Indentation is Lisp's own rule rather than a nesting count: a continuation
line lines up under the enclosing form's first argument, so a multi-line
call reads as one call, while forms with a body rather than an argument
list (`defn`, `let`, `when`, `try`, anything def-ish or `with-`/`when-`/
`if-`-shaped) indent a flat two columns in from their bracket. `RET` uses
it, and `TAB` re-indents the line you're on.

```clojure
(println "a"
         "b")        ; aligned under the first argument
(defn foo [x]
  (inc x))           ; body form: two in from the (
```

Syntax highlighting (strings, comments, a curated set of special-form
names), paren matching, auto-indent, and expand-region all share one
scanner, [`legmacs/lisp_syntax.lg`](../legmacs/lisp_syntax.lg), that finds
brackets, strings, and comments in one pass, correctly skipping character
literals (`\(`, `\"`, `\\`, ...) so they don't get mistaken for real
structure. Paren matching and expand-region scan the whole buffer, so they
work correctly across line breaks. The highlighter reads one line at a time,
but carries a single bit between them (`:highlight-line`, above) -- whether a
string literal was left open -- which is the only construct in let-go that
crosses a line, so a multi-line string stays colored to its closing quote no
matter how far above the screen it started.

See [`legmacs/modes/letgo.lg`](../legmacs/modes/letgo.lg) for all of the
above. Registering your own mode from `legmacs-init.lg`, with or without
these features, is the same `register-mode!`/`register-auto-mode!` call.

### Vibing (`C-c C-v`)

[`legmacs/vibe.lg`](../legmacs/vibe.lg) adds one ordinary function to the
generic let-go evaluator.
The function is ordinary: `(vibe "a quicksort")` asks the configured model
for let-go code and returns it as a string, so `C-x C-e` on it echoes the
code, and it composes (`(str (vibe "a") (vibe "b"))` is fine). `C-c C-v`
is let-go-mode's universal eval-and-replace command: it works on `(vibe ...)`
exactly as it works on `(+ 1 2)`, replacing the form as one undo step so one
`C-_` puts the source back.

What makes the answers usable is the context. While the generic evaluator
runs it, `vibe` derives the call site from `*eval-replace-context*` and sends the
whole buffer with the call site swapped for a `<<<VIBE:HERE>>>` marker,
plus the column that marker starts at, plus a live catalog of let-go's
canonical namespaces (`string/`, `json/`, `os/`, never `clojure.string/`)
and the file's existing `:require` aliases. The model can also call
`list-namespaces`, `ns-publics`, and `var-doc` against the running VM
before it writes code. The splice is one undo step: the generated
forms replace the call, and any namespace those forms use that the file
does not already require is inserted into the `ns` form. A bare
`(vibe "...")` from the REPL still works, just without a
file to splice requires into. JVM-Clojure prefixes in the reply
(`clojure.string/join`) are rewritten to the canonical names before
anything is inserted.

The catalog also includes the live editor's scripting surface
(`legmacs.keymap`, `legmacs.modeline`, `legmacs.theme`, `legmacs.modes`,
`legmacs.buffer`, `legmacs.minibuffer`, `legmacs.modes.prog`) — the same
APIs `legmacs-init.lg` uses. A `(vibe "clock on the modeline")` can
therefore emit `register-modeline-segment!` / `bind-key!` / `set-theme!`
code that fits the file. Those calls take effect when you evaluate them
(`C-x C-e` / `C-c C-e`), the same as any other init form; vibe only
splices the source.

Setup is `~/.config/legmacs/vibe.edn` (or `$LEGMACS_CONFIG_DIR/vibe.edn` --
the same variable `bin/legmacs` already honours for `legmacs_init.lg`):

```clojure
{:key "sk-..."}
```

A key is all that's required. The default model is `gpt-5.6-luna`;
`:model` picks another. The file is merged into the config map wholesale
rather than being read key by key, so everything else works from there too:

| Key | Meaning |
|---|---|
| `:key` | API key. Falls back to the provider's env var (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) if absent. |
| `:model` | Model id. Defaults to the provider's (`gpt-5.6-luna` / `claude-sonnet-5`). |
| `:provider` | `:openai` (default) or `:anthropic`. |
| `:url` | Endpoint. Default is OpenAI `/v1/responses`. Point it at llama.cpp/ollama/openrouter/groq (`.../v1/chat/completions`) and the older chat-completions shape is inferred. Or set `:api` explicitly. |
| `:api` | `:responses` (default) or `:chat-completions`. Responses is the current OpenAI API and the one that can combine function tools with reasoning on gpt-5.6-*. |
| `:max-tokens` | Default 8192. Reasoning models spend this budget on reasoning *and* output, so a small value shows up as an empty reply, not a truncated one. |
| `:system` | The instructions the model gets. Replace to change house style. |
| `:tools` | `true` (default): the model may call `list-namespaces` / `ns-publics` / `var-doc`. Set `false` for local endpoints that do not speak tool calls. A 400 on a tools turn retries once without them. |
| `:max-tool-rounds` | Cap on tool-call turns before the model has to answer. Default 4. |
| `:reasoning-effort` | How hard the model thinks: `none`, `low`, `medium` (default), `high`, and whatever else the model accepts (`xhigh`, `max`, ...). Keywords or strings. Sent as Responses `reasoning.effort`. On chat/completions, tools force `none` because that endpoint rejects the combination. |

Config is four layers, each beating the one before it: `legmacs.vibe/defaults`,
then vibe.edn, then the `legmacs.vibe/config` atom (for `legmacs-init.lg`:
`(swap! vibe/config assoc :model "gpt-5.6-terra")`), then a per-call opts
map (`(vibe "a quicksort" {:model "gpt-5.6-sol"})`). vibe.edn is re-read on
every call, not cached at startup, so editing it in a legmacs buffer takes
effect on the next `C-c C-v`. A malformed or half-saved file parses as no
config rather than an error, so a stray keystroke in it can't break the
command with a reader error -- you get the ordinary "no API key, put one
in ..." message instead.

A provider in `legmacs.vibe/providers` is four keys and two functions
(`:body` builds the request map, `:extract` pulls the text out of the
parsed response — or `{:text ... :calls [...]}` when the model asked
for tools), so adding one is a `swap!`.

Vibe has to wait on a network round-trip, so the generic `C-c C-v` command
snapshots the chosen form and runs its eval through `bufs/spawn-task`.
The HTTP call (or any other evaluation) runs in a future; `main.lg` keeps reading keys, and
`drain-jobs` splices the answer on the main thread *before the next
frame is painted*, so the replacement shows up without a keystroke. C-g
discards the result (it does not abort the request). A throw in the apply
step is an echo-area message, not a crash. The mode line shows `evaluating...`
so you still know work is in flight after the echo area has been cleared
by typing. The same async boundary is where other evaluators plug in: form
discovery, highlighting, stale-source checks, and main-thread application
do not depend on the in-process backend. Register an async-safe
`(fn [request] -> value)` with `register-eval-backend!` and select it with
`use-eval-backend!`; the request carries `:op` (`:eval`, or `:load` for
`C-c C-e`), `:source`, and filename/line/column/namespace context. A
backend whose server has already printed the value returns
`(printed-result text out)`, which `C-c C-v` splices verbatim and the
other commands show verbatim. While any backend other than `:in-process`
is selected, `C-x C-e`, `C-j` and `C-c C-e` go through it asynchronously
too, and the let-go-mode lighter names it.

### Completion

`TAB` in let-go-mode re-indents, and when the line is already indented
right and point sits just after a symbol, completes it instead (Emacs's
`tab-always-indent 'complete`); `M-TAB` always completes. One candidate
is inserted; several sharing a longer prefix insert that prefix; otherwise
they open in the minibuffer picker, fuzzy-filtered, with each candidate's
type and namespace beside it and arglists/doc on the selected one.

Candidates come from wherever the buffer evaluates. In-process that is the
editor's own VM: the buffer's namespace if it's loaded (else the current
one), `alias/` prefixes from its aliases *and* from its `ns` form's
`:require`s (so a file you haven't evaluated still completes `str/`), and
special forms. On an nREPL connection the server answers, using
cider-nrepl's context-aware `complete` when it has it. A backend supplies
its own with
`(legmacs.modes.letgo-complete/register-completer! :backend f)`.

### Eldoc, doc, and definitions

While you type inside a call, the echo area shows what's being called and
its arglists -- `legmacs.buffer/new-state: [filename lines]` -- whenever
nothing else is using it (a result or an error always wins).
`C-c C-d d` puts the symbol at point's arglists, docstring and source
location in `*Help*`. `M-.` jumps to its definition, opening the file if
it isn't open yet; `M-,` jumps back, across buffers, as many times as you
went forward.

These answer from the same place as eval and completion. In the editor's
VM that's the var's own metadata, resolved through the buffer's `ns`,
its aliases, and its `ns` form's `:require`s; a relative `:file` (let-go
records `legmacs/buffer.lg`) is found from the buffer's directory
upward, then the let-go source path. On an nREPL connection the server
answers: with cider-nrepl (every Clojure CLI jack-in) that is full
arglists and docs for everything including `clojure.core`, and `M-.`
into a library opens its source straight out of the jar as a read-only
buffer. Remote eldoc answers are cached for a few seconds, so typing
doesn't mean a round trip per key.

Two limits come from let-go rather than legmacs: its core functions
(`map`, `str`, ...) carry no arglists or docstrings, and live in the
runtime's embedded sources, so they get neither eldoc nor a file to
jump to; and its nREPL server's `info` returns only a name, so on an
`lg`/lgx connection doc and `M-.` have little to go on yet. The
AOT-built binary (`build.sh`) currently also loses `def` metadata, so
there in-process eldoc/doc/`M-.` know a var's name but not its arglists,
doc or file; running from source (`lg main.lg`, `bin/legmacs`) has all
of it.
`(reset! legmacs.modes.letgo-lookup/eldoc-enabled? false)` turns eldoc
off; a backend adds lookups with `register-lookup!`.

### nREPL

legmacs is an nREPL client
([`legmacs/nrepl.lg`](../legmacs/nrepl.lg)), and can start the server for
you ([`legmacs/jack_in.lg`](../legmacs/jack_in.lg)), CIDER-style.

**Jack-in.** `C-c M-j` (`M-x nrepl-jack-in`) in a let-go/Clojure buffer
finds the project root (the nearest directory with a marker file), picks
how to start it, starts it on a free port, waits for it, and connects:

| Project | Marker | Server command |
|---|---|---|
| lgx | `lgx.edn` | `lgx nrepl --port N` |
| Clojure CLI | `deps.edn` | `clojure -Sdeps '{nrepl + cider-nrepl}' -M -m nrepl.cmdline --port N --middleware '[cider.nrepl/cider-middleware]'` |
| Babashka | `bb.edn` | `bb nrepl-server N` |
| let-go (`lg`) | `deps.edn`, or none | `lg -n -p N` |

`deps.edn` is both a Clojure CLI and a plain-`lg` project file, and
`bb.edn` often sits next to it, so the buffer's extension decides (`.lg`
→ lgx, else lg; `.clj`/`.cljc`/`.cljs` → Clojure CLI; `.bb` → Babashka);
a real tie asks. A lone `.lg` file with no project gets `lg -n` in its
own directory. `M-x nrepl-jack-in-command` shows the command line for
editing first (CIDER's `C-u C-c M-j`), and `M-x nrepl-server-log` shows
the server's output -- which is also where to look when a jack-in fails
(the echo area gets the log's last lines). The first Clojure CLI jack-in
in a project downloads nrepl and cider-nrepl, which can take a couple of
minutes; the editor stays usable meanwhile. Knobs:
`legmacs.jack-in/clojure-cli-aliases` (e.g. `":dev:test"`),
`nrepl-version`, `cider-nrepl-version` and `startup-timeout-ms` are atoms.
A new kind of project is one `register-launch-config!` call (see the
namespace docstring for the map's keys).

**Connections are per project.** A buffer visiting a file under a
connected project's root evaluates and completes on that project's
server; the lighter says `nrepl:clj` (or `:bb`, `:lg`, `:lgx`, or plain
`nrepl` for a manual connection). Buffers without a file, `*scratch*`
and `*repl*` among them, stay in the editor's VM, so two projects on two
servers and editor scripting all work at once. `C-c C-q` disconnects the
buffer's project and stops its server if jack-in started it; every
connection and server is closed when the editor exits.

**Connecting by hand.** For a server you started yourself, `C-c M-c`
(`M-x nrepl-connect`) prompts for host:port, pre-filled from the nearest
`.nrepl-port` (`lg -n` and `lgx nrepl` write one); a bare port means
localhost.

While connected:

- `C-x C-e` / `C-j` / `C-c C-v` send the form with the buffer's `ns`,
  file, line and column; `C-c C-e` is the server's `load-file`
- anything the form printed shows in the echo area above `=> value`
- a server error is an ordinary `Eval error:` with the message and its
  `caused by:` chain on one line (the source excerpt and stack are dropped)
- evaluating in a namespace the server hasn't loaded yet says so: let-go's
  server quietly falls back to `user` instead of failing, so defs would
  otherwise land somewhere unexpected. `C-c C-e` the file once first
- `C-j` checks the form is still where it was before inserting, like
  `C-c C-v` does, and both check you're still in the same buffer

Requests are one blocking round trip at a time inside the usual
`spawn-task` future, so there is no background reader for the main loop to
poll; `C-g` drops a pending result but the server still finishes the
request.

Any file ending in `.md` or `.markdown` opens into **markdown-mode**
(highlighting only, no key bindings of its own, so every global key still
works): ATX headers (`#` through `######`), `**bold**`/`__bold__`,
`*italic*`/`_italic_` (a bare `_` inside a word, like `some_var_name`,
correctly doesn't count, markdown's own rule, unlike `*`), `` `inline
code` ``, `[links](url)`/`![images](url)`, `>` blockquotes, `-`/`*`/`+`/`1.`
list markers, and `---`/`***`/`___` horizontal rules; see
[`legmacs/modes/markdown.lg`](../legmacs/modes/markdown.lg). Fenced code
blocks are the one thing it carries state for: everything between two
` ``` ` lines is colored as code, and -- just as important -- the inline
scanner stands down there, so
the asterisks and underscores in a shell command or a code snippet don't read
as emphasis. Indented (4-space) code blocks still aren't recognized: telling
one from a lazy list continuation needs more context than a line at a time.

### The language pack (Go, JS/TS, Python, C, shell, rc, Rust, JSON, YAML, TOML, Lua, Ruby, SQL, Dockerfile, CSS, HTML, Zig, Java, Kotlin, Swift, C#, PHP, Makefile)

Beyond the two hand-written modes, one spec-driven scanner
([`legmacs/modes/prog.lg`](../legmacs/modes/prog.lg)) provides major modes
for the usual suspects: **go-mode** (`.go`), **javascript-mode**
(`.js`/`.jsx`/`.mjs`/`.cjs`/`.ts`/`.tsx`), **python-mode** (`.py`),
**c-mode** (`.c`/`.h`/`.cpp`/`.hpp`/`.cc`/...), **shell-mode**
(`.sh`/`.bash`/`.zsh`), **rc-mode** (`.rc`/`rcmain`, the plan 9 shell:
`''`-quoted strings, and `$x`/`$#x`/`$"x` variable forms), **rust-mode**
(`.rs`), **json-mode** (`.json`),
**yaml-mode** (`.yaml`/`.yml`), **toml-mode** (`.toml`), **lua-mode**
(`.lua`), **ruby-mode** (`.rb`), **sql-mode** (`.sql`, keywords
case-insensitive), **dockerfile-mode** (`Dockerfile`/`.dockerfile`, also
case-insensitive), **css-mode** (`.css`), **html-mode**
(`.html`/`.htm`/`.xhtml`/`.xml`/`.svg` -- comments, quoted attribute
values, and tag names, since HTML's structure is tags rather than the
keyword vocabulary this scanner's word-based classification expects), **zig-mode**
(`.zig`, no block comments, just `//`/`///`/`//!` line comments),
**java-mode** (`.java`), **kotlin-mode** (`.kt`/`.kts`), **swift-mode**
(`.swift`), **csharp-mode** (`.cs`), **php-mode** (`.php`, both `//` and
`#` line comments), and **makefile-mode** (`Makefile`/`makefile`/
`GNUmakefile`/`.mk`). A language here is just a data map -- line and block
comment markers, string delimiters (backslash-escapes honored), and
keyword/type/constant word sets -- and `register-prog-mode!` turns it into
a registered mode plus its filename associations. Adding your own from
`legmacs-init.lg` is one call:

```clojure
(require '[legmacs.modes.prog :as prog])

(prog/register-prog-mode! :elixir "elixir-mode"
  {:line-comments ["#"]
   :string-delims ["\"" "'"]
   :keywords #{"def" "defmodule" "defp" "do" "end" "if" "else" "case"
               "cond" "fn" "import" "alias" "require" "use"}
   :constants #{"true" "false" "nil"}}
  [".ex" ".exs"])
```

(That last argument is the general file-association mechanism, usable with
any mode: `(legmacs.modes/register-auto-mode! ".ext" :mode-kw)` maps a
filename suffix to a major mode, consulted whenever a file is opened.)

Beyond those word sets the spec has a few optional knobs:
`:multiline-strings` (`[{:open "`" :close "`" :escape? false} ...]`, for Go's
raw strings, Python's and Swift's triple quotes, JS template literals, Lua's
`[[ ]]`, C#'s `@"..."`), `:preprocessor` (a set of C-style `#directive`
names, recognized only at the head of a line -- an `#include`'s `<header>`
comes out as a string rather than two comparisons), `:var-sigils`
(`["$"]`-style prefixes, so `$HOME`, `${PATH}`, `$(uname)` and rc's `$#argv`
read as one variable reference), and `:functions?` (a name sitting
immediately before a `(` is a call site). A block comment or a multi-line
string stays colored across every line it covers; a plain `"` or `'` string
still ends at end of line, since in the languages without
`:multiline-strings` an unclosed quote is a typo far more often than a
literal, and one stray quote shouldn't repaint the rest of the file.
`:block-comment` is always checked before `:line-comments`, so a language
whose block-open marker starts with its own line-comment marker (Lua's
`--[[` is prefixed by its `--`) still opens a block comment rather than a
line comment swallowing the rest of the line. SQL and Dockerfile keywords
are matched case-insensitively (`register-prog-mode!`'s callers just widen
`:keywords`/`:constants` to both cases, via `both-cases` in
`legmacs/modes/prog.lg`) since real-world SQL/Dockerfiles mix casing.

Every language in the pack also gets paren matching, auto-paired brackets,
and auto-indent for free -- the same three features
let-go-mode has always had, generalized instead of reimplemented per
language. [`legmacs/prog_syntax.lg`](../legmacs/prog_syntax.lg) is a
full-buffer scanner like `legmacs.lisp-syntax`, but spec-driven (the exact
same `:line-comments`/`:block-comment`/`:string-delims` map
`register-prog-mode!` already builds a highlighter from, plus `:brackets`
and `:indent-width` if you want to override either). Three ordinary minor
modes -- `:paren-match`, `:electric-pair`, `:auto-indent`
([`legmacs/modes/structural.lg`](../legmacs/modes/structural.lg)) -- read
whatever spec the buffer's *major* mode declares and behave accordingly; a
major mode with no spec makes them no-ops (plain newline, plain
self-insert), so `register-prog-mode!` is the only thing a language needs
to call to get all three, tuned to it, with zero extra wiring.
`legmacs.modes/switch-to-mode` auto-enables all three the moment a buffer's
major mode has a `:syntax-spec`, so this happens by default for every
built-in language and any you add the same way. let-go-mode keeps its own
older, hand-written versions of all three (built on `legmacs.lisp-syntax`,
which additionally understands Lisp character literals like `\(` --
genuinely Lisp-specific structure the generic scanner doesn't model) rather
than switching to the generic ones.

#### Indentation

`:auto-indent` binds two keys: `RET` opens a new line already indented, and
`TAB` re-indents the line point is on (moving point to the first non-blank
character if it was in the leading whitespace, like Emacs'
`indent-for-tab-command`). Two more moments indent without being asked:
typing a closing bracket pulls its line back to its opener's level when
that bracket is the first thing on the line, and pressing `RET` between a
bracket and its closer -- exactly where auto-pairing leaves point -- opens
the block out into three lines, with the closer back at the opener's
level:

```
func f() {|}      ->      func f() {
                              |
                          }
```

Where a line *should* start is one pure function,
`legmacs.indent/indent-column`, and the rules it follows are data in the
language's own spec, under `:indent`. There are two base styles and an
escape hatch:

- **`:style :absolute`** (the default) -- indentation is bracket nesting
  depth, counted by `legmacs.prog-syntax` over everything above the line.
  Right for C, Go, Rust, JS, Java and friends, and self-correcting: one
  badly indented line doesn't drag the rest of the file with it.
- **`:style :relative`** -- indentation is the previous non-blank line's own
  indentation, adjusted by what that line did and what this line starts
  with. The only thing that works for languages whose blocks aren't
  brackets: shell's `if`/`then`/`fi`, Ruby's `def`/`end`, Lua's
  `function`/`end`, and Python, where the indentation *is* the syntax and
  can only be read off the line above.
- **`:fn (fn [state row spec] -> column)`** -- full control, for rules that
  don't fit either.

The rule keys, all optional, all matched against the line with its strings
and comments blanked out (so an `end` inside a comment closes nothing):

| Key | Meaning |
|---|---|
| `:open-first` | the line's **first** word opens a block (Ruby's `if`, which is also a statement modifier at the *end* of a line) |
| `:open-last` | the line's **last** word opens one (shell's `then`, `do`) |
| `:open-words` | the word opens one wherever it appears (Lua's `function`, buried behind `local`) |
| `:open-suffix` | the line's code ends with this literal (Python's and YAML's `:`) |
| `:close-words` | closes a block wherever it appears, and dedents a line that starts with one (`end`, `fi`, `done`) |
| `:mid-words` | dedents its own line without changing depth, then opens its body again -- `else`, `elsif`, `when`, a switch's `case` |
| `:outdent-after` | `:relative` only: a line starting with one of these pulls the *next* line back (Python's `return`, `pass`, `break`) |
| `:width` | columns per level, overriding the spec's `:indent-width` |

A line opens at most one level, and a line that also closes one (Lua's
`if x then y() end`) opens none -- which is what stops one-liners from
dragging everything after them to the right.

So go-mode says `{:mid-words #{"case" "default"}}` and gets gofmt's switch
layout; python-mode says `{:style :relative :open-suffix [":"] :mid-words
#{"else" "elif" "except" "finally"} :outdent-after #{"return" "pass" ...}}`
and gets the offside rule. In a `:relative` language, finishing a closing
word on a line of its own (`end`, `fi`, `else:`) snaps that line back
immediately -- the word-language version of typing `}`. Type on past it
(`end` into `endpoint`) and the line stays where the keyword put it until
`TAB`; that's the same bargain Emacs' `electric-indent-mode` makes.

### CRUTCH mode (vi-style modal editing)

**CRUTCH** (Could Really Use The Cool Hotkeys) is a bundled *minor* mode that
adds vi/evil-style modal editing. Toggle it per buffer with `M-x crutch-mode`.
Because it's a minor mode it layers *over* whatever major mode the buffer
already has, so a `.lg` file keeps let-go-mode's highlighting, paren-matching,
and auto-pairing while you drive it modally. The status line shows the
current state (`NORMAL`/`INSERT`/`VISUAL`) next to the major mode's name.

It's the same three vi states, but they aren't three modes fighting over a
slot: they're one minor mode carrying a buffer-local `:crutch-state`, whose
`:keymap` is a `(fn [state] -> keymap)` picking the right keymap per state.
Insert state's keymap is nearly empty (just `ESC`), so every other key falls
straight through to the major mode / global self-insert, so insert mode *is*
the normal editor. Normal and visual state set `:suppress-self-insert?` so
unbound keys stay inert without enumerating the keyboard.

| Keys | Action |
|---|---|
| `ESC` | return to normal state |
| `h`/`j`/`k`/`l` | left / down / up / right (counts: `3j`) |
| `w` / `b` | forward / back a word |
| `0` / `$` | start / end of line |
| `gg` / `G` | start / end of buffer |
| `i` / `I` | insert before point / at start of line |
| `a` / `A` | append after point / at end of line |
| `o` / `O` | open a line below / above and insert |
| `x` | delete char(s) under point |
| `dd` / `dw` | delete line / to next word (yankable; counts: `2dd`) |
| `p` | paste the clipboard |
| `u` / `C-r` | undo / redo |
| `v` | start a characterwise selection; `d`/`x` delete it, `y` copies it |
| `/` | search forward |
| `:` | ex line: `w`, `q`, `wq`, `q!` |

Nothing here is privileged: [`legmacs/modes/crutch.lg`](../legmacs/modes/crutch.lg)
is built entirely from `defcommand`/`bind-key!`/`register-minor-mode!`, the
same API your `legmacs-init.lg` has. It's a worked example of the minor-mode
system as much as a feature.

### keycast mode (keystroke preview)

**`M-x keycast-mode`** shows a live preview of your recent keystrokes,
right-aligned on the mode line, the thing screencasts use to show what's
being pressed. It's a pure *observer* minor mode
([`legmacs/modes/keycast.lg`](../legmacs/modes/keycast.lg)): an empty keymap
that binds and changes nothing, an `:after-command` that appends
`:last-chord` (the raw key `legmacs.dispatch` records each keystroke) to a
bounded log, and a `:status-right` lighter that renders it. It's especially
handy next to CRUTCH: the same physical `j` shows up as a bare motion in
normal state and as a typed character in insert state, so the difference
between the modes is visible as you press keys.

### Theming colors

Syntax highlighting, the gutter, echo-area messages, region/search
highlights, and the inactive mode line all read from one palette
([`legmacs/theme.lg`](../legmacs/theme.lg)): an atom of face specs, the same
split the mode line already uses. Render compiles those specs into ANSI;
modes still emit style keywords (`:string`, `:comment`, ...), never escape
codes, so a theme change never touches a mode.

**`M-x load-theme`** completes against the catalog and *replaces* the live
palette. The shipped names:

| Theme | Notes |
|---|---|
| `default` | The original hardcoded colors. Unstyled text uses the terminal's own fg/bg. |
| `acme` | Plan 9 Acme: pale yellow body, cyan tag, black text. Paints its own editor bg. |
| `catppuccin-mocha` | Catppuccin Mocha (MIT, © Catppuccin Org). |
| `catppuccin-latte` | Catppuccin Latte, the light flavor. Paints its own editor bg. |
| `gruvbox-dark` | morhetz Gruvbox dark / medium contrast (MIT/X11). |
| `tokyo-night` | Enkia's Tokyo Night (MIT). |
| `nord` | Nord polar-bluish palette (MIT, © Sven Greb). |

Ported themes include a `:default` face, so they cover the terminal's
background -- that's what makes a light theme work on a dark terminal.
`default` deliberately doesn't, so it still looks like it did before the
palette was data.

```clojure
(require '[legmacs.theme :as theme])

;; switch to a named palette (same as M-x load-theme)
(theme/load-theme! :catppuccin-mocha)

;; recolor strings; everything else stays
(theme/set-theme! {:string {:fg [255 80 80]}})

;; one face, and a vector means :fg
(theme/set-face! :comment [90 90 105])
(theme/set-face! :comment {:italic true})   ; keeps the :fg from above

;; add your own, then M-x load-theme sees it
(theme/register-theme! :paper {:default {:fg [30 30 30] :bg [248 244 232]}
                               :string {:fg [32 100 32]}})

;; the shipped default, matching the colors that used to be hardcoded
(theme/reset-theme!)
```

A face spec is a map with any of `:fg` / `:bg` (Truecolor `[r g b]` triples)
and `:bold` / `:italic` / `:underline` (booleans). `set-theme!` merges per
face; `load-theme!` replaces. The faces highlighters emit (`:string`,
`:keyword`, `:md-header`, `:help-key`, `:region`, ...) share a keyword with
the span style. UI-only faces (`:default`, `:gutter`, `:gutter-current`,
`:tilde`, `:message`, `:error`, `:modeline`, `:inactive-modeline`,
`:candidate-keys`, `:candidate-sel`) are looked up by render directly.
`:candidate-sel` styles the highlighted M-x / swiper row and falls back
to `:region` if a theme doesn't set it. Unknown keys you `set-theme!` are stored and
compiled, so a mode you add can introduce a new style without a render
change.

### Theming the mode line

The mode line ([`legmacs/modeline.lg`](../legmacs/modeline.lg)) is a registry
of named, ordered, independently-colored **segments** plus a small theme. The
built-in segments (buffer name, major mode, position, ...) are registered
through the exact same API your `legmacs-init.lg` uses, so there's no
privileged built-in path here either.

```clojure
(require '[legmacs.modeline :as ml])

;; change the bar's default colors ([r g b] triples; either key optional)
(ml/set-modeline-theme! {:bg [30 30 40] :fg [200 200 210]})

;; the string placed between adjacent segments on the same side
(ml/set-modeline-separator! " | ")

;; add a segment: (fn [state] -> seg), where seg is nil/"" (omit this
;; frame), a plain string (theme's default colors), or a map with its own
;; colors, a "block" that pops out of the bar
(ml/register-modeline-segment! :clock :right
  (fn [state] {:text "12:34" :fg [0 0 0] :bg [90 200 140]}))

;; drop a built-in segment, or move segments around (unmentioned ids keep
;; their relative order, appended after the ones you listed)
(ml/remove-modeline-segment! :pending-keys)
(ml/set-modeline-order! [:mode :buffer :position])
```

A segment fn that throws, or that returns a non-string `:text` (the usual
`(string/trim (os/sh ...))` mistake — `os/sh` returns a map), is shown as a
red `id!` badge instead of taking down the frame. `:text` must be a string;
for a live clock, `(.Format (now) "15:04")` is cheap enough to run every
paint, `os/sh` is not.

A minor mode's `:status`/`:status-right` lighters (see keycast above) still
work unchanged: they're aggregated into the built-in `:modes-status`/
`:modes-status-right` segments, which you can recolor, move, or remove the
same as any other segment.

## Scripting

Everything above is wired up in [`legmacs/bindings.lg`](../legmacs/bindings.lg)
using exactly the API available to you. There is no distinction between
"built-in" and "user" commands.

The two building blocks, from [`legmacs/keymap.lg`](../legmacs/keymap.lg):

```clojure
(require '[legmacs.keymap :as km :refer [defcommand]])

;; defcommand both defines a normal function (state -> state) and registers
;; it under a matching keyword, so it can be bound to a key or run from M-x.
(defcommand insert-banner "Insert a banner." [state]
  (legmacs.buffer/insert-string state "<<< hello >>>"))

;; bind-key! takes a space-separated chord sequence; multi-key sequences
;; build nested prefix keymaps automatically, same as the built-in C-x map.
(km/bind-key! "C-c b" :insert-banner)
```

Commands that need text from the user (find-file, search, M-x) go through
[`legmacs/minibuffer.lg`](../legmacs/minibuffer.lg), which is just as available
to your own commands:

```clojure
(require '[legmacs.minibuffer :as mb])

(defcommand insert-doubled "Prompt for text, insert it twice." [state]
  (mb/open state "Double: "
           (fn [state text] (legmacs.buffer/insert-string state (str text text)))))
```

`mb/open` takes an optional 4th arg for `:on-cancel`, `:completer`, and
`:initial` text. Pass `:completer` (a `(fn [input] -> [candidate ...])`)
and `TAB` completes against it using the same policy `M-x` and `find-file`
use (see [`legmacs/completion.lg`](../legmacs/completion.lg)).

`legmacs.buffer` has the rest of the editing primitives (movement, killing,
undo, etc.) if you want to build a command out of them rather than starting
from scratch. See [`legmacs/commands.lg`](../legmacs/commands.lg) for the full
built-in set, and [`legmacs/buffer.lg`](../legmacs/buffer.lg) for what it's
built out of.

### Loading your own config

Run the editor via `bin/legmacs` (see Running, above), then drop a file at
`~/.config/legmacs/legmacs_init.lg`:

```clojure
(ns legmacs-init
  (:require [legmacs.keymap :as km :refer [defcommand]]))

;; ... your bind-key!/defcommand calls here ...
```

The namespace must be named `legmacs-init`, that's what `main.lg` requires
on startup, after the default bindings are installed, so anything you bind
here overrides a default with the same key. See
[`examples/legmacs_init.example.lg`](../examples/legmacs_init.example.lg) for a
complete, working example (custom motions, a minibuffer-driven command).

Set `LEGMACS_CONFIG_DIR` to use a different directory than
`~/.config/legmacs`.

## Architecture

| File | Responsibility |
|---|---|
| [`legmacs/buffer.lg`](../legmacs/buffer.lg) | Pure text-buffer ops: editing, movement, undo, scrolling. No I/O. |
| [`legmacs/keys.lg`](../legmacs/keys.lg) | Raw terminal bytes → canonical chord names (`"C-x"`, `"M-w"`, `"RET"`, ...). |
| [`legmacs/keymap.lg`](../legmacs/keymap.lg) | The command registry and key-binding tree. The scripting surface. |
| [`legmacs/minibuffer.lg`](../legmacs/minibuffer.lg) | The bottom-line prompt, as plain state plus closures. Knows a prompt *has* a completer, not what kind. |
| [`legmacs/completion.lg`](../legmacs/completion.lg) | Fuzzy (subsequence) matching and scoring, plus the completers it feeds: command names, filesystem paths. (The older extend-to-common-prefix policy still lives here too.) |
| [`legmacs/commands.lg`](../legmacs/commands.lg) | The built-in commands, defined with `defcommand`. |
| [`legmacs/bindings.lg`](../legmacs/bindings.lg) | The default keymap, defined with `bind-key!`. |
| [`legmacs/modes.lg`](../legmacs/modes.lg) | Mode registry: a keymap that shadows the global one, a line highlighter (stateless, or carrying state across lines for multi-line constructs), an after-every-command hook, a mode-line lighter, plus filename → mode auto-detection. Majors go in the buffer's `:mode` slot; minor modes stack in `:minor-modes` and shadow the major. |
| [`legmacs/indent.lg`](../legmacs/indent.lg) | Where should this line start? One pure function driven by the language's `:indent` rules (bracket depth, or the line above plus keyword rules, or the language's own `:fn`), plus the edit that applies the answer without losing point. |
| [`legmacs/lisp_syntax.lg`](../legmacs/lisp_syntax.lg) | Pure bracket/string/comment scanner. What paren matching, syntax highlighting, auto-indent, and expand-region are all built on. |
| [`legmacs/modes/letgo.lg`](../legmacs/modes/letgo.lg) | let-go-mode: structural form discovery, in-process eval, async eval-and-replace, syntax highlighting, paren matching, auto-indent, expand-region, auto-paired brackets. Registered for `*scratch*`, `.lg` files, and (syntax-only) `.clj`/`.cljc`/`.cljs`/`.bb`/`.edn` files. |
| [`legmacs/modes/repl.lg`](../legmacs/modes/repl.lg) | The `*repl*` buffer (`C-c C-z`): plain let-go-mode plus a `:repl` minor mode that reinterprets `RET` as evaluate-if-complete, else newline-and-indent. |
| [`legmacs/nrepl.lg`](../legmacs/nrepl.lg) | nREPL client (`C-c M-c` / `C-c C-q`): bencode round trips over `net`, one connection per project root, registered as the `:nrepl` eval backend and completer for buffers under a connected root. |
| [`legmacs/jack_in.lg`](../legmacs/jack_in.lg) | `C-c M-j`: launch configs (lgx, Clojure CLI, Babashka, lg), project detection, and the server process's lifecycle -- spawn under `sh`, wait for the port, stop on disconnect or exit. |
| [`legmacs/modes/letgo_lookup.lg`](../legmacs/modes/letgo_lookup.lg) | eldoc, `C-c C-d d` doc and `M-.`/`M-,` goto-definition for let-go-mode, from the editor's VM or the buffer's nREPL server. |
| [`legmacs/modes/letgo_complete.lg`](../legmacs/modes/letgo_complete.lg) | Symbol completion for let-go-mode (`TAB` / `M-TAB`): in-process from the editor's VM, or from the buffer's nREPL server. |
| [`legmacs/vibe.lg`](../legmacs/vibe.lg) | `(vibe "...")`: ask an LLM for let-go code, deriving surrounding-buffer context when invoked by generic `C-c C-v`, then use a tagged eval-result handler to rewrite namespaces and add missing `(:require ...)`s. |
| [`legmacs/modes/markdown.lg`](../legmacs/modes/markdown.lg) | markdown-mode: syntax highlighting only (headers, emphasis, code, links, quotes, lists, rules, and fenced code blocks carried across lines). Registered for `.md`/`.markdown` files. |
| [`legmacs/modes/prog.lg`](../legmacs/modes/prog.lg) | The language pack: one spec-driven line scanner (comments, strings, keyword/type/constant sets, `#directives`, `$variables`, call sites, and multi-line strings carried across lines) behind major modes for Go, JS/TS, Python, C/C++, shell, rc, Rust, JSON, YAML, TOML, Lua, Ruby, SQL, Dockerfile, CSS, HTML, Zig, Java, Kotlin, Swift, C#, PHP, and Makefile. `register-prog-mode!` is the one-call way to add a language; the same spec map also becomes the mode's `:syntax-spec`. |
| [`legmacs/prog_syntax.lg`](../legmacs/prog_syntax.lg) | A full-buffer bracket/string/comment scanner like `legmacs.lisp-syntax`, but spec-driven instead of Lisp-specific -- what the generic structural modes below are built on. |
| [`legmacs/modes/structural.lg`](../legmacs/modes/structural.lg) | Three generic minor modes -- `:paren-match`, `:electric-pair`, `:auto-indent` (`RET`, `TAB`, dedent-on-closer, and opening a bracket pair out into a block) -- driven by whatever `:syntax-spec` the buffer's major mode declares; no-op (plain newline/self-insert) when it has none. `legmacs.modes/switch-to-mode` auto-enables all three for any major mode with a spec, so the whole language pack gets them by default. |
| [`legmacs/modes/crutch.lg`](../legmacs/modes/crutch.lg) | CRUTCH: vi-style modal editing as a bundled minor mode (`M-x crutch-mode`). A worked example of the minor-mode/`:keymap`-fn/`:suppress-self-insert?` machinery. |
| [`legmacs/modes/keycast.lg`](../legmacs/modes/keycast.lg) | keycast: a right-aligned live keystroke preview (`M-x keycast-mode`). A pure observer minor mode built on `:last-chord` + `:status-right`. |
| [`legmacs/modes/help.lg`](../legmacs/modes/help.lg) | help-mode: the read-only, syntax-highlighted major mode for `*Help*` buffers (`C-h b`/`describe-bindings`, `C-h k`, `C-h f`, `C-h m`). Highlight-only, like markdown-mode. |
| [`legmacs/dispatch.lg`](../legmacs/dispatch.lg) | One key chord → a new editor state. Layers every active mode's keymap (minor modes, then major) over the global one; runs their `:after-command` hooks on the way out. |
| [`legmacs/theme.lg`](../legmacs/theme.lg) | The editor's color palette as data (face keyword → `{:fg :bg :bold ...}`). `load-theme!` switches a named map; `set-theme!`/`set-face!` merge; `register-theme!` is how user palettes get into `M-x load-theme`. Render compiles the atom into ANSI. |
| [`legmacs/modeline.lg`](../legmacs/modeline.lg) | The mode-line's registry: named, ordered, independently-colorable segments plus a theme (bg/fg + separator), all plain data. `register-modeline-segment!`/`set-modeline-theme!` are the scripting surface; the built-in segments (buffer name, mode, position, ...) are registered the same way. |
| [`legmacs/render.lg`](../legmacs/render.lg) | Editor state → one ANSI string per frame. A per-column fg/bg span compositor, so region highlight/paren-match/syntax colors can all coexist on one line; the mode line is a block compositor over `legmacs.modeline`'s segments instead. Draws each window of the workspace into its own rectangle (per-window mode line, `│` dividers), the echo area under all of them. |
| [`legmacs/windows.lg`](../legmacs/windows.lg) | The window tree: split panes as pure data (leaves show a buffer through a per-window view; splits stack `:below` or sit `:beside` with proportional weights) plus the layout geometry that turns the tree into screen rectangles and divider positions. |
| [`legmacs/buffers.lg`](../legmacs/buffers.lg) | Wraps a collection of buffers — and the window tree showing them — behind the exact same flat state shape everything above expects. Also the home of `:pending-task` (`arm-task` / `run-pending-task`) and `:async-jobs` (`spawn-task` / `drain-jobs`). See below. |
| [`main.lg`](../main.lg) | Wires it all up; the only place that touches the terminal. |

The editor state is one plain map, threaded through a `loop`/`recur` in
`main.lg`, so there's no global mutable buffer. The only genuinely global,
mutable things are the command registry and keymap (both atoms), because
that's exactly what needs to be visible to a user's config file.

Every module above `main.lg` is pure (state in, state or string out), which
is what makes it possible to unit-test the entire editor (buffer edits, key
parsing, command dispatch, even rendering) without a terminal. See `test/`.

### How multiple buffers fit in without touching any of that

`legmacs.dispatch`/`render`/`commands` only ever see one flat state, which
was true before multiple buffers existed and it's still true now.
`legmacs.buffers` holds a "workspace" (every buffer, plus the handful of
things that are genuinely global rather than per-buffer: the message, an
open minibuffer/overlay, a pending prefix key, a `:pending-task` queue,
`:async-jobs`, the one shared clipboard).
`bufs/current` merges the active buffer with those into one flat map,
exactly what dispatch already expects; `bufs/put-current` splits whatever
comes back apart again. `main.lg`'s loop is just:

```clojure
(let [workspace (render! workspace)]
  ...
  (recur (bufs/put-current workspace (dispatch/dispatch (bufs/current workspace) chord))))
```

The one thing a normal `(fn [state] -> state)` command can't express this
way is "switch to a different buffer": from inside one buffer's flat view,
there is no other buffer to switch to. Commands that need to
(`switch-to-buffer`, `kill-buffer`, `find-file` opening a second buffer)
say so by setting `:buffer-command` on the state they return (e.g.
`{:op :switch :id 2}`); `put-current` reads that, applies it to the
workspace, and clears it. It's ordinary state, not a side channel, and
it's the only new concept multiple buffers needed.

### ...and windows are the same trick, one level up

Splits reuse both halves of that design. The workspace holds one *window
tree* ([`legmacs/windows.lg`](../legmacs/windows.lg)): each leaf is a
window showing some buffer through its own `:view` — the cursor, scroll
position, and scroll bookkeeping, i.e. everything about *looking at* a
buffer rather than the buffer itself. That per-window view is what lets
two windows show one buffer at two different spots (Emacs's window-point).
`bufs/current` now flattens the *active window's* buffer-plus-view;
`put-current` folds the view slice back into that window the same way it
folds shared fields back onto the workspace. And the window commands are
just more `:buffer-command` ops — `{:op :split :dir :below}`,
`{:op :other-window}`, `{:op :delete-window}` — while the existing buffer
ops (`:switch`, `:open-new`, ...) now mean "in the active window", and
`:kill` re-points any window that showed the killed buffer. Dispatch and
the whole command set still see one flat state and never learned windows
exist.

Rendering is the one layer that genuinely sees more than one window per
frame: `render/prepare-workspace` runs the scroll-to-fit pass per window
(full treatment for the active window, clamping for the rest — an
inactive window shouldn't scroll on its own), and
`render/workspace-frame` lays the tree out into rectangles
(`windows/layout` — proportional weights, so a terminal resize re-flows
the same shape), draws each window with its own mode line (dimmed when
inactive), fills the `│` divider columns between side-by-side windows,
and keeps the echo/minibuffer area global across the bottom. Every cell
of every rectangle is still written every frame, so the no-clear,
no-flicker invariant survives splits unchanged.

## Testing

```sh
lg test/run.lg
```

## Known limitations

- Windows split and cycle, but there's no interactive *resizing* of an
  existing split yet (`C-x ^`/`C-x {` style) — proportions are set by the
  splits themselves.
- `C-x C-c` only *saves* the current buffer. Other modified buffers get a
  warning (how many, not which ones) before it lets you quit anyway, but
  it won't save them for you, so switch to them and save individually first
  if you want to keep those changes too.
- `term/size` is the only thing that knows the real terminal size; resizing
  while running is handled (see `term/read-key`'s SIGWINCH wake-up in
  let-go), but there's no manual test harness for it here.

The architecture doesn't fight you on any of these; they're scoped out for
now, not designed against.

## A let-go quirk worth knowing if you're hacking on the eval commands

`(def x (load-string "..."))` as a top-level form in a script breaks
resolving the *next* top-level form's symbols (reproducible with `lg -e`,
nothing legmacs-specific). It doesn't affect legmacs itself: `eval-buffer`
calls `load-string` from inside an already-compiled function, called
during `main`'s already-running loop, never as a script's own top-level
form, confirmed safe by both the test suite and live runs. Just don't be
surprised if you hit it experimenting with `load-string` at the REPL.
