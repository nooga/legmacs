# Acme-inspired Look and Execute experiment

LegMacs can explore Acme's text-as-interface model without turning the whole
editor into an Acme clone and without precluding a future Dired mode.  The
first experiments are optional keyboard-driven `Look` and `Execute` actions.

## First slice

`look-at-point` obtains a target from the active region, or from a
filename-like token touching point when there is no selection.  It interprets
relative targets in the current buffer's `:default-directory`.

- An existing regular file opens through LegMacs's normal file/buffer path.
- An existing directory opens as a plain writable text buffer containing one
  entry per line; subdirectories carry a trailing slash.
- Missing text reports an error.  Look never creates a new file.
- Re-looking an open file or directory reuses its buffer.
- Directory-buffer edits are only text edits and never mutate the filesystem.

The directory listing has no special major mode and suppresses no ordinary
editing keys.  This is the intentional contrast with Dired: it is contextual
text on which the same global Look action can operate.

## Optional loading

The extension is inert unless a user loads it.  Requiring its namespace makes
`look-at-point` and `execute-at-point` available through `M-x`; calling
`enable!` additionally binds F7 to Execute and F8 to Look:

```clojure
(ns legmacs-init
  (:require [legmacs.extensions.acme :as acme]))

(acme/enable!) ; F7 Execute, F8 Look
```

## Directory context

Every buffer may carry `:default-directory`, analogous to Acme's window
directory and Emacs's buffer-local `default-directory`:

- file buffers use the file's containing directory;
- directory listings use the listed directory;
- generated buffers can inherit the originating buffer's context;
- buffers without an explicit value fall back to their filename or the
  editor process directory.

This context is deliberately useful outside the extension: a future shell
command, compiler-output buffer, Dired mode, or REPL can use the same field.

## Deferred behavior

The first slice does not implement Acme addresses (`file.c:27`), literal-text
search, include lookup, plumber fallback, automatic filesystem mutation, or
Dired operations.  Those should be separate additions
after the basic Look workflow has been tested interactively.

Display policy is also separate from target resolution.  This first command
uses the active LegMacs window.  A future `look-at-point-other-window` can
split or focus another window without changing path recognition.

## Execute slice

`execute-at-point` is the keyboard analogue of Acme button 2 and is bound to
F7 by `enable!`.  It uses the same text expansion as Look: a non-empty active
region wins; otherwise the conservative filename-like token touching point is
the command.  This matches the expansion documented by 9front release 11554's
`acme(1)` and the Acme paper.

The command is evaluated by `rc -p -c` in the originating buffer's
`:default-directory`.  On Plan 9, `-p` prevents a new rc from importing
CPU-session `/env/fn#*` entries itself.  The script creates a private
environment group with `rfork e` and removes copied `fn#*` entries there, so
nested rc scripts also start cleanly without changing LegMacs' environment.
It then establishes `path=(. /bin)` for Acme-like local-first lookup.  A safely
quoted `cd` prefix gives the child its context without changing the LegMacs
process directory.  Standard input is not connected; stdout and stderr are
captured after the command completes.

Each directory has one ordinary writable buffer named `dir/+Errors`.  F7
opens or focuses that buffer, and later executions from the same directory
append a blank-line-separated block containing the command and its output.
Edits in this buffer are only text edits.

This is still a non-interactive slice: no live streaming, no stdin, no
process kill on C-g.  The command itself is asynchronous -- `os/sh` runs in
a future so you can keep typing.

`execute-at-point` does not call `rc` itself; it snapshots the script and
`(bufs/spawn-task …)` (see the header comment in `legmacs/buffers.lg`).
The I/O thunk runs in a goroutine; `drain-jobs` folds the result into
`+Errors` on the main thread when the promise realizes.  The editor keeps
reading keys while rc runs.  C-g discards the result without killing the
process.  A throw in `:then` is an echo-area message, not a session death.
Killing the process, streaming output, and job ids are still later.
