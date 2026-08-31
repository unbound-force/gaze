package analysis_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/taxonomy"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// ---------------------------------------------------------------------------
// SC-001 / SC-002: panic recovery contract tests
//
// Note: BuildSSA's panic recovery cannot be tested end-to-end because
// prog.Build() is a concrete method on *ssa.Program that cannot be
// mocked or injected. The recovery pattern is verified through the
// ssaguard.SafeSSABuild tests in internal/ssaguard/ssaguard_test.go
// (which exercise the identical defer/recover logic). BuildSSA's logging
// behavior is verified by code inspection — the log.Warn/log.Debug calls
// are co-located with the ssaguard.SafeSSABuild call in the same if-block.
// ---------------------------------------------------------------------------

// TestSC001_BuildSSANoPanicReturnsPackage verifies that BuildSSA
// returns a non-nil *ssa.Package for a valid input package when no
// panic occurs (the normal path). This confirms the recover() guard
// is a no-op in the non-panic case (SC-001, FR-005).
func TestSC001_BuildSSANoPanicReturnsPackage(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	ssaPkg := analysis.BuildSSA(pkg)
	if ssaPkg == nil {
		t.Fatal("BuildSSA returned nil for a valid package — recover() guard may have interfered")
	}

	if _, ok := ssaPkg.Members["Normalize"]; !ok {
		t.Error("expected 'Normalize' in SSA members after BuildSSA")
	}
}

// ---------------------------------------------------------------------------
// AST mutation fallback tests (spec 036)
// ---------------------------------------------------------------------------

// TestAnalyzeASTMutations_PointerReceiverFieldAssignment verifies
// that the AST fallback detects ReceiverMutation effects when SSA
// is unavailable (ssaPkg = nil) for a pointer receiver method that
// assigns to a receiver field. Uses the Counter.Increment fixture
// (c.count++). SC-001: void methods with pointer receivers detect
// at least one ReceiverMutation.
func TestAnalyzeASTMutations_PointerReceiverFieldAssignment(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found in mutation package")
	}

	effects := analysis.AnalyzeMutations(pkg.Fset, nil, fd, toTypesFunc(pkg, fd), pkg.PkgPath, "Increment")
	if len(effects) == 0 {
		t.Fatal("expected at least one effect from AST fallback for (*Counter).Increment, got none")
	}

	foundReceiverMutation := false
	for _, e := range effects {
		if e.Type == taxonomy.ReceiverMutation {
			foundReceiverMutation = true
			if !strings.Contains(e.Description, "(AST fallback)") {
				t.Errorf("AST fallback effect missing '(AST fallback)' annotation in Description: %q", e.Description)
			}
			if e.Tier != taxonomy.TierP0 {
				t.Errorf("AST fallback ReceiverMutation tier = %q, want %q", e.Tier, taxonomy.TierP0)
			}
		}
	}
	if !foundReceiverMutation {
		t.Errorf("expected ReceiverMutation effect, got: %v", effects)
	}
}

// TestAnalyzeASTMutations_ValueReceiverNoEffect verifies that the
// AST fallback returns nil for a value receiver method (FR-004:
// value receiver mutations are not observable by the caller).
func TestAnalyzeASTMutations_ValueReceiverNoEffect(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	fd := analysis.FindMethodDecl(pkg, "Counter", "ValueReceiverTrap")
	if fd == nil {
		t.Fatal("(Counter).ValueReceiverTrap not found in mutation package")
	}

	effects := analysis.AnalyzeMutations(pkg.Fset, nil, fd, toTypesFunc(pkg, fd), pkg.PkgPath, "ValueReceiverTrap")
	if len(effects) != 0 {
		t.Errorf("expected nil/empty effects for value receiver method, got %d: %v", len(effects), effects)
	}
}

