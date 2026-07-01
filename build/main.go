// legmacs standalone driver: boots a let-go runtime, wires in the
// gogen-lowered legmacs.* packages (below) as native overrides for their
// interpreted counterparts, then loads a precompiled bytecode bundle
// (legmacs.lgb) instead of raw .lg source.
//
// The bundle is produced by `lg -c legmacs.lgb main.lg` (see build.sh): it
// contains every legmacs.* namespace chunk plus legmacs.main, in dependency
// order. We replay each chunk here the same way the stock `lg` binary boots a
// bundled executable (pkg cmd runLGB), with ONE addition: after replaying each
// namespace we drain its gogen-registered Go-native overrides. The plain
// chunk-replay path does NOT go through the resolver's Load (which is what
// normally calls ApplyGoOverrides), so without this the lowered Go functions
// would never clobber the interpreted ones and the binary would run pure
// bytecode. legmacs.main itself is NOT lowered — its ns leaf "main" collides
// with Go's package-main convention — so its chunk stays interpreted; it's
// thin orchestration glue and isn't perf-sensitive.
//
// The bundle is embedded (see bundleLGB below) so the built binary is
// self-contained: no source extraction, no cwd-relative namespace resolution.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"

	"github.com/nooga/let-go/pkg/api"
	"github.com/nooga/let-go/pkg/bytecode"
	"github.com/nooga/let-go/pkg/rt"
	"github.com/nooga/let-go/pkg/vm"

	_ "legmacs/build/legmacs/bindings"
	_ "legmacs/build/legmacs/buffer"
	_ "legmacs/build/legmacs/buffers"
	_ "legmacs/build/legmacs/commands"
	_ "legmacs/build/legmacs/completion"
	_ "legmacs/build/legmacs/dispatch"
	_ "legmacs/build/legmacs/keymap"
	_ "legmacs/build/legmacs/keys"
	_ "legmacs/build/legmacs/lisp_syntax"
	_ "legmacs/build/legmacs/minibuffer"
	_ "legmacs/build/legmacs/modes"
	_ "legmacs/build/legmacs/modes/letgo"
	_ "legmacs/build/legmacs/modes/markdown"
	_ "legmacs/build/legmacs/render"
)

//go:embed legmacs.lgb
var bundleLGB []byte

func main() {
	// api.NewLetGo wires up the compiler, resolver, and namespace loader
	// (rt.SetNSLoader) so embedded core namespaces (term, os, string, …)
	// referenced by the bundle load on demand. core is already loaded via
	// pkg/compiler's init().
	if _, err := api.NewLetGo("user"); err != nil {
		fmt.Fprintln(os.Stderr, "legmacs: starting runtime:", err)
		os.Exit(1)
	}

	// Set before any chunk runs — legmacs.main's top level reads it.
	rt.CoreNS.Lookup("*command-line-args*").(*vm.Var).SetRoot(commandLineArgsValue(os.Args[1:]))

	store, err := rt.NewDefaultFileStorage("legmacs")
	if err == nil {
		if v := rt.LookupCoreVar("*storage*"); v != nil {
			v.SetRoot(vm.NewBoxed(store))
		}
	}

	// resolve fills var references during decode. For namespaces defined by
	// the bundle it hands back a stub the chunk fills when it runs; for
	// already-registered namespaces (core, pure-Go term) it returns the real
	// var. Same callback the stock `lg` bundle loader uses.
	resolve := func(nsName, name string) *vm.Var {
		n := rt.DefNSBare(nsName)
		if v := n.LookupLocal(vm.Symbol(name)); v != nil {
			return v
		}
		return n.DefStub(name)
	}
	unit, err := bytecode.DecodeToExecUnit(bytes.NewReader(bundleLGB), resolve)
	if err != nil {
		fmt.Fprintln(os.Stderr, "legmacs: decoding embedded bundle:", err)
		os.Exit(1)
	}

	// Replay each namespace chunk in dependency order, draining gogen native
	// overrides for it once its interpreted vars exist. Skip the main chunk —
	// it runs last.
	for _, name := range unit.NSOrder {
		chunk := unit.NSChunks[name]
		if chunk == nil || chunk == unit.MainChunk {
			continue
		}
		f := vm.NewFrame(chunk, nil)
		_, err := f.RunProtected()
		vm.ReleaseFrame(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "legmacs: loading namespace %s:\n", name)
			fmt.Fprint(os.Stderr, vm.FormatError(err))
			os.Exit(1)
		}
		rt.ApplyGoOverrides(rt.LookupNS(name))
	}

	// main chunk last: runs legmacs.main's top level, including
	// (when-not *compiling-aot* (main)) — *compiling-aot* is false at runtime,
	// so this launches the editor.
	f := vm.NewFrame(unit.MainChunk, nil)
	_, err = f.RunProtected()
	vm.ReleaseFrame(f)
	if err != nil {
		fmt.Fprint(os.Stderr, vm.FormatError(err))
		os.Exit(1)
	}
}

func commandLineArgsValue(args []string) vm.Value {
	if len(args) == 0 {
		return vm.NIL
	}
	vs := make([]vm.Value, len(args))
	for i, a := range args {
		vs[i] = vm.String(a)
	}
	return vm.NewList(vs)
}
