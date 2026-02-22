package classify

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// maxCallerWeight is the maximum weight for caller dependency
// signals.
const maxCallerWeight = 15

// AnalyzeCallerSignal scans TypesInfo.Uses across module packages
// to find call sites of the target function and computes a weight
// proportional to the ratio of callers that use/depend on the
// side effect.
func AnalyzeCallerSignal(
	funcObj types.Object,
	effectType taxonomy.SideEffectType,
	modulePkgs []*packages.Package,
) taxonomy.Signal {
	if funcObj == nil {
		return taxonomy.Signal{}
	}

	callerCount := countCallers(funcObj, modulePkgs)
	if callerCount == 0 {
		return taxonomy.Signal{}
	}

	// Weight is proportional to caller count, capped at max.
	// 1 caller = 5, 2-3 callers = 10, 4+ callers = 15.
	weight := 5
	if callerCount >= 4 {
		weight = maxCallerWeight
	} else if callerCount >= 2 {
		weight = 10
	}

	return taxonomy.Signal{
		Source: "caller",
		Weight: weight,
		Reasoning: fmt.Sprintf(
			"%d caller(s) reference this function",
			callerCount,
		),
	}
}

// funcKey returns a stable string identity for a types.Object
// that is safe to compare across separate packages.Load calls.
// Pointer identity cannot be used because the target package and
// module packages may be loaded in different type-checker universes.
//
// For methods, the receiver type name is included to avoid collisions
// when two types in the same package share a method name (e.g.,
// FileStore.Write and MemStore.Write are distinct keys).
func funcKey(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	fn, ok := obj.(*types.Func)
	if ok {
		sig, sigOk := fn.Type().(*types.Signature)
		if sigOk && sig.Recv() != nil {
			// Include the receiver's named type to disambiguate
			// methods with the same name on different types.
			recv := sig.Recv().Type()
			// Unwrap pointer to get the named type.
			if ptr, ptrOk := recv.(*types.Pointer); ptrOk {
				recv = ptr.Elem()
			}
			if named, namedOk := recv.(*types.Named); namedOk {
				return obj.Pkg().Path() + "." +
					named.Obj().Name() + "." + obj.Name()
			}
			// If recv is a *types.TypeParam (generic type parameter),
			// neither branch above matches. We fall through to the plain
			// key (pkg.Name) — two generic methods with the same name on
			// different type parameters in the same package would collide.
			// This is a known limitation for generic code; non-generic
			// Go (the primary Gaze target) is unaffected.
		}
	}
	return obj.Pkg().Path() + "." + obj.Name()
}

// countCallers counts the number of distinct packages that
// reference the given function object via TypesInfo.Uses.
// It uses string-based identity (package path + name) rather than
// pointer identity so that it works correctly when the target
// package and the module packages are loaded in separate
// packages.Load calls.
func countCallers(funcObj types.Object, pkgs []*packages.Package) int {
	key := funcKey(funcObj)
	if key == "" {
		return 0
	}

	targetPkgPath := funcObj.Pkg().Path()
	count := 0

	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		// Skip the package that defines the function.
		if pkg.PkgPath == targetPkgPath {
			continue
		}

		for _, obj := range pkg.TypesInfo.Uses {
			if funcKey(obj) == key {
				count++
				break // Count each package only once.
			}
		}
	}

	return count
}
