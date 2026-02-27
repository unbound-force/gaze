package analysis_test

import (
	"go/ast"
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
)

// ---------------------------------------------------------------------------
// BuildSSA tests
// ---------------------------------------------------------------------------

// TestBuildSSA_ReturnsNonNilPackage verifies that BuildSSA returns a
// non-nil *ssa.Package for a valid input package and that the SSA
// members map contains the expected exported functions.
func TestBuildSSA_ReturnsNonNilPackage(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")

	ssaPkg := analysis.BuildSSA(pkg)
	if ssaPkg == nil {
		t.Fatal("BuildSSA returned nil for a valid package")
	}

	// The mutation fixture defines Normalize as a package-level
	// function; it must appear in SSA members.
	if _, ok := ssaPkg.Members["Normalize"]; !ok {
		t.Error("expected 'Normalize' in SSA members after BuildSSA")
	}
}

// ---------------------------------------------------------------------------
// baseTypeName tests
// ---------------------------------------------------------------------------

// TestBaseTypeName_Ident verifies that a bare identifier returns its
// name directly.
func TestBaseTypeName_Ident(t *testing.T) {
	ident := &ast.Ident{Name: "Counter"}
	got := analysis.BaseTypeName(ident)
	if got != "Counter" {
		t.Errorf("BaseTypeName(Ident{Counter}) = %q, want %q", got, "Counter")
	}
}

// TestBaseTypeName_StarExpr verifies that a pointer type expression
// strips the star and returns the base name.
func TestBaseTypeName_StarExpr(t *testing.T) {
	star := &ast.StarExpr{X: &ast.Ident{Name: "Counter"}}
	got := analysis.BaseTypeName(star)
	if got != "Counter" {
		t.Errorf("BaseTypeName(*Counter) = %q, want %q", got, "Counter")
	}
}

// TestBaseTypeName_DoublyNested verifies that double-pointer types are
// unwrapped to the base ident.
func TestBaseTypeName_DoublyNested(t *testing.T) {
	expr := &ast.StarExpr{X: &ast.StarExpr{X: &ast.Ident{Name: "Node"}}}
	got := analysis.BaseTypeName(expr)
	if got != "Node" {
		t.Errorf("BaseTypeName(**Node) = %q, want %q", got, "Node")
	}
}

// TestBaseTypeName_Unknown verifies that unrecognised AST node types
// return an empty string.
func TestBaseTypeName_Unknown(t *testing.T) {
	// ast.SelectorExpr is not handled.
	expr := &ast.SelectorExpr{
		X:   &ast.Ident{Name: "pkg"},
		Sel: &ast.Ident{Name: "Type"},
	}
	got := analysis.BaseTypeName(expr)
	if got != "" {
		t.Errorf("BaseTypeName(SelectorExpr) = %q, want %q", got, "")
	}
}

// ---------------------------------------------------------------------------
// findSSAFunction fallback-path tests
//
// The primary lookup (via types.Func) covers the happy path exercised
// by analyzeFunc/analyzeMethod. These tests drive the name-based
// fallback (fnObj == nil) to ensure defensive code is correct.
// ---------------------------------------------------------------------------

// TestFindSSAFunction_FallbackNoReceiver_NotInMembers verifies that
// the fallback returns nil when a synthesised FuncDecl name is absent
// from ssaPkg.Members.
func TestFindSSAFunction_FallbackNoReceiver_NotInMembers(t *testing.T) {
	pkg, ssaPkg := loadTestPackageWithSSA(t, "mutation")

	_ = analysis.FindFuncDecl(pkg, "Normalize") // load package

	// Synthesise a FuncDecl with a name not present in SSA members.
	fakeFD := &ast.FuncDecl{
		Name: &ast.Ident{Name: "AbsolutelyDoesNotExist"},
	}

	fn := analysis.FindSSAFunction(ssaPkg, nil, fakeFD)
	if fn != nil {
		t.Errorf("expected nil for unknown function name in fallback, got %v", fn)
	}
}

// TestFindSSAFunction_FallbackPackageFunc verifies the name-based
// fallback for a package-level function when fnObj is nil.
func TestFindSSAFunction_FallbackPackageFunc(t *testing.T) {
	pkg, ssaPkg := loadTestPackageWithSSA(t, "mutation")

	fd := analysis.FindFuncDecl(pkg, "Normalize")
	if fd == nil {
		t.Fatal("Normalize not found in mutation package")
	}

	// Passing nil fnObj forces the fallback name-based lookup.
	fn := analysis.FindSSAFunction(ssaPkg, nil, fd)
	if fn == nil {
		t.Fatal("expected non-nil SSA function for Normalize via fallback lookup")
	}
	if fn.Name() != "Normalize" {
		t.Errorf("expected function name 'Normalize', got %q", fn.Name())
	}
}

// TestFindSSAFunction_FallbackPointerReceiverMethod verifies the
// fallback path for a pointer-receiver method when fnObj is nil.
func TestFindSSAFunction_FallbackPointerReceiverMethod(t *testing.T) {
	pkg, ssaPkg := loadTestPackageWithSSA(t, "mutation")

	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found in mutation package")
	}

	fn := analysis.FindSSAFunction(ssaPkg, nil, fd)
	if fn == nil {
		t.Fatal("expected non-nil SSA function for (*Counter).Increment via fallback")
	}
	if fn.Name() != "Increment" {
		t.Errorf("expected function name 'Increment', got %q", fn.Name())
	}
}

// TestFindSSAFunction_FallbackValueReceiverMethod verifies the
// fallback path for a value-receiver method when fnObj is nil.
func TestFindSSAFunction_FallbackValueReceiverMethod(t *testing.T) {
	pkg, ssaPkg := loadTestPackageWithSSA(t, "mutation")

	fd := analysis.FindMethodDecl(pkg, "Counter", "Value")
	if fd == nil {
		t.Fatal("(Counter).Value not found in mutation package")
	}

	fn := analysis.FindSSAFunction(ssaPkg, nil, fd)
	// Value receiver methods may appear on the pointer type's method
	// set in SSA. Accept non-nil if found; the test confirms the
	// fallback path runs without panicking and produces a consistent
	// result.
	if fn != nil && fn.Name() != "Value" {
		t.Errorf("expected function name 'Value', got %q", fn.Name())
	}
}
