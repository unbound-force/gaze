package apidoc

import (
	"testing"

	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/protocol"
)

func TestAnalyze_NilData(t *testing.T) {
	report, err := Analyze(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report != nil {
		t.Errorf("expected nil report for nil data, got %+v", report)
	}
}

func TestAnalyze_NativeDocCoverage(t *testing.T) {
	data := &AnalyzerData{
		Language: "python",
		DocCoverage: &protocol.DocCoverageResult{
			Symbols: []protocol.SymbolDocStatus{
				{Name: "ProcessData", Package: "mymod", File: "mymod/process.py", Line: 10, Kind: "function", Documented: true},
				{Name: "HandleError", Package: "mymod", File: "mymod/errors.py", Line: 20, Kind: "function", Documented: false},
				{Name: "DataStore", Package: "mymod", File: "mymod/store.py", Line: 5, Kind: "type", Documented: true},
				{Name: "MAX_RETRIES", Package: "mymod", File: "mymod/config.py", Line: 3, Kind: "constant", Documented: false},
			},
		},
	}

	docs := []docscan.DocumentFile{
		{Path: "docs/api.md", Content: "# API\n\nUse `ProcessData` to process.\n"},
	}

	report, err := Analyze(docs, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.TotalSymbols != 4 {
		t.Errorf("TotalSymbols = %d, want 4", report.TotalSymbols)
	}
	if report.DocumentedSymbols != 2 {
		t.Errorf("DocumentedSymbols = %d, want 2", report.DocumentedSymbols)
	}
	if report.CoveragePercent != 50.0 {
		t.Errorf("CoveragePercent = %f, want 50.0", report.CoveragePercent)
	}
	if report.Source != "doc_coverage" {
		t.Errorf("Source = %q, want %q", report.Source, "doc_coverage")
	}
	if len(report.Undocumented) != 2 {
		t.Fatalf("len(Undocumented) = %d, want 2", len(report.Undocumented))
	}

	undocNames := make(map[string]bool)
	for _, u := range report.Undocumented {
		undocNames[u.Name] = true
	}
	if !undocNames["HandleError"] {
		t.Errorf("expected HandleError in Undocumented")
	}
	if !undocNames["MAX_RETRIES"] {
		t.Errorf("expected MAX_RETRIES in Undocumented")
	}
}

func TestAnalyze_HeuristicPath(t *testing.T) {
	data := &AnalyzerData{
		Language: "python",
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "some/path/mymod", File: "mymod/process.py", Line: 10},
			{Name: "HandleError", Package: "some/path/mymod", File: "mymod/errors.py", Line: 20},
			{Name: "Transform", Package: "some/path/mymod", File: "mymod/transform.py", Line: 30},
		},
	}

	docs := []docscan.DocumentFile{
		{Path: "docs/api.md", Content: "# API\n\nUse `ProcessData` to process data.\n\nSee `HandleError` for error handling.\n"},
	}

	report, err := Analyze(docs, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.TotalSymbols != 3 {
		t.Errorf("TotalSymbols = %d, want 3", report.TotalSymbols)
	}
	if report.DocumentedSymbols != 2 {
		t.Errorf("DocumentedSymbols = %d, want 2", report.DocumentedSymbols)
	}
	if report.Source != "heuristic" {
		t.Errorf("Source = %q, want %q", report.Source, "heuristic")
	}
	if len(report.Undocumented) != 1 {
		t.Fatalf("len(Undocumented) = %d, want 1", len(report.Undocumented))
	}
	if report.Undocumented[0].Name != "Transform" {
		t.Errorf("Undocumented[0].Name = %q, want %q", report.Undocumented[0].Name, "Transform")
	}

	// Coverage should be ~66.67%.
	expectedPct := float64(2) / float64(3) * 100.0
	if report.CoveragePercent != expectedPct {
		t.Errorf("CoveragePercent = %f, want %f", report.CoveragePercent, expectedPct)
	}
}

func TestAnalyze_CombinedReport(t *testing.T) {
	data := &AnalyzerData{
		Language: "python",
		Functions: []protocol.AnalyzedFunction{
			{Name: "ProcessData", Package: "mymod", File: "mymod/process.py", Line: 10},
		},
	}

	docs := []docscan.DocumentFile{
		{
			Path: "docs/api.md",
			Content: "# API\n\n" +
				"Use `ProcessData` to process.\n\n" +
				"See also `OldFunction` for legacy support.\n\n" +
				"```ruby\nputs 'hello'\n```\n",
		},
	}

	report, err := Analyze(docs, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Coverage: 1 function, 1 documented.
	if report.TotalSymbols != 1 {
		t.Errorf("TotalSymbols = %d, want 1", report.TotalSymbols)
	}
	if report.DocumentedSymbols != 1 {
		t.Errorf("DocumentedSymbols = %d, want 1", report.DocumentedSymbols)
	}

	// Stale references: OldFunction is not a known symbol.
	if len(report.StaleReferences) != 1 {
		t.Fatalf("len(StaleReferences) = %d, want 1", len(report.StaleReferences))
	}
	if report.StaleReferences[0].Symbol != "OldFunction" {
		t.Errorf("StaleReferences[0].Symbol = %q, want %q", report.StaleReferences[0].Symbol, "OldFunction")
	}

	// Code block issues: ruby != python.
	if len(report.CodeBlockIssues) != 1 {
		t.Fatalf("len(CodeBlockIssues) = %d, want 1", len(report.CodeBlockIssues))
	}
	if report.CodeBlockIssues[0].DeclaredLang != "ruby" {
		t.Errorf("CodeBlockIssues[0].DeclaredLang = %q, want %q", report.CodeBlockIssues[0].DeclaredLang, "ruby")
	}
	if report.CodeBlockIssues[0].ExpectedLang != "python" {
		t.Errorf("CodeBlockIssues[0].ExpectedLang = %q, want %q", report.CodeBlockIssues[0].ExpectedLang, "python")
	}
}

func TestAnalyze_EmptyFunctions(t *testing.T) {
	data := &AnalyzerData{
		Language:  "python",
		Functions: []protocol.AnalyzedFunction{},
	}

	docs := []docscan.DocumentFile{
		{Path: "README.md", Content: "# Project\n\nNothing here.\n"},
	}

	report, err := Analyze(docs, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.TotalSymbols != 0 {
		t.Errorf("TotalSymbols = %d, want 0", report.TotalSymbols)
	}
	if report.DocumentedSymbols != 0 {
		t.Errorf("DocumentedSymbols = %d, want 0", report.DocumentedSymbols)
	}
	if report.CoveragePercent != 0.0 {
		t.Errorf("CoveragePercent = %f, want 0.0", report.CoveragePercent)
	}
	if report.Source != "heuristic" {
		t.Errorf("Source = %q, want %q", report.Source, "heuristic")
	}
	if len(report.Undocumented) != 0 {
		t.Errorf("len(Undocumented) = %d, want 0", len(report.Undocumented))
	}
}

func TestAnalyze_SymbolMapContainsBothForms(t *testing.T) {
	// This test verifies that both unqualified and qualified symbol names
	// are present in the symbol map used by ValidateReferences. A doc
	// reference using the qualified form (e.g., `mymod.ProcessData`)
	// should NOT be flagged as stale.

	t.Run("native path", func(t *testing.T) {
		data := &AnalyzerData{
			Language: "python",
			DocCoverage: &protocol.DocCoverageResult{
				Symbols: []protocol.SymbolDocStatus{
					{Name: "ProcessData", Package: "mymod", Documented: true},
				},
			},
		}

		docs := []docscan.DocumentFile{
			{
				Path:    "docs/api.md",
				Content: "# API\n\nUse `mymod.ProcessData` to process.\n\nAlso `ProcessData` works.\n",
			},
		}

		report, err := Analyze(docs, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Neither unqualified nor qualified form should be stale.
		for _, ref := range report.StaleReferences {
			if ref.Symbol == "mymod.ProcessData" || ref.Symbol == "ProcessData" {
				t.Errorf("unexpected stale reference for %q — both forms should be in symbol map", ref.Symbol)
			}
		}
	})

	t.Run("heuristic path", func(t *testing.T) {
		data := &AnalyzerData{
			Language: "python",
			Functions: []protocol.AnalyzedFunction{
				{Name: "ProcessData", Package: "some/path/math_utils", File: "math_utils/process.py", Line: 10},
			},
		}

		docs := []docscan.DocumentFile{
			{
				Path:    "docs/api.md",
				Content: "# API\n\nUse `math_utils.ProcessData` to process.\n\nAlso `ProcessData` works.\n",
			},
		}

		report, err := Analyze(docs, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Neither unqualified nor qualified form should be stale.
		for _, ref := range report.StaleReferences {
			if ref.Symbol == "math_utils.ProcessData" || ref.Symbol == "ProcessData" {
				t.Errorf("unexpected stale reference for %q — both forms should be in symbol map", ref.Symbol)
			}
		}
	})
}
