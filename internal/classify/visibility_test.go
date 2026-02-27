package classify_test

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// findFuncDeclInFiles searches AST files for a function declaration
// by name, optionally matching a receiver type name.
func findFuncDeclInFiles(files []*ast.File, funcName, recvTypeName string) *ast.FuncDecl {
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != funcName {
				continue
			}
			if recvTypeName == "" {
				// Package-level function.
				if fd.Recv == nil {
					return fd
				}
				continue
			}
			// Method — match receiver type name.
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				rtype := fd.Recv.List[0].Type
				// Unwrap *T to T.
				if star, ok := rtype.(*ast.StarExpr); ok {
					rtype = star.X
				}
				if ident, ok := rtype.(*ast.Ident); ok && ident.Name == recvTypeName {
					return fd
				}
			}
		}
	}
	return nil
}

// TestAnalyzeVisibilitySignal_ExportedFunction verifies that an
// exported package-level function produces the exported-function
// weight dimension (FR-001).
func TestAnalyzeVisibilitySignal_ExportedFunction(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// GetData is an exported package-level function with no exported
	// return type (returns []byte) and no receiver.
	funcDecl := findFuncDeclInFiles(contractsPkg.Syntax, "GetData", "")
	if funcDecl == nil {
		t.Fatal("GetData func decl not found")
	}
	funcObj := contractsPkg.Types.Scope().Lookup("GetData")
	if funcObj == nil {
		t.Fatal("GetData types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, taxonomy.ReturnValue)

	// Exported function (+8), no exported return (+0), no receiver (+0) = 8.
	if sig.Weight != 8 {
		t.Errorf("GetData: weight = %d, want 8", sig.Weight)
	}
	if sig.Source != "visibility" {
		t.Errorf("GetData: source = %q, want %q", sig.Source, "visibility")
	}
	if !strings.Contains(sig.Reasoning, "function is exported") {
		t.Errorf("GetData: reasoning %q does not contain %q",
			sig.Reasoning, "function is exported")
	}
}

// TestAnalyzeVisibilitySignal_ExportedReturnType verifies that an
// exported return type adds the return-type weight dimension (FR-001).
func TestAnalyzeVisibilitySignal_ExportedReturnType(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// ComputeResult is exported, returns ExportedResult (exported type),
	// no receiver. Weight = 8 (func) + 6 (return) = 14.
	funcDecl := findFuncDeclInFiles(contractsPkg.Syntax, "ComputeResult", "")
	if funcDecl == nil {
		t.Fatal("ComputeResult func decl not found")
	}
	funcObj := contractsPkg.Types.Scope().Lookup("ComputeResult")
	if funcObj == nil {
		t.Fatal("ComputeResult types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, taxonomy.ReturnValue)

	if sig.Weight != 14 {
		t.Errorf("ComputeResult: weight = %d, want 14", sig.Weight)
	}
	if !strings.Contains(sig.Reasoning, "return type is exported") {
		t.Errorf("ComputeResult: reasoning %q does not contain %q",
			sig.Reasoning, "return type is exported")
	}
}

// TestAnalyzeVisibilitySignal_ExportedReceiverType verifies that an
// exported receiver type adds the receiver-type weight dimension
// (FR-001).
func TestAnalyzeVisibilitySignal_ExportedReceiverType(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// (*FileStore).Save is exported, receiver FileStore is exported,
	// returns error (builtin, not exported type).
	// Weight = 8 (func) + 6 (receiver) = 14.
	funcDecl := findFuncDeclInFiles(contractsPkg.Syntax, "Save", "FileStore")
	if funcDecl == nil {
		t.Fatal("FileStore.Save func decl not found")
	}

	// For methods, we need to look up via the type's method set.
	fsObj := contractsPkg.Types.Scope().Lookup("FileStore")
	if fsObj == nil {
		t.Fatal("FileStore type not found")
	}
	named, ok := fsObj.Type().(*types.Named)
	if !ok {
		t.Fatal("FileStore is not a named type")
	}
	// Find the Save method on *FileStore (pointer receiver).
	ptrType := types.NewPointer(named)
	mset := types.NewMethodSet(ptrType)
	var saveObj types.Object
	for i := 0; i < mset.Len(); i++ {
		if mset.At(i).Obj().Name() == "Save" {
			saveObj = mset.At(i).Obj()
			break
		}
	}
	if saveObj == nil {
		t.Fatal("Save method not found on *FileStore")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, saveObj, taxonomy.ReceiverMutation)

	if sig.Weight != 14 {
		t.Errorf("FileStore.Save: weight = %d, want 14", sig.Weight)
	}
	if !strings.Contains(sig.Reasoning, "receiver type is exported") {
		t.Errorf("FileStore.Save: reasoning %q does not contain %q",
			sig.Reasoning, "receiver type is exported")
	}
}

// TestAnalyzeVisibilitySignal_WeightClamp verifies that the total
// weight is clamped to maxVisibilityWeight (20) when all three
// dimensions are present (FR-001).
func TestAnalyzeVisibilitySignal_WeightClamp(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// (*RemoteFetcher).Fetch: exported method (+8), receiver
	// RemoteFetcher is exported (+6), returns ([]byte, error) —
	// []byte is not an exported named type, error is builtin.
	// So this is only 8 + 6 = 14, not clamped.
	//
	// For a clamped scenario, we need all 3 dimensions. We can
	// construct a synthetic AST node with exported function,
	// exported return type, and exported receiver.
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent("DoStuff"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.StarExpr{X: ast.NewIdent("MyType")}},
			},
		},
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("Result")},
				},
			},
		},
	}

	// Create a synthetic exported types.Object. We use a real
	// exported function from the contracts package since we need
	// Exported() to return true.
	funcObj := contractsPkg.Types.Scope().Lookup("GetData")
	if funcObj == nil {
		t.Fatal("GetData types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, taxonomy.ReturnValue)

	// 8 (exported func) + 6 (exported return) + 6 (exported receiver)
	// = 20, clamped to 20.
	if sig.Weight != 20 {
		t.Errorf("clamped: weight = %d, want 20", sig.Weight)
	}
}

// TestAnalyzeVisibilitySignal_AllDimensionsInReasoning verifies that
// the reasoning string mentions each matching dimension (FR-002).
func TestAnalyzeVisibilitySignal_AllDimensionsInReasoning(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// Use the synthetic scenario from the clamp test above to get
	// all three dimensions.
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent("DoStuff"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.StarExpr{X: ast.NewIdent("MyType")}},
			},
		},
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("Result")},
				},
			},
		},
	}

	funcObj := contractsPkg.Types.Scope().Lookup("GetData")
	if funcObj == nil {
		t.Fatal("GetData types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, taxonomy.ReturnValue)

	expected := []string{
		"function is exported",
		"return type is exported",
		"receiver type is exported",
	}
	for _, want := range expected {
		if !strings.Contains(sig.Reasoning, want) {
			t.Errorf("reasoning %q does not contain %q", sig.Reasoning, want)
		}
	}
}

// TestAnalyzeVisibilitySignal_UnexportedFunction verifies that an
// unexported function produces a zero signal (FR-003).
func TestAnalyzeVisibilitySignal_UnexportedFunction(t *testing.T) {
	pkgs := loadTestPackages(t)
	incidentalPkg := findPackage(pkgs, "incidental")
	if incidentalPkg == nil {
		t.Fatal("incidental package not found")
	}

	// debugTrace is unexported.
	funcDecl := findFuncDeclInFiles(incidentalPkg.Syntax, "debugTrace", "")
	if funcDecl == nil {
		t.Fatal("debugTrace func decl not found")
	}

	// For unexported functions, Scope().Lookup won't find them
	// (they're not in the package scope). We need to search
	// TypesInfo.Defs.
	var funcObj types.Object
	for ident, obj := range incidentalPkg.TypesInfo.Defs {
		if ident.Name == "debugTrace" && obj != nil {
			if _, ok := obj.(*types.Func); ok {
				funcObj = obj
				break
			}
		}
	}
	if funcObj == nil {
		t.Fatal("debugTrace types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, taxonomy.LogWrite)

	if sig.Weight != 0 {
		t.Errorf("unexported debugTrace: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("unexported debugTrace: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeVisibilitySignal_NilFuncDecl verifies that nil funcDecl
// returns a zero signal (FR-003).
func TestAnalyzeVisibilitySignal_NilFuncDecl(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	funcObj := contractsPkg.Types.Scope().Lookup("GetData")
	if funcObj == nil {
		t.Fatal("GetData types.Object not found")
	}

	sig := classify.AnalyzeVisibilitySignal(nil, funcObj, taxonomy.ReturnValue)

	if sig.Weight != 0 {
		t.Errorf("nil funcDecl: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("nil funcDecl: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeVisibilitySignal_NilFuncObj verifies that nil funcObj
// returns a zero signal (FR-003).
func TestAnalyzeVisibilitySignal_NilFuncObj(t *testing.T) {
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent("Foo"),
		Type: &ast.FuncType{},
	}

	sig := classify.AnalyzeVisibilitySignal(funcDecl, nil, taxonomy.ReturnValue)

	if sig.Weight != 0 {
		t.Errorf("nil funcObj: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("nil funcObj: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeVisibilitySignal_TableDriven uses a table-driven
// approach to verify weight computation across multiple functions
// (FR-001).
func TestAnalyzeVisibilitySignal_TableDriven(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	tests := []struct {
		name            string
		funcName        string
		recvType        string
		effectType      taxonomy.SideEffectType
		wantWeight      int
		wantInReasoning []string
	}{
		{
			name:            "exported func only (GetData)",
			funcName:        "GetData",
			effectType:      taxonomy.ReturnValue,
			wantWeight:      8,
			wantInReasoning: []string{"function is exported"},
		},
		{
			name:       "exported func + exported return (ComputeResult)",
			funcName:   "ComputeResult",
			effectType: taxonomy.ReturnValue,
			wantWeight: 14,
			wantInReasoning: []string{
				"function is exported",
				"return type is exported",
			},
		},
		{
			name:       "exported func + exported return with error (ApplyTransform)",
			funcName:   "ApplyTransform",
			effectType: taxonomy.ReturnValue,
			wantWeight: 14,
			wantInReasoning: []string{
				"function is exported",
				"return type is exported",
			},
		},
		{
			name:       "exported method + exported receiver (FileStore.Save)",
			funcName:   "Save",
			recvType:   "FileStore",
			effectType: taxonomy.ReceiverMutation,
			wantWeight: 14,
			wantInReasoning: []string{
				"function is exported",
				"receiver type is exported",
			},
		},
		{
			name:       "exported method + exported receiver (FileStore.Write)",
			funcName:   "Write",
			recvType:   "FileStore",
			effectType: taxonomy.ReceiverMutation,
			wantWeight: 14,
			wantInReasoning: []string{
				"function is exported",
				"receiver type is exported",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcDecl := findFuncDeclInFiles(contractsPkg.Syntax, tt.funcName, tt.recvType)
			if funcDecl == nil {
				t.Fatalf("%s func decl not found", tt.funcName)
			}

			var funcObj types.Object
			if tt.recvType == "" {
				funcObj = contractsPkg.Types.Scope().Lookup(tt.funcName)
			} else {
				typeObj := contractsPkg.Types.Scope().Lookup(tt.recvType)
				if typeObj == nil {
					t.Fatalf("%s type not found", tt.recvType)
				}
				named, ok := typeObj.Type().(*types.Named)
				if !ok {
					t.Fatalf("%s is not a named type", tt.recvType)
				}
				ptrType := types.NewPointer(named)
				mset := types.NewMethodSet(ptrType)
				for i := 0; i < mset.Len(); i++ {
					if mset.At(i).Obj().Name() == tt.funcName {
						funcObj = mset.At(i).Obj()
						break
					}
				}
			}
			if funcObj == nil {
				t.Fatalf("%s types.Object not found", tt.funcName)
			}

			sig := classify.AnalyzeVisibilitySignal(funcDecl, funcObj, tt.effectType)

			if sig.Weight != tt.wantWeight {
				t.Errorf("weight = %d, want %d", sig.Weight, tt.wantWeight)
			}
			for _, want := range tt.wantInReasoning {
				if !strings.Contains(sig.Reasoning, want) {
					t.Errorf("reasoning %q does not contain %q",
						sig.Reasoning, want)
				}
			}
		})
	}
}
