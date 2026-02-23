package quality

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// TestFunc represents a test function found in a test package.
type TestFunc struct {
	// Name is the function name (e.g., "TestFoo").
	Name string

	// Decl is the AST declaration of the test function.
	Decl *ast.FuncDecl

	// Location is the source position as "file:line".
	Location string
}

// InferredTarget represents a function identified as the target
// under test by call graph inference.
type InferredTarget struct {
	// FuncName is the qualified function name matching
	// FunctionTarget.QualifiedName() format.
	FuncName string

	// SSAFunc is the SSA representation of the target function.
	SSAFunc *ssa.Function
}

// FindTestFunctions scans the loaded test package for functions
// matching the Test*(*testing.T) signature. It returns a list of
// TestFunc values for each test function found.
func FindTestFunctions(pkg *packages.Package) []TestFunc {
	var tests []TestFunc

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !isTestFunction(fn) {
				continue
			}
			pos := pkg.Fset.Position(fn.Pos())
			tests = append(tests, TestFunc{
				Name:     fn.Name.Name,
				Decl:     fn,
				Location: fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
			})
		}
	}
	return tests
}

// isTestFunction checks whether the function declaration has the
// Test* name prefix and accepts a single *testing.T parameter.
func isTestFunction(fn *ast.FuncDecl) bool {
	if fn.Recv != nil {
		return false // methods are not test functions
	}
	if !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	// Must have exactly one parameter of type *testing.T.
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	param := fn.Type.Params.List[0]
	return isTestingTParam(param)
}

// isTestingTParam checks whether a field is of type *testing.T.
func isTestingTParam(field *ast.Field) bool {
	star, ok := field.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "testing" && sel.Sel.Name == "T"
}

// BuildTestSSA builds the SSA representation for a test package.
// It returns both the SSA program (needed for cross-package call
// graph analysis) and the SSA package for the test code.
func BuildTestSSA(pkg *packages.Package) (*ssa.Program, *ssa.Package, error) {
	prog, ssaPkgs := ssautil.AllPackages(
		[]*packages.Package{pkg},
		ssa.InstantiateGenerics,
	)
	prog.Build()
	if len(ssaPkgs) == 0 || ssaPkgs[0] == nil {
		return nil, nil, fmt.Errorf("failed to build SSA for package %s", pkg.PkgPath)
	}
	return prog, ssaPkgs[0], nil
}

// InferTargets identifies which non-test functions the given test
// function exercises, using SSA call graph analysis bounded to
// opts.MaxHelperDepth levels.
//
// It returns the inferred targets and any warnings (e.g., ambiguous
// targets, no target found).
func InferTargets(
	testFunc *ssa.Function,
	testPkg *packages.Package,
	opts Options,
) ([]InferredTarget, []string) {
	if testFunc.Blocks == nil {
		return nil, []string{"test function has no SSA body"}
	}

	testPkgPath := testPkg.PkgPath
	// External test packages have path suffix "_test"; the target
	// package is the base path without the suffix.
	targetPkgPath := strings.TrimSuffix(testPkgPath, "_test")

	candidates := make(map[string]*ssa.Function)
	var warnings []string

	// Walk the SSA blocks looking for calls to functions in the
	// target package (non-test, non-stdlib).
	walkCalls(testFunc, targetPkgPath, candidates, opts.MaxHelperDepth, make(map[*ssa.Function]bool))

	if len(candidates) == 0 {
		return nil, []string{"no target function identified"}
	}

	targets := make([]InferredTarget, 0, len(candidates))
	for name, fn := range candidates {
		targets = append(targets, InferredTarget{
			FuncName: name,
			SSAFunc:  fn,
		})
	}

	// Sort targets deterministically by name to ensure stable
	// ordering across runs (SC-004 determinism requirement).
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].FuncName < targets[j].FuncName
	})

	if len(targets) > 1 {
		names := make([]string, 0, len(targets))
		for _, t := range targets {
			names = append(names, t.FuncName)
		}
		warnings = append(warnings, fmt.Sprintf(
			"multiple target functions detected: %s", strings.Join(names, ", ")))
	}

	return targets, warnings
}