// TestAnalyzeASTMutations_PointerArgMutation verifies that the AST
// fallback detects PointerArgMutation effects for functions that
// assign to fields of pointer parameters. Uses the Normalize fixture
// (v[0] = 1.0 on *[3]float64).
func TestAnalyzeASTMutations_PointerArgMutation(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	fd := analysis.FindFuncDecl(pkg, "Normalize")
	if fd == nil {
		t.Fatal("Normalize not found in mutation package")
	}

	effects := analysis.AnalyzeMutations(pkg.Fset, nil, fd, toTypesFunc(pkg, fd), pkg.PkgPath, "Normalize")
	if len(effects) == 0 {
		t.Fatal("expected at least one effect from AST fallback for Normalize, got none")
	}

	foundPtrArgMutation := false
	for _, e := range effects {
		if e.Type == taxonomy.PointerArgMutation {
			foundPtrArgMutation = true
			if !strings.Contains(e.Description, "(AST fallback)") {
				t.Errorf("AST fallback effect missing '(AST fallback)' annotation: %q", e.Description)
			}
			if e.Tier != taxonomy.TierP0 {
				t.Errorf("AST fallback PointerArgMutation tier = %q, want %q", e.Tier, taxonomy.TierP0)
			}
		}
	}
	if !foundPtrArgMutation {
		t.Errorf("expected PointerArgMutation effect, got: %v", effects)
	}
}

// TestAnalyzeASTMutations_NoMutation verifies that the AST fallback
// returns nil for a function that takes a pointer argument but does
// not write through it (FR-009: no false positives for read-only
// access). Uses the ReadOnly fixture (return *v).
func TestAnalyzeASTMutations_NoMutation(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	fd := analysis.FindFuncDecl(pkg, "ReadOnly")
	if fd == nil {
		t.Fatal("ReadOnly not found in mutation package")
	}

	effects := analysis.AnalyzeMutations(pkg.Fset, nil, fd, toTypesFunc(pkg, fd), pkg.PkgPath, "ReadOnly")
	if len(effects) != 0 {
		t.Errorf("expected nil/empty effects for read-only pointer access, got %d: %v", len(effects), effects)
	}
}

// TestAnalyzeASTMutations_SSAPrecedence verifies that when SSA
// succeeds, SSA-based mutation detection takes precedence and the
// AST fallback is not used (FR-005). Effects from SSA should NOT
// contain "(AST fallback)" in their Description.
func TestAnalyzeASTMutations_SSAPrecedence(t *testing.T) {
	pkg, ssaPkg := loadTestPackageWithSSA(t, "mutation")
	if ssaPkg == nil {
		t.Skip("SSA construction failed for mutation package")
	}

	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found in mutation package")
	}

	effects := analysis.AnalyzeMutations(pkg.Fset, ssaPkg, fd, toTypesFunc(pkg, fd), pkg.PkgPath, "Increment")
	for _, e := range effects {
		if strings.Contains(e.Description, "(AST fallback)") {
			t.Errorf("SSA path should not produce AST fallback effects, got: %q", e.Description)
		}
	}
}

