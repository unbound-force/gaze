package quality

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// AssertionKind enumerates the kinds of assertion patterns that
// can be detected in test functions.
type AssertionKind string

// Assertion kind constants.
const (
	// AssertionKindStdlibComparison is an if-based comparison
	// followed by t.Errorf/t.Fatalf (e.g., "if got != want").
	AssertionKindStdlibComparison AssertionKind = "stdlib_comparison"

	// AssertionKindStdlibErrorCheck is an error nil check
	// (e.g., "if err != nil { t.Fatal(err) }").
	AssertionKindStdlibErrorCheck AssertionKind = "stdlib_error_check"

	// AssertionKindTestifyEqual is a testify assert/require
	// equality call (e.g., assert.Equal(t, got, want)).
	AssertionKindTestifyEqual AssertionKind = "testify_equal"

	// AssertionKindTestifyNoError is a testify assert/require
	// NoError call (e.g., require.NoError(t, err)).
	AssertionKindTestifyNoError AssertionKind = "testify_noerror"

	// AssertionKindGoCmpDiff is a go-cmp diff check
	// (e.g., if diff := cmp.Diff(want, got); diff != "").
	AssertionKindGoCmpDiff AssertionKind = "gocmp_diff"

	// AssertionKindUnknown is an unrecognized assertion pattern.
	AssertionKindUnknown AssertionKind = "unknown"
)

// AssertionSite represents a detected assertion location in a test
// function or helper.
type AssertionSite struct {
	// Location is the source position as "file:line".
	Location string

	// Kind is the type of assertion pattern detected.
	Kind AssertionKind

	// FuncDecl is the containing function declaration.
	FuncDecl *ast.FuncDecl

	// Depth is the call depth from the test function body.
	// 0 = direct in the test body or t.Run closure.
	// 1-3 = inside helper functions at increasing depth.
	Depth int

	// Expr is the comparison or call expression that constitutes
	// the assertion.
	Expr ast.Expr
}

// DetectAssertions walks the test function's AST looking for
// assertion patterns. It detects stdlib comparisons, testify calls,
// go-cmp diffs, and recurses into helper functions up to maxDepth.
func DetectAssertions(
	testDecl *ast.FuncDecl,
	pkg *packages.Package,
	maxDepth int,
) []AssertionSite {
	d := &assertionDetector{
		pkg:      pkg,
		fset:     pkg.Fset,
		maxDepth: maxDepth,
		visited:  make(map[string]bool),
	}
	return d.detect(testDecl, 0)
}

// assertionDetector holds state for the assertion detection walk.
type assertionDetector struct {
	pkg      *packages.Package
	fset     *token.FileSet
	maxDepth int
	visited  map[string]bool // prevents infinite recursion
}

// detect walks a function declaration for assertion patterns.
func (d *assertionDetector) detect(fn *ast.FuncDecl, depth int) []AssertionSite {
	if fn == nil || fn.Body == nil {
		return nil
	}

	// Mark visited to prevent infinite recursion.
	key := fmt.Sprintf("%s:%d", fn.Name.Name, d.fset.Position(fn.Pos()).Line)
	if d.visited[key] {
		return nil
	}
	d.visited[key] = true

	var sites []AssertionSite

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			// Check for stdlib assertion patterns:
			// if got != want { t.Errorf(...) }
			// if err != nil { t.Fatal(err) }
			if site := d.detectStdlibAssertion(node, fn, depth); site != nil {
				sites = append(sites, *site)
			}

		case *ast.ExprStmt:
			// Check for testify/go-cmp assertion calls.
			if call, ok := node.X.(*ast.CallExpr); ok {
				if site := d.detectCallAssertion(call, fn, depth); site != nil {
					sites = append(sites, *site)
				}
			}

		case *ast.AssignStmt:
			// Check for go-cmp diff pattern:
			// diff := cmp.Diff(want, got)
			if site := d.detectGoCmpAssign(node, fn, depth); site != nil {
				sites = append(sites, *site)
			}

		case *ast.CallExpr:
			// Check for t.Run sub-tests.
			if d.isTRunCall(node) {
				subSites := d.detectTRunAssertions(node, fn)
				sites = append(sites, subSites...)
				return false // don't recurse into t.Run again
			}

			// Check for helper function calls (accepting *testing.T).
			if depth < d.maxDepth {
				helperSites := d.detectHelperAssertions(node, fn, depth)
				sites = append(sites, helperSites...)
			}
		}
		return true
	})

	return sites
}

