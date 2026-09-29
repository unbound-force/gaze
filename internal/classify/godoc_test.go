package classify_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
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
			name:        "returns + ReceiverMutation (indirect match)",
			doc:         "GetVersion returns the current version.",
			effectType:  taxonomy.ReceiverMutation,
			wantWeight:  5,
			wantNonZero: true,
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
			name:        "sets + ReturnValue (indirect match)",
			doc:         "SetPrimary sets the primary data source.",
			effectType:  taxonomy.ReturnValue,
			wantWeight:  5,
			wantNonZero: true,
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
			name:        "deletes + PointerArgMutation (indirect match)",
			doc:         "Remove deletes the entry from the store.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  5,
			wantNonZero: true,
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
		// Reduced signal: keyword found but effect type doesn't match.
		{
			name:        "returns + PointerArgMutation (reduced signal)",
			doc:         "Returns a new value configured with defaults.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  5,
			wantNonZero: true,
		},
		// Full-weight signal unchanged: keyword matches effect type.
		{
			name:        "returns + ReturnValue (full weight unchanged)",
			doc:         "Returns a value to the caller.",
			effectType:  taxonomy.ReturnValue,
			wantWeight:  15,
			wantNonZero: true,
		},
		// No contractual keyword at all: zero signal, no spurious reduced signal.
		{
			name:        "no keyword + PointerArgMutation (zero signal)",
			doc:         "Processes the data efficiently.",
			effectType:  taxonomy.PointerArgMutation,
			wantWeight:  0,
			wantNonZero: false,
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
			if tt.wantNonZero && sig.Source == "" {
				t.Errorf("source is empty, want non-empty for non-zero signal")
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
// weight -15 for I/O effect types (FR-005). See issue #105 for the
// type guard that scopes this to I/O effects only.
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
			// I/O effect type: incidental keyword wins over contractual.
			sig := classify.AnalyzeGodocSignal(fd, taxonomy.LogWrite)

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
// keyword alone produces -15 weight for I/O effect types.
func TestAnalyzeGodocSignal_IncidentalOnly(t *testing.T) {
	fd := makeFuncDeclWithDoc("LogError", "LogError logs the error to stderr.")
	// I/O effect type: incidental keyword applies.
	sig := classify.AnalyzeGodocSignal(fd, taxonomy.LogWrite)

	if sig.Weight != -15 {
		t.Errorf("weight = %d, want -15", sig.Weight)
	}
	if sig.Source != "godoc" {
		t.Errorf("source = %q, want %q", sig.Source, "godoc")
	}
}

// TestAnalyzeGodocSignal_IncidentalTypeGuard verifies that incidental
// godoc keywords only apply to I/O effect types, not to P0 effects
// like ReturnValue. See issue #105.
func TestAnalyzeGodocSignal_IncidentalTypeGuard(t *testing.T) {
	tests := []struct {
		name       string
		funcName   string
		doc        string
		effectType taxonomy.SideEffectType
		wantWeight int
	}{
		// P0 effects with incidental keywords → no penalty.
		{
			name:       "ReturnValue/logs",
			funcName:   "LogAndCompute",
			doc:        "LogAndCompute logs the request and computes a result.",
			effectType: taxonomy.ReturnValue,
			wantWeight: 0,
		},
		{
			name:       "ErrorReturn/debugs",
			funcName:   "DebugAndFetch",
			doc:        "DebugAndFetch debugs the connection and fetches data.",
			effectType: taxonomy.ErrorReturn,
			wantWeight: 0,
		},
		{
			name:       "ReceiverMutation/traces",
			funcName:   "TraceState",
			doc:        "TraceState traces state changes on the receiver.",
			effectType: taxonomy.ReceiverMutation,
			wantWeight: 0,
		},

		// Non-I/O P1/P2 effects → no penalty (boundary: not in appliesTo).
		{
			name:       "ChannelSend/logs",
			funcName:   "LogEvents",
			doc:        "LogEvents logs events and sends them on a channel.",
			effectType: taxonomy.ChannelSend,
			wantWeight: 0,
		},

		// I/O effects with incidental keywords → penalty applied.
		{
			name:       "LogWrite/logs",
			funcName:   "LogAndCompute",
			doc:        "LogAndCompute logs the request and computes a result.",
			effectType: taxonomy.LogWrite,
			wantWeight: -15,
		},
		{
			name:       "StderrWrite/prints",
			funcName:   "PrintWarning",
			doc:        "PrintWarning prints the warning to stderr.",
			effectType: taxonomy.StderrWrite,
			wantWeight: -15,
		},
		{
			name:       "StdoutWrite/prints",
			funcName:   "PrintSummary",
			doc:        "PrintSummary prints the summary to stdout.",
			effectType: taxonomy.StdoutWrite,
			wantWeight: -15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := makeFuncDeclWithDoc(tt.funcName, tt.doc)
			sig := classify.AnalyzeGodocSignal(fd, tt.effectType)
			if sig.Weight != tt.wantWeight {
				t.Errorf("AnalyzeGodocSignal(%q, %s) weight = %d, want %d",
					tt.funcName, tt.effectType, sig.Weight, tt.wantWeight)
			}
		})
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
	// Reasoning should mention the matched keyword.
	if !strings.Contains(sig.Reasoning, "returns") {
		t.Errorf("reasoning %q should mention keyword 'returns'", sig.Reasoning)
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