// TestExprRootIdent_AllBranches verifies that exprRootIdent correctly
// unwraps composite AST expressions to find the base identifier.
// Covers all branches: Ident, SelectorExpr, IndexExpr, StarExpr,
// ParenExpr, and the default nil return for unrecognized types.
func TestExprRootIdent_AllBranches(t *testing.T) {
	base := &ast.Ident{Name: "x"}

	tests := []struct {
		name     string
		expr     ast.Expr
		wantName string // empty means expect nil
	}{
		{"Ident", base, "x"},
		{"SelectorExpr", &ast.SelectorExpr{X: base, Sel: &ast.Ident{Name: "Field"}}, "x"},
		{"IndexExpr", &ast.IndexExpr{X: base, Index: &ast.BasicLit{}}, "x"},
		{"StarExpr", &ast.StarExpr{X: base}, "x"},
		{"ParenExpr", &ast.ParenExpr{X: base}, "x"},
		{"Nested_Selector_Index", &ast.SelectorExpr{
			X:   &ast.IndexExpr{X: base, Index: &ast.BasicLit{}},
			Sel: &ast.Ident{Name: "Field"},
		}, "x"},
		{"Nested_Star_Selector", &ast.StarExpr{
			X: &ast.SelectorExpr{X: base, Sel: &ast.Ident{Name: "Field"}},
		}, "x"},
		{"CallExpr_returns_nil", &ast.CallExpr{Fun: base}, ""},
		{"BinaryExpr_returns_nil", &ast.BinaryExpr{X: base, Y: base}, ""},
		{"Nil_input", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analysis.ExprRootIdent(tt.expr)
			if tt.wantName == "" {
				if got != nil {
					t.Errorf("ExprRootIdent(%s) = %q, want nil", tt.name, got.Name)
				}
			} else {
				if got == nil {
					t.Fatalf("ExprRootIdent(%s) = nil, want %q", tt.name, tt.wantName)
				}
				if got.Name != tt.wantName {
					t.Errorf("ExprRootIdent(%s) = %q, want %q", tt.name, got.Name, tt.wantName)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleReceiverAssignStmt tests (task 2.2)
// ---------------------------------------------------------------------------

// findNode walks a parsed file and returns the first node of type T.
func findNode[T ast.Node](file *ast.File) T {
	var result T
	ast.Inspect(file, func(n ast.Node) bool {
		if v, ok := n.(T); ok {
			result = v
			return false
		}
		return true
	})
	return result
}

// parseSource parses a Go source string and returns the file and file set.
func parseSource(t *testing.T, src string) (*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file, fset
}

func TestHandleReceiverAssignStmt_FieldAssign(t *testing.T) {
	src := `package p; func f() { recv.Field = 1 }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.AssignStmt](file)
	if node == nil {
		t.Fatal("no AssignStmt found")
	}

	found, pos := analysis.HandleReceiverAssignStmt(node, "recv")
	if !found {
		t.Error("expected mutation to be detected for recv.Field = 1")
	}
	if !pos.IsValid() {
		t.Error("expected valid position")
	}
}

func TestHandleReceiverAssignStmt_BareAssign(t *testing.T) {
	src := `package p; func f() { recv = v }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.AssignStmt](file)
	if node == nil {
		t.Fatal("no AssignStmt found")
	}

	found, _ := analysis.HandleReceiverAssignStmt(node, "recv")
	if found {
		t.Error("bare receiver assignment (recv = v) should not be detected as mutation")
	}
}

func TestHandleReceiverAssignStmt_Unrelated(t *testing.T) {
	src := `package p; func f() { x.Field = 1 }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.AssignStmt](file)
	if node == nil {
		t.Fatal("no AssignStmt found")
	}

	found, _ := analysis.HandleReceiverAssignStmt(node, "recv")
	if found {
		t.Error("assignment to unrelated variable x should not be detected as receiver mutation")
	}
}

// ---------------------------------------------------------------------------
// handleReceiverIncDecStmt tests (task 2.2)
// ---------------------------------------------------------------------------

func TestHandleReceiverIncDecStmt_FieldIncrement(t *testing.T) {
	src := `package p; func f() { recv.count++ }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.IncDecStmt](file)
	if node == nil {
		t.Fatal("no IncDecStmt found")
	}

	found, pos := analysis.HandleReceiverIncDecStmt(node, "recv")
	if !found {
		t.Error("expected mutation to be detected for recv.count++")
	}
	if !pos.IsValid() {
		t.Error("expected valid position")
	}
}

func TestHandleReceiverIncDecStmt_BareIncrement(t *testing.T) {
	src := `package p; func f() { recv++ }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.IncDecStmt](file)
	if node == nil {
		t.Fatal("no IncDecStmt found")
	}

	found, _ := analysis.HandleReceiverIncDecStmt(node, "recv")
	if found {
		t.Error("bare receiver increment (recv++) should not be detected as mutation")
	}
}

// ---------------------------------------------------------------------------
// handleReceiverCallExpr tests (task 2.2)
// ---------------------------------------------------------------------------

func TestHandleReceiverCallExpr_FieldMethod(t *testing.T) {
	src := `package p; func f() { recv.f.Delete(k) }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.CallExpr](file)
	if node == nil {
		t.Fatal("no CallExpr found")
	}

	found, pos := analysis.HandleReceiverCallExpr(node, "recv")
	if !found {
		t.Error("expected mutation to be detected for recv.f.Delete(k)")
	}
	if !pos.IsValid() {
		t.Error("expected valid position")
	}
}

func TestHandleReceiverCallExpr_DirectMethod(t *testing.T) {
	src := `package p; func f() { recv.Method() }`
	file, _ := parseSource(t, src)

	node := findNode[*ast.CallExpr](file)
	if node == nil {
		t.Fatal("no CallExpr found")
	}

	found, _ := analysis.HandleReceiverCallExpr(node, "recv")
	if found {
		t.Error("direct receiver method call (recv.Method()) should not be detected as field mutation")
	}
}

// toTypesFunc extracts the *types.Func for a FuncDecl from the
// package's type information. Returns nil if not found.
func toTypesFunc(pkg *packages.Package, fd *ast.FuncDecl) *types.Func {
	obj := pkg.TypesInfo.Defs[fd.Name]
	if obj == nil {
		return nil
	}
	fn, _ := obj.(*types.Func)
	return fn
}

// ---------------------------------------------------------------------------
// isPointerArgStore helpers
// ---------------------------------------------------------------------------

// extractStoresForFunc walks the SSA blocks of the named function
// and returns the function and all its Store instructions. For
// top-level functions, looks up via ssaPkg.Members. Fails the test
// if the function is not found.
func extractStoresForFunc(t *testing.T, ssaPkg *ssa.Package, funcName string) (*ssa.Function, []*ssa.Store) {
	t.Helper()

	member, ok := ssaPkg.Members[funcName]
	if !ok {
		t.Fatalf("SSA member %q not found in package", funcName)
	}
	fn, ok := member.(*ssa.Function)
	if !ok {
		t.Fatalf("SSA member %q is %T, want *ssa.Function", funcName, member)
	}

	var stores []*ssa.Store
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok {
				stores = append(stores, store)
			}
		}
	}
	return fn, stores
}