// walkCalls recursively walks SSA call instructions to find calls
// to functions in the target package. It uses a visited set to
// prevent infinite recursion and respects the maxDepth bound.
func walkCalls(
	fn *ssa.Function,
	targetPkgPath string,
	candidates map[string]*ssa.Function,
	maxDepth int,
	visited map[*ssa.Function]bool,
) {
	if maxDepth <= 0 || fn == nil || fn.Blocks == nil {
		return
	}
	if visited[fn] {
		return
	}
	visited[fn] = true

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Handle MakeClosure instructions: walk into anonymous
			// functions (closures) passed to t.Run or helpers. This
			// is critical for table-driven tests where the target
			// call is inside the t.Run closure.
			if mc, ok := instr.(*ssa.MakeClosure); ok {
				if closureFn, ok := mc.Fn.(*ssa.Function); ok {
					walkCalls(closureFn, targetPkgPath, candidates, maxDepth, visited)
				}
				continue
			}

			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			callee := resolveCallee(call)
			if callee == nil {
				continue
			}

			// Check if the callee belongs to the target package.
			if isTargetFunction(callee, targetPkgPath) {
				name := qualifiedSSAName(callee)
				candidates[name] = callee
				continue
			}

			// Recurse into non-stdlib, non-test callees (helpers).
			if shouldRecurse(callee, targetPkgPath) {
				walkCalls(callee, targetPkgPath, candidates, maxDepth-1, visited)
			}
		}
	}
}

// resolveCallee extracts the concrete *ssa.Function from a call
// instruction, handling both static and invoke calls.
func resolveCallee(call *ssa.Call) *ssa.Function {
	if call.Call.IsInvoke() {
		return nil // interface calls cannot be statically resolved
	}
	return call.Call.StaticCallee()
}

// isTargetFunction checks whether the callee belongs to the target
// package and is not a test function or stdlib function.
func isTargetFunction(callee *ssa.Function, targetPkgPath string) bool {
	pkg := callee.Package()
	if pkg == nil {
		return false
	}
	pkgPath := pkg.Pkg.Path()

	// Must be in the target package (not the test variant).
	if pkgPath != targetPkgPath {
		return false
	}

	// Skip test functions and init functions.
	name := callee.Name()
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || name == "init" {
		return false
	}

	// Skip unexported anonymous functions (closures).
	if strings.Contains(name, "$") {
		return false
	}

	return true
}

// shouldRecurse checks if we should follow calls into this function
// to find deeper target calls. We recurse into functions in the
// same module that are not stdlib.
func shouldRecurse(callee *ssa.Function, _ string) bool {
	pkg := callee.Package()
	if pkg == nil {
		return false
	}
	pkgPath := pkg.Pkg.Path()
	// Don't recurse into stdlib.
	if !strings.Contains(pkgPath, ".") {
		return false
	}
	// Don't recurse into test functions.
	if strings.HasPrefix(callee.Name(), "Test") {
		return false
	}
	return callee.Blocks != nil
}

// HasTestSyntax checks if a package's syntax trees contain test
// function declarations (func Test*(*testing.T)). This is used by
// both the CLI layer and tests to select the correct package variant
// when loading with Tests=true.
func HasTestSyntax(pkg *packages.Package) bool {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if strings.HasPrefix(fn.Name.Name, "Test") {
				return true
			}
		}
	}
	return false
}

// qualifiedSSAName returns a qualified name for an SSA function
// matching the FunctionTarget.QualifiedName() format.
//
// For methods, the analysis module formats receiver types as
// "(*Type).Method" for pointer receivers and "Type.Method" for
// value receivers. SSA may present the receiver differently (e.g.,
// as a value parameter even for pointer-receiver methods). To match,
// we check whether the receiver type is a pointer using go/types.
func qualifiedSSAName(fn *ssa.Function) string {
	if fn.Signature.Recv() == nil {
		return fn.Name()
	}

	recvType := fn.Signature.Recv().Type()
	typeName := ""
	isPtr := false

	// Unwrap the pointer if present.
	if ptr, ok := recvType.(*types.Pointer); ok {
		isPtr = true
		recvType = ptr.Elem()
	}

	// Get the type name (strip package path).
	typeStr := recvType.String()
	if idx := strings.LastIndex(typeStr, "."); idx >= 0 {
		typeName = typeStr[idx+1:]
	} else {
		typeName = typeStr
	}

	if isPtr {
		return fmt.Sprintf("(*%s).%s", typeName, fn.Name())
	}
	return fmt.Sprintf("%s.%s", typeName, fn.Name())
}