// detectStdlibAssertion checks if an if-statement is an assertion
// pattern like "if got != want { t.Errorf(...) }" or
// "if err != nil { t.Fatal(err) }".
func (d *assertionDetector) detectStdlibAssertion(
	ifStmt *ast.IfStmt,
	fn *ast.FuncDecl,
	depth int,
) *AssertionSite {
	// Must have a binary comparison condition.
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	// The body must contain a t.Errorf, t.Fatalf, t.Error, or
	// t.Fatal call.
	if !d.bodyContainsTestFail(ifStmt.Body) {
		return nil
	}

	// Determine the assertion kind.
	kind := AssertionKindStdlibComparison
	if isErrorNilCheck(binExpr) {
		kind = AssertionKindStdlibErrorCheck
	}

	return &AssertionSite{
		Location: d.posString(ifStmt.Pos()),
		Kind:     kind,
		FuncDecl: fn,
		Depth:    depth,
		Expr:     binExpr,
	}
}

// isErrorNilCheck checks if a binary expression is "err != nil" or
// "err == nil" (or the reverse "nil != err").
func isErrorNilCheck(expr *ast.BinaryExpr) bool {
	// Check pattern: <ident containing "err"> != nil
	if ident, ok := expr.X.(*ast.Ident); ok {
		if nilIdent, ok := expr.Y.(*ast.Ident); ok && nilIdent.Name == "nil" {
			if strings.Contains(strings.ToLower(ident.Name), "err") {
				return true
			}
		}
	}
	// Check reverse: nil != <ident containing "err">
	if nilIdent, ok := expr.X.(*ast.Ident); ok && nilIdent.Name == "nil" {
		if ident, ok := expr.Y.(*ast.Ident); ok {
			if strings.Contains(strings.ToLower(ident.Name), "err") {
				return true
			}
		}
	}
	return false
}

// bodyContainsTestFail checks if a block contains t.Errorf,
// t.Fatalf, t.Error, t.Fatal, or similar calls.
func (d *assertionDetector) bodyContainsTestFail(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		if name == "Errorf" || name == "Fatalf" || name == "Error" ||
			name == "Fatal" || name == "FailNow" || name == "Fail" {
			found = true
		}
		return !found
	})
	return found
}

// detectCallAssertion checks if a function call is a testify
// assertion (assert.Equal, require.NoError, etc.).
func (d *assertionDetector) detectCallAssertion(
	call *ast.CallExpr,
	fn *ast.FuncDecl,
	depth int,
) *AssertionSite {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// Check for package-qualified calls: assert.Equal, require.Equal, etc.
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name
	pkgName := pkgIdent.Name

	// testify patterns.
	if pkgName == "assert" || pkgName == "require" {
		kind := d.classifyTestifyCall(methodName)
		if kind != AssertionKindUnknown {
			return &AssertionSite{
				Location: d.posString(call.Pos()),
				Kind:     kind,
				FuncDecl: fn,
				Depth:    depth,
				Expr:     call,
			}
		}
	}

	// go-cmp pattern: direct cmp.Diff call in an expression statement
	// is unusual, but detect it anyway.
	if pkgName == "cmp" && methodName == "Diff" {
		return &AssertionSite{
			Location: d.posString(call.Pos()),
			Kind:     AssertionKindGoCmpDiff,
			FuncDecl: fn,
			Depth:    depth,
			Expr:     call,
		}
	}

	return nil
}

// classifyTestifyCall maps testify method names to assertion kinds.
func (d *assertionDetector) classifyTestifyCall(method string) AssertionKind {
	switch method {
	case "Equal", "EqualValues", "EqualExportedValues",
		"Exactly", "JSONEq", "YAMLEq",
		"NotEqual", "NotEqualValues",
		"Contains", "NotContains",
		"ElementsMatch", "Subset", "NotSubset",
		"Len", "Empty", "NotEmpty",
		"True", "False",
		"Greater", "GreaterOrEqual", "Less", "LessOrEqual",
		"Regexp", "NotRegexp",
		"IsType", "Implements",
		"Same", "NotSame",
		"InDelta", "InDeltaSlice", "InEpsilon", "InEpsilonSlice",
		"Zero", "NotZero",
		"FileExists", "NoFileExists", "DirExists", "NoDirExists":
		return AssertionKindTestifyEqual

	case "NoError", "Error", "ErrorIs", "ErrorAs",
		"ErrorContains", "EqualError":
		return AssertionKindTestifyNoError

	case "Nil", "NotNil":
		return AssertionKindTestifyEqual

	case "Panics", "PanicsWithValue", "PanicsWithError", "NotPanics":
		return AssertionKindTestifyEqual
	}
	return AssertionKindUnknown
}

