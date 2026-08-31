// Package ssaguard provides a shared panic-recovery guard for SSA
// construction. SSA builds via golang.org/x/tools can panic on certain
// upstream bugs (e.g., generic type substitution under Go 1.25); this
// package isolates the recover() pattern so callers degrade gracefully
// instead of crashing.
package ssaguard

// SafeSSABuild calls buildFn and recovers from any panic it produces.
// It returns the recovered panic value, or nil if buildFn completed
// without panicking. Isolating the recover() pattern here lets it be
// tested independently of the SSA builder and shared by every SSA
// build site.
//
// Caller precondition — ssa.BuildSerially: callers MUST construct the
// SSA program with the ssa.BuildSerially mode flag (alongside
// ssa.InstantiateGenerics) before invoking SafeSSABuild(prog.Build).
// Go's recover() is goroutine-scoped and cannot catch panics raised in
// child goroutines. Without ssa.BuildSerially, prog.Build() spawns a
// child goroutine per package and any panic there escapes this guard,
// crashing the process. BuildSerially forces all construction onto the
// calling goroutine so the deferred recover() below can catch it. See
// specs/033-ssa-goroutine-panic for the invariant.
//
// SafeSSABuild does NOT validate the build mode at runtime: the mode
// flags are set by callers before ssautil.AllPackages, which is outside
// this guard's scope. The precondition is documented and enforced by
// convention, not by a runtime check.
func SafeSSABuild(buildFn func()) (panicVal any) {
	defer func() {
		panicVal = recover()
	}()
	buildFn()
	return nil
}
