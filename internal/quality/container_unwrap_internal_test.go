package quality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// parseAndTypeCheck parses Go source code and type-checks it,
// returning the AST file and populated types.Info. Used by unit
// tests for the container unwrap helper functions.
func parseAndTypeCheck(t *testing.T, src string) (*ast.File, *types.Info) {
	t.Helper()
	file, info, _ := parseAndTypeCheckWithFset(t, src)
	return file, info
}

// extractCallFromFunc extracts a CallExpr from the specified function
// declaration and statement index. The funcIdx is the index into
// file.Decls, and stmtIdx is the index into the function's body
// statement list. Handles both ExprStmt (bare calls) and AssignStmt
// (calls as RHS).
func extractCallFromFunc(t *testing.T, file *ast.File, funcIdx, stmtIdx int) *ast.CallExpr {
	t.Helper()
	fn, ok := file.Decls[funcIdx].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[%d] is not a FuncDecl", funcIdx)
	}
	if stmtIdx >= len(fn.Body.List) {
		t.Fatalf("stmt index %d out of range (body has %d stmts)", stmtIdx, len(fn.Body.List))
	}
	stmt := fn.Body.List[stmtIdx]
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		if !ok {
			t.Fatalf("ExprStmt.X is not a CallExpr")
		}
		return call
	case *ast.AssignStmt:
		if len(s.Rhs) == 0 {
			t.Fatalf("AssignStmt has no RHS")
		}
		call, ok := s.Rhs[0].(*ast.CallExpr)
		if !ok {
			t.Fatalf("AssignStmt.Rhs[0] is not a CallExpr")
		}
		return call
	default:
		t.Fatalf("stmt is %T, expected ExprStmt or AssignStmt", stmt)
		return nil
	}
}

// TestIsDataExtraction_SelectorExpr verifies that field access
// expressions (x.Field) are classified as data extraction.
func TestIsDataExtraction_SelectorExpr(t *testing.T) {
	expr := &ast.SelectorExpr{
		X:   &ast.Ident{Name: "x"},
		Sel: &ast.Ident{Name: "Field"},
	}
	if !isDataExtraction(expr) {
		t.Error("SelectorExpr should be classified as data extraction")
	}
}

// TestIsDataExtraction_IndexExpr verifies that index access
// expressions (x[i]) are classified as data extraction.
func TestIsDataExtraction_IndexExpr(t *testing.T) {
	expr := &ast.IndexExpr{
		X:     &ast.Ident{Name: "x"},
		Index: &ast.BasicLit{Kind: token.INT, Value: "0"},
	}
	if !isDataExtraction(expr) {
		t.Error("IndexExpr should be classified as data extraction")
	}
}

// TestIsDataExtraction_TypeAssertExpr verifies that type assertion
// expressions (x.(T)) are classified as data extraction.
func TestIsDataExtraction_TypeAssertExpr(t *testing.T) {
	expr := &ast.TypeAssertExpr{
		X:    &ast.Ident{Name: "x"},
		Type: &ast.Ident{Name: "int"},
	}
	if !isDataExtraction(expr) {
		t.Error("TypeAssertExpr should be classified as data extraction")
	}
}

// TestIsDataExtraction_TypeConversion verifies that type conversion
// expressions with *ast.Ident as Fun (e.g., string(x)) are
// classified as data extraction.
func TestIsDataExtraction_TypeConversion(t *testing.T) {
	expr := &ast.CallExpr{
		Fun:  &ast.Ident{Name: "string"},
		Args: []ast.Expr{&ast.Ident{Name: "x"}},
	}
	if !isDataExtraction(expr) {
		t.Error("single-arg CallExpr with Ident Fun should be classified as data extraction (type conversion)")
	}
}

// TestIsDataExtraction_ArrayTypeConversion verifies that slice type
// conversion expressions ([]byte(x)) are classified as data extraction.
func TestIsDataExtraction_ArrayTypeConversion(t *testing.T) {
	expr := &ast.CallExpr{
		Fun: &ast.ArrayType{
			Elt: &ast.Ident{Name: "byte"},
		},
		Args: []ast.Expr{&ast.Ident{Name: "x"}},
	}
	if !isDataExtraction(expr) {
		t.Error("single-arg CallExpr with ArrayType Fun should be classified as data extraction ([]byte conversion)")
	}
}

// TestIsDataExtraction_ParenExpr verifies that parenthesized
// expressions recurse into the inner expression.
func TestIsDataExtraction_ParenExpr(t *testing.T) {
	inner := &ast.SelectorExpr{
		X:   &ast.Ident{Name: "x"},
		Sel: &ast.Ident{Name: "Field"},
	}
	expr := &ast.ParenExpr{X: inner}
	if !isDataExtraction(expr) {
		t.Error("ParenExpr wrapping SelectorExpr should be classified as data extraction")
	}
}

