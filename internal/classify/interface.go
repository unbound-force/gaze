// Package classify implements the contractual classification engine.
package classify

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// maxInterfaceWeight is the maximum weight for interface
// satisfaction signals.
const maxInterfaceWeight = 30

// analyzeInterfaceSignal checks if the function's receiver type
// satisfies any interface defined in the module. When a method's
// side effect matches the interface's method signature, it is
// strong contractual evidence. Returns a zero signal for
// non-method functions.
//
// ifaces is a pre-computed slice from collectInterfaces; callers
// should compute this once per Classify invocation to avoid O(n²)
// interface collection across side effects.
//
// This function is intentionally unexported because its ifaces
// parameter uses the unexported namedInterface type. All callers
// go through Classify() which pre-computes the interface list.
func analyzeInterfaceSignal(
	funcName string,
	receiverType types.Type,
	_ taxonomy.SideEffectType,
	ifaces []namedInterface,
) taxonomy.Signal {
	if receiverType == nil {
		return taxonomy.Signal{}
	}

	if len(ifaces) == 0 {
		return taxonomy.Signal{}
	}

	// Check if the receiver type (or pointer to it) satisfies any
	// interface that declares a method with the same name.
	for _, iface := range ifaces {
		if !satisfies(receiverType, iface.iface) {
			continue
		}

		// The type satisfies this interface. Check if the method
		// is declared in the interface.
		for i := 0; i < iface.iface.NumMethods(); i++ {
			method := iface.iface.Method(i)
			if method.Name() != funcName {
				continue
			}

			// The method is in the interface — this side effect
			// is contractual.
			return taxonomy.Signal{
				Source: "interface",
				Weight: maxInterfaceWeight,
				Reasoning: fmt.Sprintf(
					"method %s satisfies interface %s",
					funcName, iface.name,
				),
			}
		}
	}

	return taxonomy.Signal{}
}

// namedInterface pairs an interface type with its qualified name.
type namedInterface struct {
	name  string
	iface *types.Interface
}

// collectInterfaces scans all packages and returns every interface
// type found.
func collectInterfaces(pkgs []*packages.Package) []namedInterface {
	var result []namedInterface
	seen := make(map[*types.Interface]bool)

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			iface, ok := tn.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			if seen[iface] {
				continue
			}
			seen[iface] = true
			result = append(result, namedInterface{
				name:  pkg.PkgPath + "." + name,
				iface: iface,
			})
		}
	}

	return result
}

// satisfies checks if typ or *typ implements the given interface.
func satisfies(typ types.Type, iface *types.Interface) bool {
	if types.Implements(typ, iface) {
		return true
	}
	// Also check pointer to type.
	ptr := types.NewPointer(typ)
	return types.Implements(ptr, iface)
}
