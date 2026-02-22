// Package classify implements the contractual classification engine
// for Gaze, determining whether each side effect is contractual,
// incidental, or ambiguous using weighted confidence scoring.
package classify

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/jflowers/gaze/internal/config"
	"github.com/jflowers/gaze/internal/taxonomy"
)

// Options configures the classification engine.
type Options struct {
	// Config is the Gaze configuration. If nil, defaults are used.
	Config *config.GazeConfig

	// ModulePackages is the list of all packages in the module,
	// used for interface satisfaction and caller analysis.
	ModulePackages []*packages.Package

	// TargetPkg is the loaded target package (for AST access).
	TargetPkg *packages.Package

	// Verbose controls whether signal detail fields (SourceFile,
	// Excerpt, Reasoning) are populated.
	Verbose bool
}

// Classify classifies each side effect in the given analysis
// results using mechanical signal analyzers. It attaches a
// Classification to each SideEffect and returns the modified
// results.
func Classify(results []taxonomy.AnalysisResult, opts Options) []taxonomy.AnalysisResult {
	if opts.Config == nil {
		opts.Config = config.DefaultConfig()
	}

	// Build a lookup from function name to AST declaration and
	// types.Object for the target package.
	funcDecls := buildFuncDeclMap(opts.TargetPkg)
	funcObjs := buildFuncObjMap(opts.TargetPkg)

	// Pre-compute interfaces once to avoid O(n²) collection.
	ifaces := collectInterfaces(opts.ModulePackages)

	for i := range results {
		result := &results[i]
		funcName := result.Target.Function
		funcDecl := funcDecls[funcName]

		// Prefer a receiver-qualified lookup to avoid collisions
		// between methods with the same name on different types.
		funcObj := lookupFuncObj(funcObjs, result.Target.Receiver, funcName)

		// Determine receiver type if this is a method.
		var receiverType types.Type
		if funcObj != nil {
			if sig, ok := funcObj.Type().(*types.Signature); ok && sig.Recv() != nil {
				receiverType = sig.Recv().Type()
				// Unwrap pointer for interface checks.
				if ptr, ok := receiverType.(*types.Pointer); ok {
					receiverType = ptr.Elem()
				}
			}
		}

		for j := range result.SideEffects {
			se := &result.SideEffects[j]

			// For sentinel errors, the meaningful name is the
			// variable name (se.Target, e.g. "ErrNotFound"), not
			// the package-scope funcName ("<package>").
			namingName := funcName
			if se.Type == taxonomy.SentinelError && se.Target != "" {
				namingName = se.Target
			}

			signals := classifySideEffect(
				funcName, funcDecl, funcObj,
				receiverType, se.Type,
				namingName, ifaces, opts,
			)

			classification := ComputeScore(signals, opts.Config)

			// Strip detail fields if not verbose.
			if !opts.Verbose {
				for k := range classification.Signals {
					classification.Signals[k].SourceFile = ""
					classification.Signals[k].Excerpt = ""
					classification.Signals[k].Reasoning = ""
				}
			}

			se.Classification = &classification
		}
	}

	return results
}

// classifySideEffect runs all five mechanical signal analyzers
// for a single side effect and returns the collected signals.
// ifaces is the pre-computed interface list from collectInterfaces.
// namingName is the name used for naming-convention analysis; for
// sentinel errors it is the variable name (se.Target) rather than
// the enclosing funcName ("<package>").
func classifySideEffect(
	funcName string,
	funcDecl *ast.FuncDecl,
	funcObj types.Object,
	receiverType types.Type,
	effectType taxonomy.SideEffectType,
	namingName string,
	ifaces []namedInterface,
	opts Options,
) []taxonomy.Signal {
	var signals []taxonomy.Signal

	// 1. Interface satisfaction.
	if s := analyzeInterfaceSignal(funcName, receiverType, effectType, ifaces); s.Source != "" {
		signals = append(signals, s)
	}

	// 2. API surface visibility.
	if s := AnalyzeVisibilitySignal(funcDecl, funcObj, effectType); s.Source != "" {
		signals = append(signals, s)
	}

	// 3. Caller dependency.
	if s := AnalyzeCallerSignal(funcObj, effectType, opts.ModulePackages); s.Source != "" {
		signals = append(signals, s)
	}

	// 4. Naming convention (use namingName to handle sentinel vars).
	if s := AnalyzeNamingSignal(namingName, effectType); s.Source != "" {
		signals = append(signals, s)
	}

	// 5. Godoc comment.
	if s := AnalyzeGodocSignal(funcDecl, effectType); s.Source != "" {
		signals = append(signals, s)
	}

	return signals
}

// buildFuncDeclMap creates a lookup from function/method name to
// its AST declaration in the given package.
func buildFuncDeclMap(pkg *packages.Package) map[string]*ast.FuncDecl {
	m := make(map[string]*ast.FuncDecl)
	if pkg == nil {
		return m
	}
	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			m[fd.Name.Name] = fd
		}
	}
	return m
}

// buildFuncObjMap creates a lookup from function/method identifier
// to its types.Object in the given package.
//
// Package-level functions are keyed by their plain name (e.g. "Foo").
// Methods are keyed by "TypeName.MethodName" (e.g. "FileStore.Write")
// to avoid collisions when two types share a method name. A plain
// method name key is also stored as a fallback for callers that do
// not have receiver information, but it will be overwritten by the
// last method encountered with that name.
func buildFuncObjMap(pkg *packages.Package) map[string]types.Object {
	m := make(map[string]types.Object)
	if pkg == nil {
		return m
	}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if _, ok := obj.(*types.Func); ok {
			m[name] = obj
		}
	}

	// Also look up methods on named types.
	// Key by "TypeName.MethodName" to avoid collisions.
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		for i := 0; i < named.NumMethods(); i++ {
			method := named.Method(i)
			qualifiedKey := name + "." + method.Name()
			m[qualifiedKey] = method
			// Unqualified fallback — first method with this name wins.
			if _, exists := m[method.Name()]; !exists {
				m[method.Name()] = method
			}
		}
	}

	return m
}

// lookupFuncObj resolves a types.Object by preferring the
// receiver-qualified key "TypeName.MethodName" when receiver is
// non-empty, falling back to the plain function name.
func lookupFuncObj(m map[string]types.Object, receiver, funcName string) types.Object {
	if receiver != "" {
		// Strip leading * from pointer receivers (e.g. "*FileStore" → "FileStore").
		typeName := receiver
		if len(typeName) > 0 && typeName[0] == '*' {
			typeName = typeName[1:]
		}
		qualifiedKey := typeName + "." + funcName
		if obj, ok := m[qualifiedKey]; ok {
			return obj
		}
	}
	return m[funcName]
}