// TestIsDataExtraction_MethodCall verifies that method calls
// (x.Method()) are NOT classified as data extraction.
func TestIsDataExtraction_MethodCall(t *testing.T) {
	expr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "s"},
			Sel: &ast.Ident{Name: "Get"},
		},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"key"`}},
	}
	if isDataExtraction(expr) {
		t.Error("method call (SelectorExpr as Fun) should NOT be classified as data extraction")
	}
}

// TestIsDataExtraction_MultiArgCall verifies that multi-argument
// function calls are NOT classified as data extraction.
func TestIsDataExtraction_MultiArgCall(t *testing.T) {
	expr := &ast.CallExpr{
		Fun: &ast.Ident{Name: "process"},
		Args: []ast.Expr{
			&ast.Ident{Name: "x"},
			&ast.Ident{Name: "y"},
		},
	}
	if isDataExtraction(expr) {
		t.Error("multi-arg CallExpr should NOT be classified as data extraction")
	}
}

// TestIsDataExtraction_BareIdent verifies that a bare identifier
// is NOT classified as data extraction.
func TestIsDataExtraction_BareIdent(t *testing.T) {
	expr := &ast.Ident{Name: "x"}
	if isDataExtraction(expr) {
		t.Error("bare Ident should NOT be classified as data extraction")
	}
}

// extractVarObj extracts the types.Object for a variable defined in
// the given assignment statement's LHS at position 0.
func extractVarObj(t *testing.T, file *ast.File, info *types.Info, funcIdx, stmtIdx int) types.Object {
	t.Helper()
	fn, ok := file.Decls[funcIdx].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[%d] is not a FuncDecl", funcIdx)
	}
	assign, ok := fn.Body.List[stmtIdx].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("stmt[%d] is not AssignStmt", stmtIdx)
	}
	ident, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		t.Fatal("LHS[0] is not Ident")
	}
	obj := info.Defs[ident]
	if obj == nil {
		t.Fatalf("could not find types.Object for %s", ident.Name)
	}
	return obj
}

// extractRHS extracts the RHS expression from the given statement.
func extractRHS(t *testing.T, file *ast.File, funcIdx, stmtIdx int) ast.Expr {
	t.Helper()
	fn, ok := file.Decls[funcIdx].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[%d] is not a FuncDecl", funcIdx)
	}
	assign, ok := fn.Body.List[stmtIdx].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("stmt[%d] is not AssignStmt", stmtIdx)
	}
	return assign.Rhs[0]
}

// TestContainsObject_Found verifies that containsObject returns true
// when the target object is referenced in the expression.
func TestContainsObject_Found(t *testing.T) {
	file, info := parseAndTypeCheck(t, `package p; func f() { x := 1; _ = x + 2 }`)

	xObj := extractVarObj(t, file, info, 0, 0)
	rhs := extractRHS(t, file, 0, 1)

	if !containsObject(rhs, xObj, info) {
		t.Error("containsObject should find x in the expression x + 2")
	}
}

// TestContainsObject_NotFound verifies that containsObject returns
// false when the target object is not in the expression.
func TestContainsObject_NotFound(t *testing.T) {
	file, info := parseAndTypeCheck(t, `package p; func f() { x := 1; y := 2; _ = y + 3; _ = x }`)

	xObj := extractVarObj(t, file, info, 0, 0)
	rhs := extractRHS(t, file, 0, 2) // _ = y + 3

	if containsObject(rhs, xObj, info) {
		t.Error("containsObject should not find x in the expression y + 3")
	}
}

// TestContainsObject_NilInputs verifies that containsObject handles
// nil inputs gracefully.
func TestContainsObject_NilInputs(t *testing.T) {
	if containsObject(nil, nil, nil) {
		t.Error("containsObject(nil, nil, nil) should return false")
	}

	expr := &ast.Ident{Name: "x"}
	if containsObject(expr, nil, nil) {
		t.Error("containsObject with nil target should return false")
	}
}

// TestExtractPointerDest_AddressOf verifies that extractPointerDest
// correctly unwraps &data to find the underlying variable.
func TestExtractPointerDest_AddressOf(t *testing.T) {
	src := `package p
func g(x []byte, dst *int) {}
func f() {
	var data int
	g(nil, &data)
}`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 1) // f(), stmt 1: g(nil, &data)

	dest := extractPointerDest(call, 1, info)
	if dest == nil {
		t.Fatal("extractPointerDest should find the variable from &data")
	}
	if dest.Name() != "data" {
		t.Errorf("expected variable name 'data', got %q", dest.Name())
	}
}

// TestExtractPointerDest_NilInputs verifies that extractPointerDest
// handles nil and out-of-bounds inputs gracefully.
func TestExtractPointerDest_NilInputs(t *testing.T) {
	if extractPointerDest(nil, 0, nil) != nil {
		t.Error("extractPointerDest(nil, 0, nil) should return nil")
	}

	call := &ast.CallExpr{Args: []ast.Expr{&ast.Ident{Name: "x"}}}
	if extractPointerDest(call, 5, nil) != nil {
		t.Error("extractPointerDest with out-of-bounds index should return nil")
	}
}

// TestExtractPointerDest_BareIdent verifies that extractPointerDest
// handles a bare identifier argument (already a pointer variable).
func TestExtractPointerDest_BareIdent(t *testing.T) {
	src := `package p
func f() {
	var p *int
	g(p)
}
func g(x *int) {}`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 0, 1) // f(), stmt 1: g(p)

	dest := extractPointerDest(call, 0, info)
	if dest == nil {
		t.Fatal("extractPointerDest should find the variable from bare pointer ident")
	}
	if dest.Name() != "p" {
		t.Errorf("expected variable name 'p', got %q", dest.Name())
	}
}

// TestIsTransformationCall_ByteSliceAndPointer verifies that a
// function with []byte and *T parameters is detected as a
// transformation call.
func TestIsTransformationCall_ByteSliceAndPointer(t *testing.T) {
	src := `package p
func unmarshal(data []byte, dst *int) {}
func f() { var x int; unmarshal(nil, &x) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 1) // f(), stmt 1: unmarshal(nil, &x)

	byteIdx, ptrIdx, ok := isTransformationCall(call, info)
	if !ok {
		t.Fatal("isTransformationCall should match func([]byte, *int)")
	}
	if byteIdx != 0 {
		t.Errorf("byteArgIdx = %d, want 0", byteIdx)
	}
	if ptrIdx != 1 {
		t.Errorf("ptrDestIdx = %d, want 1", ptrIdx)
	}
}