// extractStoresForMethod walks the SSA blocks of a method on the
// given named type and returns all Store instructions. Looks up the
// method via the pointer receiver's method set.
func extractStoresForMethod(t *testing.T, ssaPkg *ssa.Package, typeName, methodName string) (*ssa.Function, []*ssa.Store) {
	t.Helper()

	member, ok := ssaPkg.Members[typeName]
	if !ok {
		t.Fatalf("SSA member %q not found in package", typeName)
	}
	namedType, ok := member.(*ssa.Type)
	if !ok {
		t.Fatalf("SSA member %q is %T, want *ssa.Type", typeName, member)
	}

	// Look up via pointer receiver method set (covers both value
	// and pointer receiver methods).
	ptrType := types.NewPointer(namedType.Type())
	mset := types.NewMethodSet(ptrType)
	var fn *ssa.Function
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		if sel.Obj().Name() == methodName {
			fn = ssaPkg.Prog.MethodValue(sel)
			break
		}
	}
	if fn == nil {
		t.Fatalf("method %s on type %s not found in SSA", methodName, typeName)
	}

	var stores []*ssa.Store
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok {
				stores = append(stores, store)
			}
		}
	}
	return fn, stores
}

// buildPtrParams returns a map of parameter name → *ssa.Parameter
// for all pointer-typed parameters. When skipReceiver is true and
// the function is a method (has at least one param), the first
// parameter (receiver) is skipped.
func buildPtrParams(ssaFn *ssa.Function, skipReceiver bool) map[string]*ssa.Parameter {
	params := make(map[string]*ssa.Parameter)
	start := 0
	if skipReceiver && len(ssaFn.Params) > 0 {
		start = 1
	}
	for i := start; i < len(ssaFn.Params); i++ {
		p := ssaFn.Params[i]
		if _, ok := p.Type().(*types.Pointer); ok {
			params[p.Name()] = p
		}
	}
	return params
}

// ---------------------------------------------------------------------------
// isPointerArgStore tests
// ---------------------------------------------------------------------------

