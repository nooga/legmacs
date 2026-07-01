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
| `M-f` / `M-b` | forward/backward word |
| `M-<` / `M->` | start/end of buffer |
| `C-v` / `M-v` | page down/up |
| `RET` | newline |
| `DEL` | delete backward |
| `C-d` / Delete | delete forward |
| `C-k` | kill to end of line |
| `M-k` | kill whole line |
| `C-y` | yank (paste) |
| `C-SPC` | set the mark (start a region) |
| `C-w` | kill (cut) the region |
| `M-w` | copy the region, without deleting it |
| `C-x C-x` | swap point and mark |
| `C-_` / `C-x u` | undo |
| `M-_` | redo |
| `C-s` | incremental regexp search (swiper-style); `C-s`/`C-r` step through matches |
| `C-x C-s` | save |
| `C-x C-w` | write file (always prompts) |
| `C-x C-f` | open file in a new buffer (or switch to it, if already open) |
| `C-x b` | switch to a buffer by name (`TAB` completes; a new name creates one) |
| `C-x C-b` | list open buffers |
| `C-x k` | kill (close) the current buffer |
| `C-x <right>` / `C-x <left>` | cycle to the next / previous buffer |
| `C-x C-c` | quit (prompts if the current buffer has unsaved changes) |
| `C-g` | cancel / quit-out-of-here |
| `M-x` | run a command by name (`TAB` completes) |
| `C-x C-q` | toggle the current buffer read-only |
| `C-h` | show all key bindings (in a read-only `*Help*` buffer) |

Completing prompts (`M-x`, `C-x C-f`, `C-x C-s`, `C-x C-w`, `C-x b`) show a
live, vertico-style vertical list that narrows as you type. Matching is
*fuzzy* (the characters you type just have to appear in order, so `nxtl`
finds `next-line`), best match highlighted at the top. (File prompts match
fuzzily within the current directory segment: `src/cmp` narrows the entries
of `src/`.) `C-n`/`C-p` (or the arrows) move the highlight, `TAB` pulls the
highlighted candidate up into the input line (so you can descend into a
directory or refine further). `M-x` and the buffer/mode pickers submit the
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

`C-h` (`describe-bindings`) opens the key-binding list in a real, read-only,
syntax-highlighted `*Help*` buffer rather than a one-shot overlay, so you
can scroll it, search it with `C-s`, and switch away and back like any other
buffer. Read-only is a general buffer property (`C-x C-q` toggles it on any
buffer); the buffer mutation primitives in
[`legmacs/buffer.lg`](../legmacs/buffer.lg) refuse edits when it's set, so no
command (built-in or from your config) can modify a read-only buffer, while
movement, scrolling, and search keep working.

Multiple buffers, but still one window, no splits. `lg main.lg a.txt b.txt`
opens both (the first one focused); `C-x C-f` on a new path opens another
without touching whatever you were already editing, and on a path that's
already open just switches to it rather than loading it twice. Buffer names
are basenames (`a.txt`, not the full path), deduplicated like Emacs's if two
open files share one (`a.txt`, `a.txt<2>`). Undo history, the mark, and
paren-match are all per-buffer; the echo-area message, minibuffer, and the
one shared clipboard are not, so they're exactly as visible switching buffers
as they would be switching windows in something that had windows.

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
into it automatically:

| Keys | Action |
|---|---|
| `C-x C-e` | evaluate the s-expression before point, show the result in the echo area |
| `C-j` | evaluate the s-expression before point, insert the result right there |
| `C-c C-e` | evaluate the whole buffer |
| `RET` | newline, auto-indented by paren depth |
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

Syntax highlighting (strings, comments, a curated set of special-form
names), paren matching, auto-indent, and expand-region all share one
scanner, [`legmacs/lisp_syntax.lg`](../legmacs/lisp_syntax.lg), that finds
brackets, strings, and comments in one pass, correctly skipping character
literals (`\(`, `\"`, `\\`, ...) so they don't get mistaken for real
structure. Paren matching and expand-region scan the whole buffer, so they
work correctly across line breaks; the per-line syntax highlighter doesn't
carry state between lines, so a string literal that spans multiple lines
will mis-highlight at the boundary (a limitation most lightweight
highlighters share; getting it exactly right needs the kind of continuation
state Emacs tracks via syntax-table text properties).