// TestIsTransformationCall_StringAndPointer verifies that a function
// with string and *T parameters is detected as a transformation call.
func TestIsTransformationCall_StringAndPointer(t *testing.T) {
	src := `package p
func decode(s string, dst *int) {}
func f() { var x int; decode("", &x) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 1) // f(), stmt 1: decode("", &x)

	byteIdx, ptrIdx, ok := isTransformationCall(call, info)
	if !ok {
		t.Fatal("isTransformationCall should match func(string, *int)")
	}
	if byteIdx != 0 {
		t.Errorf("byteArgIdx = %d, want 0", byteIdx)
	}
	if ptrIdx != 1 {
		t.Errorf("ptrDestIdx = %d, want 1", ptrIdx)
	}
}

// TestIsTransformationCall_NoPointer verifies that a function without
// a pointer parameter is not detected as a transformation call.
func TestIsTransformationCall_NoPointer(t *testing.T) {
	src := `package p
func process(data []byte, n int) {}
func f() { process(nil, 0) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 0) // f(), stmt 0: process(nil, 0)

	_, _, ok := isTransformationCall(call, info)
	if ok {
		t.Error("isTransformationCall should not match func([]byte, int) — no pointer dest")
	}
}

// TestIsTransformationCall_NoByteLike verifies that a function without
// a byte-like parameter is not detected as a transformation call.
func TestIsTransformationCall_NoByteLike(t *testing.T) {
	src := `package p
func store(n int, dst *int) {}
func f() { var x int; store(0, &x) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 1) // f(), stmt 1: store(0, &x)

	_, _, ok := isTransformationCall(call, info)
	if ok {
		t.Error("isTransformationCall should not match func(int, *int) — no byte-like input")
	}
}

// TestIsTransformationCall_NilInputs verifies that isTransformationCall
// handles nil inputs gracefully.
func TestIsTransformationCall_NilInputs(t *testing.T) {
	_, _, ok := isTransformationCall(nil, nil)
	if ok {
		t.Error("isTransformationCall(nil, nil) should return ok=false")
	}
}

// TestIsTransformationCall_IoReaderAndPointer verifies that a function
// with an interface parameter that has a Read method (io.Reader-like)
// and a pointer parameter is detected as a transformation call.
func TestIsTransformationCall_IoReaderAndPointer(t *testing.T) {
	// Define a local Reader interface with a Read method to avoid
	// depending on the io package importer (parseAndTypeCheck uses
	// Importer: nil).
	src := `package p

type Reader interface { Read(p []byte) (n int, err error) }
func decode(r Reader, dst *int) {}
func f() { var x int; decode(nil, &x) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 2, 1) // f(), stmt 1: decode(nil, &x)

	byteIdx, ptrIdx, ok := isTransformationCall(call, info)
	if !ok {
		t.Fatal("isTransformationCall should match func(io.Reader, *int)")
	}
	if byteIdx != 0 {
		t.Errorf("byteArgIdx = %d, want 0", byteIdx)
	}
	if ptrIdx != 1 {
		t.Errorf("ptrDestIdx = %d, want 1", ptrIdx)
	}
}

