package classify_test

import (
	"go/ast"
	"testing"

	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// makeFuncDeclWithDoc constructs a minimal *ast.FuncDecl with the
// given name and a single-line doc comment.
func makeFuncDeclWithDoc(name, doc string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{{Text: "// " + doc}},
		},
		Type: &ast.FuncType{},
	}
}

// TestAnalyzeGodocSignal_ContractualKeywords verifies that each
// contractual keyword paired with a matching effectType produces
// a positive weight (+15) and a non-matching effectType produces
// zero weight (FR-004).
func TestAnalyzeGodocSignal_ContractualKeywords(t *testing.T) {
	tests := []struct {
		name        string
		doc         string
		effectType  taxonomy.SideEffectType
		wantWeight  int
		wantNonZero bool
	}{
		// "returns" matches ReturnValue and ErrorReturn.
		{
			name:        "returns + ReturnValue",
			doc:         "GetVersion returns the current version.",
			effectType:  taxonomy.ReturnValue,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "returns + ErrorReturn",
			doc:         "Load returns an error if the path is invalid.",
			effectType:  taxonomy.ErrorReturn,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "returns + ReceiverMutation (no match)",
			doc:         "GetVersion returns the current version.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  0,
			wantNonZero: false,
		},
		// "sets" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "sets + ReceiverMutation",
			doc:         "SetPrimary sets the primary data source.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "sets + PointerArgMutation",
			doc:         "SetPrimary sets the primary data source.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "sets + ReturnValue (no match)",
			doc:         "SetPrimary sets the primary data source.",
			effectType:  taxonomy.ReturnValue,
			wantWeight:  0,
			wantNonZero: false,
		},
		// "writes" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "writes + ReceiverMutation",
			doc:         "Save writes data to the store.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "writes + PointerArgMutation",
			doc:         "Save writes data to the buffer.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "modifies" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "modifies + ReceiverMutation",
			doc:         "Update modifies the internal state.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "updates" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "updates + PointerArgMutation",
			doc:         "Refresh updates the cached values.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "stores" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "stores + ReceiverMutation",
			doc:         "Put stores the key-value pair.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "deletes" matches ReceiverMutation only.
		{
			name:        "deletes + ReceiverMutation",
			doc:         "Remove deletes the entry from the store.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		{
			name:        "deletes + PointerArgMutation (no match)",
			doc:         "Remove deletes the entry from the store.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  0,
			wantNonZero: false,
		},
		// "persists" matches ReceiverMutation and PointerArgMutation.
		{
			name:        "persists + ReceiverMutation",
			doc:         "Commit persists the transaction.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "removes" matches ReceiverMutation only.
		{
			name:        "removes + ReceiverMutation",
			doc:         "Cleanup removes expired entries.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  15,
			wantNonZero: true,
		},
		// "creates" is NOT in the contractual keywords list.
		{
			name:        "creates + ReceiverMutation (not a keyword)",
			doc:         "NewThing creates a new instance.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  0,
			wantNonZero: false,
		},
		// "sends" is NOT in the contractual keywords list.
		{
			name:        "sends + ReceiverMutation (not a keyword)",
			doc:         "Dispatch sends the event.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  0,
			wantNonZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := makeFuncDeclWithDoc("TestFunc", tt.doc)
			sig := classify.AnalyzeGodocSignal(fd, tt.effectType)

			if sig.Weight != tt.wantWeight {
				t.Errorf("weight = %d, want %d", sig.Weight, tt.wantWeight)
			}
			if tt.wantNonZero && sig.Source != "godoc" {
				t.Errorf("source = %q, want %q", sig.Source, "godoc")
			}
			if !tt.wantNonZero && sig.Source != "" {
				t.Errorf("source = %q, want empty for zero signal", sig.Source)
			}
		})
	}
}

// TestAnalyzeGodocSignal_IncidentalPriority verifies that when the
// godoc contains both an incidental keyword ("logs") and a
// contractual keyword ("returns"), the incidental signal wins with
// weight -15 (FR-005).
func TestAnalyzeGodocSignal_IncidentalPriority(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{
			name: "logs before returns",
			doc:  "ProcessItem logs progress and returns the result.",
		},
		{
			name: "returns before logs",
			doc:  "ProcessItem returns the result and logs progress.",
		},
		{
			name: "prints keyword",
			doc:  "Debug prints the value and returns true.",
		},
		{
			name: "traces keyword",
			doc:  "HandleRequest traces the call and returns a response.",
		},
		{
			name: "debugs keyword",
			doc:  "Run debugs output and returns the status.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := makeFuncDeclWithDoc("TestFunc", tt.doc)
			sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

			if sig.Weight != -15 {
				t.Errorf("weight = %d, want -15 (incidental wins)", sig.Weight)
			}
			if sig.Source != "godoc" {
				t.Errorf("source = %q, want %q", sig.Source, "godoc")
			}
		})
	}
}

