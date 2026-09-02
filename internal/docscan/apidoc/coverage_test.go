package apidoc

import (
	"testing"

	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/protocol"
)

func TestComputeCoverage_NativePath(t *testing.T) {
	data := &AnalyzerData{
		DocCoverage: &protocol.DocCoverageResult{
			Symbols: []protocol.SymbolDocStatus{
				{Name: "ProcessData", Package: "pkg", File: "pkg/process.go", Line: 10, Kind: "function", Documented: true},
				{Name: "HandleError", Package: "pkg", File: "pkg/errors.go", Line: 20, Kind: "function", Documented: false},
				{Name: "DataStore", Package: "pkg", File: "pkg/store.go", Line: 5, Kind: "type", Documented: true},
				{Name: "MaxRetries", Package: "pkg", File: "pkg/config.go", Line: 3, Kind: "constant", Documented: false},
			},
		},
	}

	result, err := ComputeCoverage(data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 4 {
		t.Errorf("Total = %d, want 4", result.Total)
	}
	if result.Documented != 2 {
		t.Errorf("Documented = %d, want 2", result.Documented)
	}
	if result.Source != "doc_coverage" {
		t.Errorf("Source = %q, want %q", result.Source, "doc_coverage")
	}
	if len(result.Undocumented) != 2 {
		t.Fatalf("len(Undocumented) = %d, want 2", len(result.Undocumented))
	}

	// Verify undocumented entries contain the right symbols.
	undocNames := make(map[string]bool)
	for _, u := range result.Undocumented {
		undocNames[u.Name] = true
	}
	if !undocNames["HandleError"] {
		t.Errorf("expected HandleError in Undocumented")
	}
	if !undocNames["MaxRetries"] {
		t.Errorf("expected MaxRetries in Undocumented")
	}

	// Verify undocumented entry fields are populated.
	for _, u := range result.Undocumented {
		if u.Name == "HandleError" {
			if u.Package != "pkg" {
				t.Errorf("HandleError Package = %q, want %q", u.Package, "pkg")
			}
			if u.File != "pkg/errors.go" {
				t.Errorf("HandleError File = %q, want %q", u.File, "pkg/errors.go")
			}
			if u.Line != 20 {
				t.Errorf("HandleError Line = %d, want 20", u.Line)
			}
			if u.Kind != "function" {
				t.Errorf("HandleError Kind = %q, want %q", u.Kind, "function")
			}
		}
	}
}

func TestComputeCoverage_HeuristicPath(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "pkg", File: "pkg/process.go", Line: 10},
			{Name: "HandleError", Package: "pkg", File: "pkg/errors.go", Line: 20},
			{Name: "Transform", Package: "pkg", File: "pkg/transform.go", Line: 5},
		},
	}

	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "# API\n\nUse `ProcessData` to process data.\nSee also `Transform` for transformations.\n",
		},
	}

	result, err := ComputeCoverage(data, docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if result.Documented != 2 {
		t.Errorf("Documented = %d, want 2", result.Documented)
	}
	if result.Source != "heuristic" {
		t.Errorf("Source = %q, want %q", result.Source, "heuristic")
	}
	if len(result.Undocumented) != 1 {
		t.Fatalf("len(Undocumented) = %d, want 1", len(result.Undocumented))
	}
	if result.Undocumented[0].Name != "HandleError" {
		t.Errorf("Undocumented[0].Name = %q, want %q", result.Undocumented[0].Name, "HandleError")
	}
	if result.Undocumented[0].Kind != "function" {
		t.Errorf("Undocumented[0].Kind = %q, want %q", result.Undocumented[0].Kind, "function")
	}
}

func TestComputeCoverage_NilData(t *testing.T) {
	_, err := ComputeCoverage(nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil data, got nil")
	}
}

func TestComputeCoverage_ZeroSymbols(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{},
	}

	result, err := ComputeCoverage(data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Documented != 0 {
		t.Errorf("Documented = %d, want 0", result.Documented)
	}
	if len(result.Undocumented) != 0 {
		t.Errorf("len(Undocumented) = %d, want 0", len(result.Undocumented))
	}
	if result.Source != "heuristic" {
		t.Errorf("Source = %q, want %q", result.Source, "heuristic")
	}
}

func TestComputeCoverage_CaseSensitive(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "pkg", File: "pkg/process.go", Line: 10},
		},
	}

	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "Use `processData` to process.\n",
		},
	}

	result, err := ComputeCoverage(data, docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Documented != 0 {
		t.Errorf("Documented = %d, want 0 (case-sensitive mismatch)", result.Documented)
	}
	if len(result.Undocumented) != 1 {
		t.Fatalf("len(Undocumented) = %d, want 1", len(result.Undocumented))
	}
	if result.Undocumented[0].Name != "ProcessData" {
		t.Errorf("Undocumented[0].Name = %q, want %q", result.Undocumented[0].Name, "ProcessData")
	}
}

func TestComputeCoverage_QualifiedName(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "pkg", File: "pkg/process.go", Line: 10},
		},
	}

	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "Use `pkg.ProcessData` for processing.\n",
		},
	}

	result, err := ComputeCoverage(data, docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Documented != 1 {
		t.Errorf("Documented = %d, want 1 (qualified name match)", result.Documented)
	}
	if len(result.Undocumented) != 0 {
		t.Errorf("len(Undocumented) = %d, want 0", len(result.Undocumented))
	}
}