See [`legmacs/modes/letgo.lg`](../legmacs/modes/letgo.lg) for all of the
above. Registering your own mode from `legmacs-init.lg`, with or without
these features, is the same `register-mode!`/`register-auto-mode!` call.

Any file ending in `.md` or `.markdown` opens into **markdown-mode**
(highlighting only, no key bindings of its own, so every global key still
works): ATX headers (`#` through `######`), `**bold**`/`__bold__`,
`*italic*`/`_italic_` (a bare `_` inside a word, like `some_var_name`,
correctly doesn't count, markdown's own rule, unlike `*`), `` `inline
code` ``, `[links](url)`/`![images](url)`, `>` blockquotes, `-`/`*`/`+`/`1.`
list markers, and `---`/`***`/`___` horizontal rules; see
[`legmacs/modes/markdown.lg`](../legmacs/modes/markdown.lg). Like
let-go-mode's highlighter, it's per-line with no state carried across lines,
so only a fenced-code-block's ` ``` ` delimiter lines are recognized, not
the code between them (the same limitation, for the same reason, as the
multi-line string case above).

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
| [`legmacs/modes.lg`](../legmacs/modes.lg) | Mode registry: a keymap that shadows the global one, a per-line highlighter, an after-every-command hook, a mode-line lighter, plus filename → mode auto-detection. Majors go in the buffer's `:mode` slot; minor modes stack in `:minor-modes` and shadow the major. |
| [`legmacs/lisp_syntax.lg`](../legmacs/lisp_syntax.lg) | Pure bracket/string/comment scanner. What paren matching, syntax highlighting, auto-indent, and expand-region are all built on. |
| [`legmacs/modes/letgo.lg`](../legmacs/modes/letgo.lg) | let-go-mode: in-process eval, syntax highlighting, paren matching, auto-indent, expand-region, auto-paired brackets. Registered for `*scratch*` and `.lg` files. |
| [`legmacs/modes/markdown.lg`](../legmacs/modes/markdown.lg) | markdown-mode: syntax highlighting only (headers, emphasis, code, links, quotes, lists, rules). Registered for `.md`/`.markdown` files. |
| [`legmacs/modes/crutch.lg`](../legmacs/modes/crutch.lg) | CRUTCH: vi-style modal editing as a bundled minor mode (`M-x crutch-mode`). A worked example of the minor-mode/`:keymap`-fn/`:suppress-self-insert?` machinery. |
| [`legmacs/modes/keycast.lg`](../legmacs/modes/keycast.lg) | keycast: a right-aligned live keystroke preview (`M-x keycast-mode`). A pure observer minor mode built on `:last-chord` + `:status-right`. |
| [`legmacs/modes/help.lg`](../legmacs/modes/help.lg) | help-mode: the read-only, syntax-highlighted major mode for `*Help*` buffers (see `C-h`/`describe-bindings`). Highlight-only, like markdown-mode. |
| [`legmacs/dispatch.lg`](../legmacs/dispatch.lg) | One key chord → a new editor state. Layers every active mode's keymap (minor modes, then major) over the global one; runs their `:after-command` hooks on the way out. |
| [`legmacs/modeline.lg`](../legmacs/modeline.lg) | The mode-line's registry: named, ordered, independently-colorable segments plus a theme (bg/fg + separator), all plain data. `register-modeline-segment!`/`set-modeline-theme!` are the scripting surface; the built-in segments (buffer name, mode, position, ...) are registered the same way. |
| [`legmacs/render.lg`](../legmacs/render.lg) | Editor state → one ANSI string per frame. A per-column fg/bg span compositor, so region highlight/paren-match/syntax colors can all coexist on one line; the mode line is a block compositor over `legmacs.modeline`'s segments instead. |
| [`legmacs/buffers.lg`](../legmacs/buffers.lg) | Wraps a collection of buffers behind the exact same flat state shape everything above expects. See below. |
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
open minibuffer/overlay, a pending prefix key, the one shared clipboard).
`bufs/current` merges the active buffer with those into one flat map,
exactly what dispatch already expects; `bufs/put-current` splits whatever
comes back apart again. `main.lg`'s loop is just:

```clojure
(let [flat (render! (bufs/current workspace))
      workspace (bufs/put-current workspace flat)]
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

## Testing

```sh
lg test/run.lg
```

## Known limitations

- One window: multiple buffers, but no splits, so you're always looking at
  exactly one of them at a time.
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