// detectGoCmpAssign checks for the go-cmp pattern:
// diff := cmp.Diff(want, got)
func (d *assertionDetector) detectGoCmpAssign(
	assign *ast.AssignStmt,
	fn *ast.FuncDecl,
	depth int,
) *AssertionSite {
	if len(assign.Rhs) != 1 {
		return nil
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}
	if pkgIdent.Name == "cmp" && sel.Sel.Name == "Diff" {
		return &AssertionSite{
			Location: d.posString(assign.Pos()),
			Kind:     AssertionKindGoCmpDiff,
			FuncDecl: fn,
			Depth:    depth,
			Expr:     call,
		}
	}
	return nil
}

// isTRunCall checks if a call expression is t.Run(name, func).
// It verifies the receiver is a conventional testing parameter
// name (t, tb, tt) to avoid matching unrelated .Run() calls.
func (d *assertionDetector) isTRunCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Run" || len(call.Args) != 2 {
		return false
	}
	// Verify the receiver is a testing parameter.
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "t" || ident.Name == "tb" || ident.Name == "tt"
}

// detectTRunAssertions extracts assertions from t.Run sub-test
// closures. Sub-test assertions are inlined at depth 0 because
// t.Run sub-tests are logically part of the parent test function.
func (d *assertionDetector) detectTRunAssertions(
	call *ast.CallExpr,
	fn *ast.FuncDecl,
) []AssertionSite {
	if len(call.Args) < 2 {
		return nil
	}

	// The second argument should be a function literal.
	funcLit, ok := call.Args[1].(*ast.FuncLit)
	if !ok {
		return nil
	}

	// Create a synthetic FuncDecl for the closure. Use the call
	// site position to make each t.Run closure's visited key unique,
	// preventing deduplication of multiple sub-tests in the same
	// parent function.
	syntheticDecl := &ast.FuncDecl{
		Name: &ast.Ident{
			Name:    fn.Name.Name + "$subtest",
			NamePos: call.Pos(),
		},
		Type: funcLit.Type,
		Body: funcLit.Body,
	}

	// Detect assertions in the sub-test at depth 0 (inlined).
	return d.detect(syntheticDecl, 0)
}

// detectHelperAssertions checks if a call is to a helper function
// (one that accepts *testing.T) and recurses into it.
func (d *assertionDetector) detectHelperAssertions(
	call *ast.CallExpr,
	_ *ast.FuncDecl,
	depth int,
) []AssertionSite {
	// Look up the called function in the package AST.
	funcName := d.extractFuncName(call)
	if funcName == "" {
		return nil
	}

	// Check if any argument looks like it's passing *testing.T.
	hasTestingT := false
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if ident.Name == "t" || ident.Name == "tb" || ident.Name == "tt" {
				hasTestingT = true
				break
			}
		}
	}
	if !hasTestingT {
		return nil
	}

	// Find the function declaration in the package.
	helperDecl := d.findFuncDecl(funcName)
	if helperDecl == nil {
		return nil
	}

	// Verify it accepts *testing.T (or *testing.TB).
	if !acceptsTestingParam(helperDecl) {
		return nil
	}

	return d.detect(helperDecl, depth+1)
}

// extractFuncName extracts the function name from a call expression.
func (d *assertionDetector) extractFuncName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		// Could be a method call or package-qualified call.
		// For helpers, we only care about unqualified calls
		// within the same package.
		return ""
	}
	return ""
}

// findFuncDecl looks up a function declaration by name in the
// package's syntax trees.
func (d *assertionDetector) findFuncDecl(name string) *ast.FuncDecl {
	for _, file := range d.pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}

// acceptsTestingParam checks if a function accepts *testing.T or
// *testing.TB as a parameter.
func acceptsTestingParam(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, param := range fn.Type.Params.List {
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "testing" &&
			(sel.Sel.Name == "T" || sel.Sel.Name == "TB" || sel.Sel.Name == "B") {
			return true
		}
	}
	return false
}

// posString formats a token.Pos as "file:line".
func (d *assertionDetector) posString(pos token.Pos) string {
	p := d.fset.Position(pos)
	return fmt.Sprintf("%s:%d", p.Filename, p.Line)
}