func TestComputeCoverage_PartialMatchRejected(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "pkg", File: "pkg/process.go", Line: 10},
		},
	}

	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "Use `Process` for processing.\n",
		},
	}

	result, err := ComputeCoverage(data, docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Documented != 0 {
		t.Errorf("Documented = %d, want 0 (partial match should not count)", result.Documented)
	}
	if len(result.Undocumented) != 1 {
		t.Fatalf("len(Undocumented) = %d, want 1", len(result.Undocumented))
	}
}

func TestComputeCoverage_HeuristicTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		functions      []protocol.AnalyzedFunction
		docContent     string
		wantDocumented int
		wantUndoc      int
	}{
		{
			name: "prose mention not counted",
			functions: []protocol.AnalyzedFunction{
				{Name: "ProcessData", Package: "pkg", File: "f.go", Line: 1},
			},
			docContent:     "ProcessData handles incoming data",
			wantDocumented: 0,
			wantUndoc:      1,
		},
		{
			name: "multiple backtick references",
			functions: []protocol.AnalyzedFunction{
				{Name: "Read", Package: "io", File: "io.go", Line: 1},
				{Name: "Write", Package: "io", File: "io.go", Line: 10},
			},
			docContent:     "Use `Read` and `Write` for I/O.",
			wantDocumented: 2,
			wantUndoc:      0,
		},
		{
			name: "qualified name with nested package",
			functions: []protocol.AnalyzedFunction{
				{Name: "Serve", Package: "github.com/example/http", File: "http.go", Line: 1},
			},
			docContent:     "Call `http.Serve` to start.",
			wantDocumented: 1,
			wantUndoc:      0,
		},
		{
			name: "superstring in backticks not matched",
			functions: []protocol.AnalyzedFunction{
				{Name: "ProcessData", Package: "pkg", File: "f.go", Line: 1},
			},
			docContent:     "See `ProcessDataHandler` for details.",
			wantDocumented: 0,
			wantUndoc:      1,
		},
		{
			name: "empty doc content",
			functions: []protocol.AnalyzedFunction{
				{Name: "Foo", Package: "pkg", File: "f.go", Line: 1},
			},
			docContent:     "",
			wantDocumented: 0,
			wantUndoc:      1,
		},
		{
			name: "code block backticks not matched as symbols",
			functions: []protocol.AnalyzedFunction{
				{Name: "Run", Package: "pkg", File: "f.go", Line: 1},
			},
			docContent:     "```go\nfunc Run() {}\n```\nUse `Run` to start.",
			wantDocumented: 1,
			wantUndoc:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &AnalyzerData{Functions: tt.functions}
			docs := []docscan.DocumentFile{
				{Path: "README.md", Content: tt.docContent},
			}

			result, err := ComputeCoverage(data, docs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Documented != tt.wantDocumented {
				t.Errorf("Documented = %d, want %d", result.Documented, tt.wantDocumented)
			}
			if len(result.Undocumented) != tt.wantUndoc {
				t.Errorf("len(Undocumented) = %d, want %d", len(result.Undocumented), tt.wantUndoc)
			}
		})
	}
}

func TestComputeCoverage_NativePathAllDocumented(t *testing.T) {
	data := &AnalyzerData{
		DocCoverage: &protocol.DocCoverageResult{
			Symbols: []protocol.SymbolDocStatus{
				{Name: "A", Package: "p", File: "a.go", Line: 1, Kind: "function", Documented: true},
				{Name: "B", Package: "p", File: "b.go", Line: 1, Kind: "type", Documented: true},
			},
		},
	}

	result, err := ComputeCoverage(data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Documented != 2 {
		t.Errorf("Documented = %d, want 2", result.Documented)
	}
	if len(result.Undocumented) != 0 {
		t.Errorf("len(Undocumented) = %d, want 0", len(result.Undocumented))
	}
}

func TestComputeCoverage_NativePathEmptySymbols(t *testing.T) {
	data := &AnalyzerData{
		DocCoverage: &protocol.DocCoverageResult{
			Symbols: []protocol.SymbolDocStatus{},
		},
	}

	result, err := ComputeCoverage(data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Documented != 0 {
		t.Errorf("Documented = %d, want 0", result.Documented)
	}
	if result.Source != "doc_coverage" {
		t.Errorf("Source = %q, want %q", result.Source, "doc_coverage")
	}
}

func TestComputeCoverage_MultipleDocFiles(t *testing.T) {
	data := &AnalyzerData{
		Functions: []protocol.AnalyzedFunction{
			{Name: "Alpha", Package: "pkg", File: "f.go", Line: 1},
			{Name: "Beta", Package: "pkg", File: "f.go", Line: 10},
			{Name: "Gamma", Package: "pkg", File: "f.go", Line: 20},
		},
	}

	docs := []docscan.DocumentFile{
		{Path: "README.md", Content: "See `Alpha` for details."},
		{Path: "docs/api.md", Content: "Use `Beta` for processing."},
	}

	result, err := ComputeCoverage(data, docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Documented != 2 {
		t.Errorf("Documented = %d, want 2", result.Documented)
	}
	if len(result.Undocumented) != 1 {
		t.Fatalf("len(Undocumented) = %d, want 1", len(result.Undocumented))
	}
	if result.Undocumented[0].Name != "Gamma" {
		t.Errorf("Undocumented[0].Name = %q, want %q", result.Undocumented[0].Name, "Gamma")
	}
}
