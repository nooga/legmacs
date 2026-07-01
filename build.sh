#!/bin/sh
# Full AOT build for legmacs:
#   1. builds `lg` fresh from a let-go source checkout (so local compiler
#      fixes/changes are picked up -- the system-installed `lg` may be an
#      older release)
#   2. AOT-compiles every legmacs/*.lg namespace to native Go via let-go's
#      scripts/lg-compile (direct cross-package calls; legmacs.main stays
#      interpreted -- its ns leaf "main" can't be a Go-importable package)
#   3. compiles main.lg + all its required legmacs.* namespaces into one
#      precompiled bytecode bundle build/legmacs.lgb (main.go go:embeds this
#      and replays it at boot instead of parsing raw source -- the gogen Go
#      packages from step 2 still override the hot functions natively)
#   4. go builds the standalone binary
#
# Usage:
#   ./build.sh
#
# Requires a let-go source checkout -- defaults to ../let-go (sibling of
# this repo); override with LETGO_DIR=/path/to/let-go.

set -e

legmacs_dir=$(cd "$(dirname "$0")" && pwd)
letgo_dir=$(cd "${LETGO_DIR:-"$legmacs_dir/../let-go"}" && pwd)
build_dir="$legmacs_dir/build"

echo "==> building lg from $letgo_dir"
lg_bin=$(mktemp -t legmacs-lg-XXXXXX)
trap 'rm -f "$lg_bin"' EXIT
(cd "$letgo_dir" && go build -o "$lg_bin" .)

echo "==> AOT-compiling legmacs/*.lg -> $build_dir/legmacs"
rm -rf "$build_dir/legmacs"
lg_files=$(find "$legmacs_dir/legmacs" -name '*.lg' | sort)
(cd "$letgo_dir" &&
  LG_SOURCE_PATHS=".:pkg/rt/gogen:$legmacs_dir" \
  "$lg_bin" scripts/lg-compile "$build_dir" legmacs/build $lg_files)

echo "==> compiling bytecode bundle -> $build_dir/legmacs.lgb"
# `lg -c` sets *compiling-aot* true before running top-level forms, so
# main.lg's guarded (when-not *compiling-aot* (main)) does NOT launch the
# editor here; it just harvests every required namespace chunk into the
# bundle. Run from $legmacs_dir so cwd-relative ns resolution finds
# legmacs/*.lg.
(cd "$legmacs_dir" && "$lg_bin" -c "$build_dir/legmacs.lgb" main.lg)

echo "==> go build"
mkdir -p "$build_dir/bin"
(cd "$build_dir" && go build -o bin/legmacs .)

echo "done: $build_dir/bin/legmacs"