// TestIsTransformationCall_EmptyInterfaceAsPointerDest verifies that a
// function with []byte and interface{} parameters is detected as a
// transformation call, where the empty interface serves as the pointer
// destination (e.g., json.Unmarshal).
func TestIsTransformationCall_EmptyInterfaceAsPointerDest(t *testing.T) {
	src := `package p

func unmarshal(data []byte, v interface{}) {}
func f() { unmarshal(nil, nil) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 0) // f(), stmt 0: unmarshal(nil, nil)

	byteIdx, ptrIdx, ok := isTransformationCall(call, info)
	if !ok {
		t.Fatal("isTransformationCall should match func([]byte, interface{})")
	}
	if byteIdx != 0 {
		t.Errorf("byteArgIdx = %d, want 0", byteIdx)
	}
	if ptrIdx != 1 {
		t.Errorf("ptrDestIdx = %d, want 1", ptrIdx)
	}
}

// TestIsTransformationCall_PointerBeforeByteSlice verifies that a
// function with *T before []byte is correctly detected — parameter
// ordering does not affect detection, and indices are correct.
func TestIsTransformationCall_PointerBeforeByteSlice(t *testing.T) {
	src := `package p

func decode(dst *int, data []byte) {}
func f() { var x int; decode(&x, nil) }`
	file, info := parseAndTypeCheck(t, src)
	call := extractCallFromFunc(t, file, 1, 1) // f(), stmt 1: decode(&x, nil)

	byteIdx, ptrIdx, ok := isTransformationCall(call, info)
	if !ok {
		t.Fatal("isTransformationCall should match func(*int, []byte) regardless of parameter order")
	}
	if ptrIdx != 0 {
		t.Errorf("ptrDestIdx = %d, want 0 (pointer is first param)", ptrIdx)
	}
	if byteIdx != 1 {
		t.Errorf("byteArgIdx = %d, want 1 ([]byte is second param)", byteIdx)
	}
}

// --- isByteLikeParam helper unit tests ---

// TestIsByteLikeParam_ByteSlice verifies that []byte is recognized
// as a byte-like parameter type.
func TestIsByteLikeParam_ByteSlice(t *testing.T) {
	typ := types.NewSlice(types.Typ[types.Byte])
	if !isByteLikeParam(typ) {
		t.Error("isByteLikeParam should return true for []byte")
	}
}

// TestIsByteLikeParam_String verifies that string is recognized
// as a byte-like parameter type.
func TestIsByteLikeParam_String(t *testing.T) {
	if !isByteLikeParam(types.Typ[types.String]) {
		t.Error("isByteLikeParam should return true for string")
	}
}

// TestIsByteLikeParam_IoReader verifies that an interface with a
// Read method is recognized as a byte-like parameter type.
func TestIsByteLikeParam_IoReader(t *testing.T) {
	// Construct an interface with a Read([]byte) (int, error) method
	// to simulate io.Reader.
	readSig := types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewSlice(types.Typ[types.Byte]))),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type()),
		),
		false,
	)
	readMethod := types.NewFunc(token.NoPos, nil, "Read", readSig)
	iface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
	iface.Complete()

	if !isByteLikeParam(iface) {
		t.Error("isByteLikeParam should return true for an interface with a Read method")
	}
}

// TestIsByteLikeParam_NonReadInterface verifies that an interface
// without a Read method is not recognized as a byte-like parameter.
func TestIsByteLikeParam_NonReadInterface(t *testing.T) {
	writeSig := types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewSlice(types.Typ[types.Byte]))),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type()),
		),
		false,
	)
	writeMethod := types.NewFunc(token.NoPos, nil, "Write", writeSig)
	iface := types.NewInterfaceType([]*types.Func{writeMethod}, nil)
	iface.Complete()

	if isByteLikeParam(iface) {
		t.Error("isByteLikeParam should return false for an interface with only a Write method (no Read)")
	}
}

// TestIsByteLikeParam_Int verifies that int is not recognized as a
// byte-like parameter type.
func TestIsByteLikeParam_Int(t *testing.T) {
	if isByteLikeParam(types.Typ[types.Int]) {
		t.Error("isByteLikeParam should return false for int")
	}
}

// --- isPointerDestParam helper unit tests ---

// TestIsPointerDestParam_Pointer verifies that *int is recognized
// as a pointer destination parameter type.
func TestIsPointerDestParam_Pointer(t *testing.T) {
	typ := types.NewPointer(types.Typ[types.Int])
	if !isPointerDestParam(typ) {
		t.Error("isPointerDestParam should return true for *int")
	}
}

// TestIsPointerDestParam_EmptyInterface verifies that interface{}
// is recognized as a pointer destination parameter type.
func TestIsPointerDestParam_EmptyInterface(t *testing.T) {
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	if !isPointerDestParam(iface) {
		t.Error("isPointerDestParam should return true for interface{}")
	}
}

// TestIsPointerDestParam_Int verifies that int is not recognized
// as a pointer destination parameter type.
func TestIsPointerDestParam_Int(t *testing.T) {
	if isPointerDestParam(types.Typ[types.Int]) {
		t.Error("isPointerDestParam should return false for int")
	}
}

// --- matchDirect helper unit tests ---

// TestMatchDirect_IdentityMatch verifies that matchDirect returns a
// mapping with confidence 75 when the assertion expression contains
// an identifier whose types.Object is in objToEffectID.
func TestMatchDirect_IdentityMatch(t *testing.T) {
	src := `package p
func f() {
	result := 42
	_ = result + 1
}`
	file, info := parseAndTypeCheck(t, src)

	// result is defined at stmt 0, used in stmt 1's RHS.
	resultObj := extractVarObj(t, file, info, 0, 0)
	rhs := extractRHS(t, file, 0, 1)

	effectID := "effect-return-1"
	objToEffectID := map[types.Object]string{resultObj: effectID}
	effectMap := map[string]*taxonomy.SideEffect{
		effectID: {ID: effectID, Type: taxonomy.ReturnValue},
	}

	site := AssertionSite{
		Location: "test.go:4",
		Kind:     AssertionKindStdlibComparison,
		Expr:     rhs,
	}

	m := matchDirect(site, objToEffectID, effectMap, info, nil)
	if m == nil {
		t.Fatal("matchDirect should return a mapping for identity match")
	}
	if m.Confidence != 75 {
		t.Errorf("confidence = %d, want 75", m.Confidence)
	}
	if m.SideEffectID != effectID {
		t.Errorf("SideEffectID = %q, want %q", m.SideEffectID, effectID)
	}
}

// TestMatchDirect_HelperBridgeMatch verifies that matchDirect returns
// a mapping with confidence 70 when an identifier maps through
// helperBridge to a key in objToEffectID.
func TestMatchDirect_HelperBridgeMatch(t *testing.T) {
	src := `package p
func f() {
	got := 42
	want := 99
	_ = got + want
}`
	file, info := parseAndTypeCheck(t, src)

	gotObj := extractVarObj(t, file, info, 0, 0)  // got := 42
	wantObj := extractVarObj(t, file, info, 0, 1) // want := 99
	rhs := extractRHS(t, file, 0, 2)              // got + want

	// Simulate: gotObj is a helper parameter, callerObj is the real
	// test variable mapped to an effect. The helperBridge maps
	// gotObj → wantObj, and wantObj is in objToEffectID.
	effectID := "effect-return-1"
	objToEffectID := map[types.Object]string{wantObj: effectID}
	effectMap := map[string]*taxonomy.SideEffect{
		effectID: {ID: effectID, Type: taxonomy.ReturnValue},
	}
	helperBridge := map[types.Object]types.Object{gotObj: wantObj}

	site := AssertionSite{
		Location: "test.go:5",
		Kind:     AssertionKindStdlibComparison,
		Expr:     rhs,
	}

	m := matchDirect(site, objToEffectID, effectMap, info, helperBridge)
	if m == nil {
		t.Fatal("matchDirect should return a mapping for helper bridge match")
	}
	if m.Confidence != 70 {
		t.Errorf("confidence = %d, want 70", m.Confidence)
	}
	if m.SideEffectID != effectID {
		t.Errorf("SideEffectID = %q, want %q", m.SideEffectID, effectID)
	}
}

// TestMatchDirect_NoMatch verifies that matchDirect returns nil when
// no identifiers in the expression match objToEffectID or helperBridge.
func TestMatchDirect_NoMatch(t *testing.T) {
	src := `package p
func f() {
	x := 1
	y := 2
	_ = y + 3
	_ = x
}`
	file, info := parseAndTypeCheck(t, src)

	xObj := extractVarObj(t, file, info, 0, 0) // x := 1
	rhs := extractRHS(t, file, 0, 2)           // y + 3

	// Map x but not y — the expression only contains y.
	effectID := "effect-return-1"
	objToEffectID := map[types.Object]string{xObj: effectID}
	effectMap := map[string]*taxonomy.SideEffect{
		effectID: {ID: effectID, Type: taxonomy.ReturnValue},
	}

	site := AssertionSite{
		Location: "test.go:5",
		Kind:     AssertionKindStdlibComparison,
		Expr:     rhs,
	}

	m := matchDirect(site, objToEffectID, effectMap, info, nil)
	if m != nil {
		t.Error("matchDirect should return nil when no identifiers match")
	}
}

// TestMatchDirect_NilExpr verifies that matchDirect returns nil when
// the assertion site has a nil expression.
func TestMatchDirect_NilExpr(t *testing.T) {
	site := AssertionSite{
		Location: "test.go:1",
		Kind:     AssertionKindStdlibComparison,
		Expr:     nil,
	}

	m := matchDirect(site, nil, nil, nil, nil)
	if m != nil {
		t.Error("matchDirect should return nil for nil expression")
	}
}

// --- matchIndirectRoot helper unit tests ---

// TestMatchIndirectRoot_SelectorMatch verifies that matchIndirectRoot
// returns a mapping with confidence 65 when a SelectorExpr's root
// identifier (via resolveExprRoot) is in objToEffectID.
func TestMatchIndirectRoot_SelectorMatch(t *testing.T) {
	src := `package p

type Result struct { Name string }

func f() {
	result := Result{Name: "test"}
	_ = result.Name
}`
	file, info := parseAndTypeCheck(t, src)

	resultObj := extractVarObj(t, file, info, 1, 0) // result := Result{...}
	rhs := extractRHS(t, file, 1, 1)                // result.Name

	effectID := "effect-return-1"
	objToEffectID := map[types.Object]string{resultObj: effectID}
	effectMap := map[string]*taxonomy.SideEffect{
		effectID: {ID: effectID, Type: taxonomy.ReturnValue},
	}

	site := AssertionSite{
		Location: "test.go:7",
		Kind:     AssertionKindStdlibComparison,
		Expr:     rhs,
	}

	m := matchIndirectRoot(site, objToEffectID, effectMap, info)
	if m == nil {
		t.Fatal("matchIndirectRoot should return a mapping for selector match")
	}
	if m.Confidence != 65 {
		t.Errorf("confidence = %d, want 65", m.Confidence)
	}
	if m.SideEffectID != effectID {
		t.Errorf("SideEffectID = %q, want %q", m.SideEffectID, effectID)
	}
}

// TestMatchIndirectRoot_NoComposite verifies that matchIndirectRoot
// returns nil when the expression contains only simple identifiers
// (no SelectorExpr, IndexExpr, or CallExpr nodes).
func TestMatchIndirectRoot_NoComposite(t *testing.T) {
	src := `package p
func f() {
	x := 1
	_ = x + 2
}`
	file, info := parseAndTypeCheck(t, src)

	xObj := extractVarObj(t, file, info, 0, 0) // x := 1
	rhs := extractRHS(t, file, 0, 1)           // x + 2

	effectID := "effect-return-1"
	objToEffectID := map[types.Object]string{xObj: effectID}
	effectMap := map[string]*taxonomy.SideEffect{
		effectID: {ID: effectID, Type: taxonomy.ReturnValue},
	}

	site := AssertionSite{
		Location: "test.go:4",
		Kind:     AssertionKindStdlibComparison,
		Expr:     rhs,
	}

	m := matchIndirectRoot(site, objToEffectID, effectMap, info)
	if m != nil {
		t.Error("matchIndirectRoot should return nil when expression has only simple identifiers")
	}
}

// parseAndTypeCheckWithFset parses Go source code and type-checks it,
// returning the AST file, populated types.Info, and the file set. This
// variant is needed for constructing packages.Package instances in
// traceForwardDataFlow tests.
func parseAndTypeCheckWithFset(t *testing.T, src string) (*ast.File, *types.Info, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	// Type-check errors are expected and intentionally ignored because
	// Importer is nil — the synthetic test code only needs local
	// definitions to be resolved, not imported packages.
	conf := types.Config{Importer: nil}
	_, _ = conf.Check("p", fset, []*ast.File{file}, info)
	return file, info, fset
}

// --- collectTrackedVars helper unit tests ---

// TestCollectTrackedVars_MultipleMatches verifies that collectTrackedVars
// returns only the types.Object keys whose effect ID matches the target
// returnEffectID when multiple entries exist.
func TestCollectTrackedVars_MultipleMatches(t *testing.T) {
	src := `package p
func f() {
	a := 1
	b := 2
	c := 3
	_ = a + b + c
}`
	file, info := parseAndTypeCheck(t, src)

	aObj := extractVarObj(t, file, info, 0, 0) // a := 1
	bObj := extractVarObj(t, file, info, 0, 1) // b := 2
	cObj := extractVarObj(t, file, info, 0, 2) // c := 3

	targetID := "effect-return-1"
	objToEffectID := map[types.Object]string{
		aObj: targetID,
		bObj: "effect-other",
		cObj: targetID,
	}

	tracked := collectTrackedVars(objToEffectID, targetID)
	if len(tracked) != 2 {
		t.Fatalf("len(tracked) = %d, want 2", len(tracked))
	}
	if !tracked[aObj] {
		t.Error("tracked set should contain aObj")
	}
	if !tracked[cObj] {
		t.Error("tracked set should contain cObj")
	}
	if tracked[bObj] {
		t.Error("tracked set should NOT contain bObj (different effect ID)")
	}
}

// TestCollectTrackedVars_NoMatches verifies that collectTrackedVars
// returns an empty map when no entries match the returnEffectID.
func TestCollectTrackedVars_NoMatches(t *testing.T) {
	src := `package p
func f() {
	a := 1
	b := 2
	_ = a + b
}`
	file, info := parseAndTypeCheck(t, src)

	aObj := extractVarObj(t, file, info, 0, 0) // a := 1
	bObj := extractVarObj(t, file, info, 0, 1) // b := 2

	objToEffectID := map[types.Object]string{
		aObj: "effect-other-1",
		bObj: "effect-other-2",
	}

	tracked := collectTrackedVars(objToEffectID, "effect-return-1")
	if len(tracked) != 0 {
		t.Errorf("len(tracked) = %d, want 0", len(tracked))
	}
}

// --- traceForwardDataFlow helper unit tests ---

// TestTraceForwardDataFlow_SimpleChain verifies that traceForwardDataFlow
// traces through a simple field-access assignment: if x is tracked,
// then y := x.Field causes y to be added to the tracked set.
func TestTraceForwardDataFlow_SimpleChain(t *testing.T) {
	src := `package p

type S struct { Field int }

func f() {
	x := S{Field: 42}
	y := x.Field
	_ = y
}`
	file, info, fset := parseAndTypeCheckWithFset(t, src)

	xObj := extractVarObj(t, file, info, 1, 0) // x := S{...}
	yObj := extractVarObj(t, file, info, 1, 1) // y := x.Field

	tracked := map[types.Object]bool{xObj: true}
	pkg := &packages.Package{
		Syntax:    []*ast.File{file},
		TypesInfo: info,
		Fset:      fset,
	}

	result := traceForwardDataFlow(tracked, pkg)
	if !result[yObj] {
		t.Error("traceForwardDataFlow should add y (from y := x.Field) to tracked set")
	}
	if !result[xObj] {
		t.Error("traceForwardDataFlow should preserve original tracked variable x")
	}
}

// TestTraceForwardDataFlow_EmptyTracked verifies that traceForwardDataFlow
// returns the empty tracked set immediately when given no initial variables.
func TestTraceForwardDataFlow_EmptyTracked(t *testing.T) {
	src := `package p
func f() {
	x := 1
	_ = x
}`
	file, info, fset := parseAndTypeCheckWithFset(t, src)

	tracked := map[types.Object]bool{}
	pkg := &packages.Package{
		Syntax:    []*ast.File{file},
		TypesInfo: info,
		Fset:      fset,
	}

	result := traceForwardDataFlow(tracked, pkg)
	if len(result) != 0 {
		t.Errorf("traceForwardDataFlow with empty tracked should return empty, got %d entries", len(result))
	}
}

// TestTraceForwardDataFlow_NonDataExtraction verifies that traceForwardDataFlow
// does NOT add the LHS variable when the RHS is a method call (not a
// data-extraction expression). Method calls like s.Get("key") are gated
// by isDataExtraction to prevent false positives.
func TestTraceForwardDataFlow_NonDataExtraction(t *testing.T) {
	src := `package p

type Store struct{}
func (Store) Get(key string) int { return 0 }

func f() {
	s := Store{}
	got := s.Get("key")
	_ = got
}`
	file, info, fset := parseAndTypeCheckWithFset(t, src)

	// s is at funcIdx=2 (after type decl and method decl), stmtIdx=0
	sObj := extractVarObj(t, file, info, 2, 0)   // s := Store{}
	gotObj := extractVarObj(t, file, info, 2, 1) // got := s.Get("key")

	tracked := map[types.Object]bool{sObj: true}
	pkg := &packages.Package{
		Syntax:    []*ast.File{file},
		TypesInfo: info,
		Fset:      fset,
	}

	result := traceForwardDataFlow(tracked, pkg)
	if result[gotObj] {
		t.Error("traceForwardDataFlow should NOT add got from s.Get(\"key\") — method call is not a data extraction")
	}
	if !result[sObj] {
		t.Error("traceForwardDataFlow should preserve original tracked variable s")
	}
}

// --- matchTrackedInExpr helper unit tests ---

// TestMatchTrackedInExpr_DirectMatch verifies that matchTrackedInExpr
// returns true when the expression contains an identifier whose
// types.Object is directly in the tracked set.
func TestMatchTrackedInExpr_DirectMatch(t *testing.T) {
	src := `package p
func f() {
	x := 42
	_ = x + 1
}`
	file, info := parseAndTypeCheck(t, src)

	xObj := extractVarObj(t, file, info, 0, 0) // x := 42
	rhs := extractRHS(t, file, 0, 1)           // x + 1

	tracked := map[types.Object]bool{xObj: true}
	if !matchTrackedInExpr(rhs, tracked, info) {
		t.Error("matchTrackedInExpr should return true when expression contains tracked identifier")
	}
}

// TestMatchTrackedInExpr_RootResolution verifies that matchTrackedInExpr
// returns true for composite expressions like tracked.Field where the
// root identifier resolves to a tracked variable via resolveExprRoot.
func TestMatchTrackedInExpr_RootResolution(t *testing.T) {
	src := `package p

type S struct { Field int }

func f() {
	tracked := S{Field: 1}
	_ = tracked.Field
}`
	file, info := parseAndTypeCheck(t, src)

	trackedObj := extractVarObj(t, file, info, 1, 0) // tracked := S{...}
	rhs := extractRHS(t, file, 1, 1)                 // tracked.Field

	trackedSet := map[types.Object]bool{trackedObj: true}
	if !matchTrackedInExpr(rhs, trackedSet, info) {
		t.Error("matchTrackedInExpr should return true for tracked.Field via root resolution")
	}
}

// TestMatchTrackedInExpr_NoMatch verifies that matchTrackedInExpr
// returns false when the expression contains no identifiers in the
// tracked set.
func TestMatchTrackedInExpr_NoMatch(t *testing.T) {
	src := `package p
func f() {
	x := 1
	y := 2
	_ = y + 3
	_ = x
}`
	file, info := parseAndTypeCheck(t, src)

	xObj := extractVarObj(t, file, info, 0, 0) // x := 1
	rhs := extractRHS(t, file, 0, 2)           // y + 3

	tracked := map[types.Object]bool{xObj: true}
	if matchTrackedInExpr(rhs, tracked, info) {
		t.Error("matchTrackedInExpr should return false when no identifiers match tracked set")
	}
}

// --- generateSuggestion tests ---

// TestGenerateSuggestion verifies all 5 switch cases + default.
func TestGenerateSuggestion(t *testing.T) {
	tests := []struct {
		name       string
		effectType taxonomy.SideEffectType
		desc       string
		wantParts  []string
	}{
		{
			name:       "LogWrite",
			effectType: taxonomy.LogWrite,
			desc:       "writes to logger",
			wantParts:  []string{"log output", "implementation detail"},
		},
		{
			name:       "StdoutWrite",
			effectType: taxonomy.StdoutWrite,
			desc:       "prints to stdout",
			wantParts:  []string{"stdout"},
		},
		{
			name:       "GoroutineSpawn",
			effectType: taxonomy.GoroutineSpawn,
			desc:       "spawns worker",
			wantParts:  []string{"goroutine lifecycle", "concurrency detail"},
		},
		{
			name:       "ContextCancellation",
			effectType: taxonomy.ContextCancellation,
			desc:       "cancels context",
			wantParts:  []string{"context usage", "implementation detail"},
		},
		{
			name:       "CallbackInvocation",
			effectType: taxonomy.CallbackInvocation,
			desc:       "invokes callback",
			wantParts:  []string{"callback invocation", "implementation detail"},
		},
		{
			name:       "Default_UnknownType",
			effectType: taxonomy.SideEffectType("CustomEffect"),
			desc:       "does something custom",
			wantParts:  []string{"CustomEffect", "contract behavior"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSuggestion(tt.effectType, tt.desc)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("generateSuggestion(%s, %q) = %q, want to contain %q",
						tt.effectType, tt.desc, got, part)
				}
			}
			// Verify the description appears in the output.
			if !strings.Contains(got, tt.desc) {
				t.Errorf("generateSuggestion(%s, %q) = %q, want to contain description %q",
					tt.effectType, tt.desc, got, tt.desc)
			}
		})
	}
}

// --- rhsReferencesAnyTracked tests ---

// TestRhsReferencesAnyTracked_DirectMatch verifies direct containsObject path.
func TestRhsReferencesAnyTracked_DirectMatch(t *testing.T) {
	src := `package p
func f() {
	x := 1
	y := x
	_ = y
}`
	file, info := parseAndTypeCheck(t, src)
	xObj := extractVarObj(t, file, info, 0, 0) // x := 1
	rhs := extractRHS(t, file, 0, 1)           // x (RHS of y := x)

	tracked := map[types.Object]bool{xObj: true}
	if !rhsReferencesAnyTracked(rhs, tracked, info) {
		t.Error("rhsReferencesAnyTracked should return true when tracked var is directly in RHS")
	}
}

// TestRhsReferencesAnyTracked_ResolveExprRootFallback verifies resolveExprRoot path.
func TestRhsReferencesAnyTracked_ResolveExprRootFallback(t *testing.T) {
	src := `package p
type S struct{ Field int }
func f() {
	x := S{Field: 1}
	y := x.Field
	_ = y
}`
	file, info := parseAndTypeCheck(t, src)
	xObj := extractVarObj(t, file, info, 1, 0) // x := S{...} (func is Decls[1] after type decl)
	rhs := extractRHS(t, file, 1, 1)           // x.Field

	tracked := map[types.Object]bool{xObj: true}
	if !rhsReferencesAnyTracked(rhs, tracked, info) {
		t.Error("rhsReferencesAnyTracked should return true via resolveExprRoot for x.Field")
	}
}

// TestRhsReferencesAnyTracked_NoMatch verifies false when no tracked var in RHS.
func TestRhsReferencesAnyTracked_NoMatch(t *testing.T) {
	src := `package p
func f() {
	x := 1
	y := 2
	_ = x
	_ = y
}`
	file, info := parseAndTypeCheck(t, src)
	xObj := extractVarObj(t, file, info, 0, 0) // x := 1
	rhs := extractRHS(t, file, 0, 1)           // 2 (RHS of y := 2)

	tracked := map[types.Object]bool{xObj: true}
	if rhsReferencesAnyTracked(rhs, tracked, info) {
		t.Error("rhsReferencesAnyTracked should return false when no tracked var in RHS")
	}
}

// --- handleTransformationCalls tests ---

// TestHandleTransformationCalls_TransformWithTrackedArg verifies transformation
// call with tracked data returns the pointer destination.
func TestHandleTransformationCalls_TransformWithTrackedArg(t *testing.T) {
	// Use a local function stub matching the transformation pattern
	// (byte-like param + pointer dest) instead of importing encoding/json.
	src := `package p

func unmarshal(data []byte, v *map[string]any) {}
func f() {
	data := []byte("{}")
	var target map[string]any
	unmarshal(data, &target)
}`
	file, info := parseAndTypeCheck(t, src)
	dataObj := extractVarObj(t, file, info, 1, 0) // data := []byte("{}")

	// Extract the unmarshal call (it's an ExprStmt, not AssignStmt).
	call := extractCallFromFunc(t, file, 1, 2)

	tracked := map[types.Object]bool{dataObj: true}
	dest, handled := handleTransformationCalls(call, tracked, info)
	if !handled {
		t.Fatal("handleTransformationCalls should handle transformation call with tracked arg")
	}
	if dest == nil {
		t.Fatal("handleTransformationCalls should return non-nil dest for pointer arg")
	}
	if dest.Name() != "target" {
		t.Errorf("dest.Name() = %q, want %q", dest.Name(), "target")
	}
}

// TestHandleTransformationCalls_NoTransformation verifies non-transformation call.
func TestHandleTransformationCalls_NoTransformation(t *testing.T) {
	src := `package p
func add(a, b int) int { return a + b }
func f() {
	x := 1
	add(x, 2)
}`
	file, info := parseAndTypeCheck(t, src)
	xObj := extractVarObj(t, file, info, 1, 0) // x := 1

	call := extractCallFromFunc(t, file, 1, 1) // add(x, 2)

	tracked := map[types.Object]bool{xObj: true}
	_, handled := handleTransformationCalls(call, tracked, info)
	if handled {
		t.Error("handleTransformationCalls should return handled=false for non-transformation call")
	}
}

// TestHandleTransformationCalls_NoTrackedArg verifies transformation call
// without tracked arg returns handled=false.
func TestHandleTransformationCalls_NoTrackedArg(t *testing.T) {
	src := `package p

func unmarshal(data []byte, v *map[string]any) {}
func f() {
	unrelated := 42
	data := []byte("{}")
	var target map[string]any
	unmarshal(data, &target)
	_ = unrelated
}`
	file, info := parseAndTypeCheck(t, src)
	unrelObj := extractVarObj(t, file, info, 1, 0) // unrelated := 42
	call := extractCallFromFunc(t, file, 1, 3)     // unmarshal(data, &target)

	tracked := map[types.Object]bool{unrelObj: true}
	_, handled := handleTransformationCalls(call, tracked, info)
	if handled {
		t.Error("handleTransformationCalls should return handled=false when no tracked arg flows into call")
	}
}

// --- extractDataFlowLHS tests ---

// TestExtractDataFlowLHS_ValidIdent verifies LHS extraction for valid identifier.
func TestExtractDataFlowLHS_ValidIdent(t *testing.T) {
	src := `package p
func f() {
	x := 42
	_ = x
}`
	file, info := parseAndTypeCheck(t, src)

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected *ast.FuncDecl")
	}
	assign, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok {
		t.Fatal("expected *ast.AssignStmt")
	}

	obj := extractDataFlowLHS(assign, 0, info)
	if obj == nil {
		t.Fatal("extractDataFlowLHS should return non-nil for valid LHS ident")
	}
	if obj.Name() != "x" {
		t.Errorf("obj.Name() = %q, want %q", obj.Name(), "x")
	}
}

// TestExtractDataFlowLHS_BlankIdent verifies nil return for blank identifier "_".
func TestExtractDataFlowLHS_BlankIdent(t *testing.T) {
	src := `package p
func f() {
	_ = 42
}`
	file, info := parseAndTypeCheck(t, src)

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected *ast.FuncDecl")
	}
	assign, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok {
		t.Fatal("expected *ast.AssignStmt")
	}

	obj := extractDataFlowLHS(assign, 0, info)
	if obj != nil {
		t.Error("extractDataFlowLHS should return nil for blank identifier '_'")
	}
}

// TestExtractDataFlowLHS_NonIdentLHS verifies nil return for non-identifier LHS (e.g., s.Field).
func TestExtractDataFlowLHS_NonIdentLHS(t *testing.T) {
	src := `package p
type S struct{ Field int }
func f() {
	var s S
	s.Field = 42
	_ = s
}`
	file, info := parseAndTypeCheck(t, src)

	fn, ok := file.Decls[1].(*ast.FuncDecl) // func f() is Decls[1] after type S
	if !ok {
		t.Fatal("expected *ast.FuncDecl")
	}
	// s.Field = 42 is the second statement (after var s S)
	assign, ok := fn.Body.List[1].(*ast.AssignStmt)
	if !ok {
		t.Fatal("expected *ast.AssignStmt for s.Field = 42")
	}

	obj := extractDataFlowLHS(assign, 0, info)
	if obj != nil {
		t.Error("extractDataFlowLHS should return nil for SelectorExpr LHS (s.Field)")
	}
}

// TestExtractDataFlowLHS_Reassignment verifies the info.Uses fallback for = reassignment.
func TestExtractDataFlowLHS_Reassignment(t *testing.T) {
	src := `package p
func f() {
	x := 1
	x = 2
	_ = x
}`
	file, info := parseAndTypeCheck(t, src)

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected *ast.FuncDecl")
	}
	assign, ok := fn.Body.List[1].(*ast.AssignStmt) // x = 2 (reassignment)
	if !ok {
		t.Fatal("expected *ast.AssignStmt for x = 2")
	}

	obj := extractDataFlowLHS(assign, 0, info)
	if obj == nil {
		t.Fatal("extractDataFlowLHS should return non-nil for reassignment via info.Uses")
	}
	if obj.Name() != "x" {
		t.Errorf("obj.Name() = %q, want %q", obj.Name(), "x")
	}
}

// TestExtractDataFlowLHS_OutOfRange verifies nil return when rhsIdx is out of range.
func TestExtractDataFlowLHS_OutOfRange(t *testing.T) {
	src := `package p
func f() {
	x := 42
	_ = x
}`
	file, info := parseAndTypeCheck(t, src)

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected *ast.FuncDecl")
	}
	assign, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok {
		t.Fatal("expected *ast.AssignStmt")
	}

	// rhsIdx=5 is beyond len(assign.Lhs)
	obj := extractDataFlowLHS(assign, 5, info)
	if obj != nil {
		t.Error("extractDataFlowLHS should return nil when rhsIdx is out of range")
	}
}

// TestTraceForwardDataFlow_MultiIteration verifies that traceForwardDataFlow
// propagates tracked variables across multiple iterations through chained
// data-extraction assignments: a → b → c.
func TestTraceForwardDataFlow_MultiIteration(t *testing.T) {
	src := `package p

type Inner struct{ Name string }
type Outer struct{ Items []Inner }
type Result struct{ Data Outer }

func f() {
	r := Result{Data: Outer{Items: []Inner{{Name: "x"}}}}
	a := r.Data
	b := a.Items[0]
	c := b.Name
	_ = c
}`
	file, info, fset := parseAndTypeCheckWithFset(t, src)

	// func f() is Decls[3] (after 3 type declarations)
	rObj := extractVarObj(t, file, info, 3, 0) // r := Result{...}
	aObj := extractVarObj(t, file, info, 3, 1) // a := r.Data
	bObj := extractVarObj(t, file, info, 3, 2) // b := a.Items[0]
	cObj := extractVarObj(t, file, info, 3, 3) // c := b.Name

	tracked := map[types.Object]bool{rObj: true}
	pkg := &packages.Package{
		Syntax:    []*ast.File{file},
		TypesInfo: info,
		Fset:      fset,
	}

	result := traceForwardDataFlow(tracked, pkg)

	if !result[rObj] {
		t.Error("should preserve original tracked variable r")
	}
	if !result[aObj] {
		t.Error("iteration 1: should track a (from a := r.Data)")
	}
	if !result[bObj] {
		t.Error("iteration 2: should track b (from b := a.Items[0])")
	}
	if !result[cObj] {
		t.Error("iteration 3: should track c (from c := b.Name)")
	}
}