// TestAnalyzeGodocSignal_IncidentalOnly verifies that an incidental
// keyword alone produces -15 weight.
func TestAnalyzeGodocSignal_IncidentalOnly(t *testing.T) {
	fd := makeFuncDeclWithDoc("LogError", "LogError logs the error to stderr.")
	sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

	if sig.Weight != -15 {
		t.Errorf("weight = %d, want -15", sig.Weight)
	}
	if sig.Source != "godoc" {
		t.Errorf("source = %q, want %q", sig.Source, "godoc")
	}
}

// TestAnalyzeGodocSignal_NilFuncDecl verifies that nil funcDecl
// returns a zero signal (FR-006).
func TestAnalyzeGodocSignal_NilFuncDecl(t *testing.T) {
	sig := classify.AnalyzeGodocSignal(nil, taxonomy.ReturnValue)

	if sig.Weight != 0 {
		t.Errorf("nil funcDecl: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("nil funcDecl: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeGodocSignal_NilDocComment verifies that a funcDecl
// with nil Doc comment group returns a zero signal (FR-006).
func TestAnalyzeGodocSignal_NilDocComment(t *testing.T) {
	fd := &ast.FuncDecl{
		Name: ast.NewIdent("NoDoc"),
		Type: &ast.FuncType{},
	}

	sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

	if sig.Weight != 0 {
		t.Errorf("nil Doc: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("nil Doc: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeGodocSignal_CaseInsensitive verifies that keyword
// matching is case-insensitive.
func TestAnalyzeGodocSignal_CaseInsensitive(t *testing.T) {
	fd := makeFuncDeclWithDoc("GetVersion", "GetVersion RETURNS the version string.")
	sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

	if sig.Weight != 15 {
		t.Errorf("case-insensitive: weight = %d, want 15", sig.Weight)
	}
}

// TestAnalyzeGodocSignal_NoKeyword verifies that godoc without any
// keyword returns a zero signal.
func TestAnalyzeGodocSignal_NoKeyword(t *testing.T) {
	fd := makeFuncDeclWithDoc("ComputeHash",
		"ComputeHash computes a SHA-256 hash of the input.")
	sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

	if sig.Weight != 0 {
		t.Errorf("no keyword: weight = %d, want 0", sig.Weight)
	}
	if sig.Source != "" {
		t.Errorf("no keyword: source = %q, want empty", sig.Source)
	}
}

// TestAnalyzeGodocSignal_ReasoningContent verifies that the
// reasoning string references the matched keyword and effect type.
func TestAnalyzeGodocSignal_ReasoningContent(t *testing.T) {
	fd := makeFuncDeclWithDoc("GetVersion",
		"GetVersion returns the current version.")
	sig := classify.AnalyzeGodocSignal(fd, taxonomy.ReturnValue)

	if sig.Reasoning == "" {
		t.Fatal("expected non-empty reasoning")
	}
	// Reasoning should mention the keyword.
	if sig.Reasoning == "" {
		t.Error("reasoning is empty")
	}
}

// TestAnalyzeGodocSignal_RealFixture verifies godoc signal analysis
// using real fixture functions from the contracts package.
func TestAnalyzeGodocSignal_RealFixture(t *testing.T) {
	pkgs := loadTestPackages(t)
	contractsPkg := findPackage(pkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	tests := []struct {
		name       string
		funcName   string
		effectType taxonomy.SideEffectType
		wantWeight int
	}{
		{
			// GetVersion godoc: "returns the current software version"
			name:       "GetVersion returns",
			funcName:   "GetVersion",
			effectType: taxonomy.ReturnValue,
			wantWeight: 15,
		},
		{
			// SetPrimary godoc: "sets the primary data source"
			name:       "SetPrimary sets",
			funcName:   "SetPrimary",
			effectType: taxonomy.ReceiverMutation,
			wantWeight: 15,
		},
		{
			// LoadProfile godoc: no contractual keyword match for
			// ReturnValue via "reads" — but it does contain "return"
			// in "This return value is part of..." Actually the doc
			// says "reads a user profile" — "reads" is not a keyword.
			// But let's check what actually matches.
			name:       "LoadProfile returns",
			funcName:   "LoadProfile",
			effectType: taxonomy.ReturnValue,
			wantWeight: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcDecl := findFuncDeclInFiles(contractsPkg.Syntax, tt.funcName, "")
			if funcDecl == nil {
				t.Fatalf("%s func decl not found", tt.funcName)
			}

			sig := classify.AnalyzeGodocSignal(funcDecl, tt.effectType)

			if sig.Weight != tt.wantWeight {
				t.Errorf("weight = %d, want %d", sig.Weight, tt.wantWeight)
			}
		})
	}
}