// TestIsPointerArgStore verifies that isPointerArgStore correctly
// identifies stores through pointer parameters across all branch
// patterns (FieldAddr, IndexAddr, UnOp dereference) and rejects
// stores that do not trace to a pointer parameter.
func TestIsPointerArgStore(t *testing.T) {
	_, ssaPkg := loadTestPackageWithSSA(t, "mutation")
	if ssaPkg == nil {
		t.Fatal("SSA construction failed for mutation package")
	}

	t.Run("FieldAddr", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "SetTimeout")
		ptrParams := buildPtrParams(fn, false)

		if _, ok := ptrParams["cfg"]; !ok {
			t.Fatal("expected 'cfg' in ptrParams for SetTimeout")
		}

		foundCfg := false
		for _, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched && name == "cfg" {
				foundCfg = true
			}
		}
		if !foundCfg {
			t.Error("expected at least one store to match pointer param 'cfg' via FieldAddr")
		}
	})

	t.Run("IndexAddr", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "Normalize")
		ptrParams := buildPtrParams(fn, false)

		if _, ok := ptrParams["v"]; !ok {
			t.Fatal("expected 'v' in ptrParams for Normalize")
		}

		foundV := false
		for _, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched && name == "v" {
				foundV = true
			}
		}
		if !foundV {
			t.Error("expected at least one store to match pointer param 'v' via IndexAddr")
		}
	})

	t.Run("UnOpDereference", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "FillSlice")
		ptrParams := buildPtrParams(fn, false)

		if _, ok := ptrParams["dst"]; !ok {
			t.Fatal("expected 'dst' in ptrParams for FillSlice")
		}

		foundDst := false
		for _, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched && name == "dst" {
				foundDst = true
			}
		}
		if !foundDst {
			t.Error("expected at least one store to match pointer param 'dst' via UnOp dereference")
		}
	})

	t.Run("DirectStore", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "SetDirect")
		ptrParams := buildPtrParams(fn, false)

		if _, ok := ptrParams["p"]; !ok {
			t.Fatal("expected 'p' in ptrParams for SetDirect")
		}

		foundP := false
		for _, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched && name == "p" {
				foundP = true
			}
		}
		if !foundP {
			t.Error("expected at least one store to match pointer param 'p' via direct store")
		}
	})

	t.Run("NestedFieldAddrChain", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "SetNestedValue")
		ptrParams := buildPtrParams(fn, false)

		if _, ok := ptrParams["cfg"]; !ok {
			t.Fatal("expected 'cfg' in ptrParams for SetNestedValue")
		}

		foundCfg := false
		for _, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched && name == "cfg" {
				foundCfg = true
			}
		}
		if !foundCfg {
			t.Error("expected at least one store to match pointer param 'cfg' via nested FieldAddr chain")
		}
	})

	t.Run("NoMatch_ReadOnly", func(t *testing.T) {
		fn, stores := extractStoresForFunc(t, ssaPkg, "ReadOnly")
		ptrParams := buildPtrParams(fn, false)

		for i, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched {
				t.Errorf("store[%d]: IsPointerArgStore returned (%q, true), want (\"\", false)", i, name)
			}
		}
	})

	t.Run("NoMatch_EmptyParams", func(t *testing.T) {
		_, stores := extractStoresForFunc(t, ssaPkg, "FillSlice")
		emptyParams := map[string]*ssa.Parameter{}

		for i, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, emptyParams)
			if matched {
				t.Errorf("store[%d]: IsPointerArgStore with empty params returned (%q, true), want (\"\", false)", i, name)
			}
		}
	})

	t.Run("NoMatch_ReceiverStore", func(t *testing.T) {
		fn, stores := extractStoresForMethod(t, ssaPkg, "Counter", "Increment")

		// Build ptrParams skipping the receiver — the receiver is
		// the first param in SSA for methods. Since Increment has
		// no other pointer params, ptrParams should be empty.
		ptrParams := buildPtrParams(fn, true)

		for i, store := range stores {
			name, matched := analysis.IsPointerArgStore(store, ptrParams)
			if matched {
				t.Errorf("store[%d]: IsPointerArgStore returned (%q, true) for receiver store, want (\"\", false)", i, name)
			}
		}
	})
}
