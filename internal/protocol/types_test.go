package protocol_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/protocol"
)

// TestRoundTrip_InitializeResult verifies JSON marshal/unmarshal
// round-trip for InitializeResult.
func TestRoundTrip_InitializeResult(t *testing.T) {
	original := protocol.InitializeResult{
		Capabilities: protocol.Capabilities{
			Discover:        true,
			TestMapping:     true,
			ClassifySignals: false,
		},
		ProtocolVersion: "1.1.0",
		AnalyzerName:    "snake-eyes",
		Language:        "python",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.AnalyzerName != original.AnalyzerName {
		t.Errorf("analyzer_name = %q, want %q", decoded.AnalyzerName, original.AnalyzerName)
	}
	if decoded.Language != original.Language {
		t.Errorf("language = %q, want %q", decoded.Language, original.Language)
	}
	if decoded.ProtocolVersion != original.ProtocolVersion {
		t.Errorf("protocol_version = %q, want %q", decoded.ProtocolVersion, original.ProtocolVersion)
	}
	if decoded.Capabilities.Discover != original.Capabilities.Discover {
		t.Errorf("capabilities.discover = %v, want %v", decoded.Capabilities.Discover, original.Capabilities.Discover)
	}
	if decoded.Capabilities.TestMapping != original.Capabilities.TestMapping {
		t.Errorf("capabilities.test_mapping = %v, want %v", decoded.Capabilities.TestMapping, original.Capabilities.TestMapping)
	}
	if decoded.Capabilities.ClassifySignals != original.Capabilities.ClassifySignals {
		t.Errorf("capabilities.classify_signals = %v, want %v", decoded.Capabilities.ClassifySignals, original.Capabilities.ClassifySignals)
	}
}

// TestRoundTrip_AnalyzeResult verifies JSON marshal/unmarshal
// round-trip for AnalyzeResult.
func TestRoundTrip_AnalyzeResult(t *testing.T) {
	original := protocol.AnalyzeResult{
		Functions: []protocol.AnalyzedFunction{
			{
				Name:    "divide",
				Package: "math_utils",
				File:    "math_utils/ops.py",
				Line:    20,
				SideEffects: []protocol.AnalyzedSideEffect{
					{
						Type:        "ReturnValue",
						Description: "returns division result",
						Location:    "math_utils/ops.py:25:5",
						Target:      "result",
						Classification: &protocol.AnalyzedClassification{
							Label:      "contractual",
							Confidence: 90,
						},
					},
					{
						Type:        "ErrorReturn",
						Description: "raises ZeroDivisionError",
						Location:    "math_utils/ops.py:22:9",
						Target:      "ZeroDivisionError",
						Classification: &protocol.AnalyzedClassification{
							Label:      "contractual",
							Confidence: 85,
						},
					},
				},
			},
			{
				Name:        "add",
				Package:     "math_utils",
				File:        "math_utils/ops.py",
				Line:        1,
				SideEffects: []protocol.AnalyzedSideEffect{},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.AnalyzeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Functions) != len(original.Functions) {
		t.Fatalf("functions count = %d, want %d", len(decoded.Functions), len(original.Functions))
	}

	// Verify divide function.
	divide := decoded.Functions[0]
	if divide.Name != "divide" {
		t.Errorf("functions[0].name = %q, want %q", divide.Name, "divide")
	}
	if divide.Package != "math_utils" {
		t.Errorf("functions[0].package = %q, want %q", divide.Package, "math_utils")
	}
	if len(divide.SideEffects) != 2 {
		t.Fatalf("functions[0].side_effects count = %d, want 2", len(divide.SideEffects))
	}
	if divide.SideEffects[0].Type != "ReturnValue" {
		t.Errorf("side_effects[0].type = %q, want %q", divide.SideEffects[0].Type, "ReturnValue")
	}
	if divide.SideEffects[0].Classification == nil {
		t.Fatal("side_effects[0].classification is nil, want non-nil")
	}
	if divide.SideEffects[0].Classification.Label != "contractual" {
		t.Errorf("classification.label = %q, want %q", divide.SideEffects[0].Classification.Label, "contractual")
	}
	if divide.SideEffects[0].Classification.Confidence != 90 {
		t.Errorf("classification.confidence = %d, want 90", divide.SideEffects[0].Classification.Confidence)
	}

	// Verify add function has empty side effects.
	add := decoded.Functions[1]
	if add.Name != "add" {
		t.Errorf("functions[1].name = %q, want %q", add.Name, "add")
	}
	if len(add.SideEffects) != 0 {
		t.Errorf("functions[1].side_effects count = %d, want 0", len(add.SideEffects))
	}
}

// TestRoundTrip_ComplexityResult verifies JSON marshal/unmarshal
// round-trip for ComplexityResult.
func TestRoundTrip_ComplexityResult(t *testing.T) {
	original := protocol.ComplexityResult{
		Functions: []protocol.FunctionComplexityData{
			{Name: "add", Package: "math_utils", File: "ops.py", Line: 1, Complexity: 2},
			{Name: "multiply", Package: "math_utils", File: "ops.py", Line: 10, Complexity: 3},
			{Name: "divide", Package: "math_utils", File: "ops.py", Line: 20, Complexity: 5},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.ComplexityResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Functions) != 3 {
		t.Fatalf("functions count = %d, want 3", len(decoded.Functions))
	}
	for i, f := range decoded.Functions {
		orig := original.Functions[i]
		if f.Name != orig.Name {
			t.Errorf("functions[%d].name = %q, want %q", i, f.Name, orig.Name)
		}
		if f.Complexity != orig.Complexity {
			t.Errorf("functions[%d].complexity = %d, want %d", i, f.Complexity, orig.Complexity)
		}
		if f.Line != orig.Line {
			t.Errorf("functions[%d].line = %d, want %d", i, f.Line, orig.Line)
		}
	}
}

// TestRoundTrip_CoverageResult verifies JSON marshal/unmarshal
// round-trip for CoverageResult.
func TestRoundTrip_CoverageResult(t *testing.T) {
	original := protocol.CoverageResult{
		Functions: []protocol.FunctionCoverageData{
			{File: "ops.py", Function: "add", StartLine: 1, EndLine: 3, CoveredStmts: 9, TotalStmts: 10, Percentage: 90.0},
			{File: "ops.py", Function: "multiply", StartLine: 10, EndLine: 15, CoveredStmts: 6, TotalStmts: 10, Percentage: 60.0},
			{File: "ops.py", Function: "divide", StartLine: 20, EndLine: 30, CoveredStmts: 0, TotalStmts: 10, Percentage: 0.0},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.CoverageResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Functions) != 3 {
		t.Fatalf("functions count = %d, want 3", len(decoded.Functions))
	}
	for i, f := range decoded.Functions {
		orig := original.Functions[i]
		if f.Function != orig.Function {
			t.Errorf("functions[%d].function = %q, want %q", i, f.Function, orig.Function)
		}
		if f.Percentage != orig.Percentage {
			t.Errorf("functions[%d].percentage = %.1f, want %.1f", i, f.Percentage, orig.Percentage)
		}
		if f.CoveredStmts != orig.CoveredStmts {
			t.Errorf("functions[%d].covered_stmts = %d, want %d", i, f.CoveredStmts, orig.CoveredStmts)
		}
		if f.TotalStmts != orig.TotalStmts {
			t.Errorf("functions[%d].total_stmts = %d, want %d", i, f.TotalStmts, orig.TotalStmts)
		}
	}
}

// TestRoundTrip_DiscoverResult verifies JSON marshal/unmarshal
// round-trip for DiscoverResult.
func TestRoundTrip_DiscoverResult(t *testing.T) {
	original := protocol.DiscoverResult{
		SourceFiles: []string{"src/main.py", "src/utils.py"},
		TestFiles:   []string{"tests/test_main.py"},
		Framework:   "pytest",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.DiscoverResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.SourceFiles) != 2 {
		t.Errorf("source_files count = %d, want 2", len(decoded.SourceFiles))
	}
	if len(decoded.TestFiles) != 1 {
		t.Errorf("test_files count = %d, want 1", len(decoded.TestFiles))
	}
	if decoded.Framework != "pytest" {
		t.Errorf("framework = %q, want %q", decoded.Framework, "pytest")
	}
}

// TestRoundTrip_TestMappingResult verifies JSON marshal/unmarshal
// round-trip for TestMappingResult.
func TestRoundTrip_TestMappingResult(t *testing.T) {
	original := protocol.TestMappingResult{
		Mappings: []protocol.AssertionMappingData{
			{
				TestFunction:      "test_multiply",
				TestFile:          "tests/test_ops.py",
				AssertionLocation: "tests/test_ops.py:10",
				AssertionType:     "equality",
				TargetFunction:    "multiply",
				TargetPackage:     "math_utils",
				SideEffectType:    "ReturnValue",
				Confidence:        80,
			},
			{
				TestFunction:      "test_divide_error",
				TestFile:          "tests/test_ops.py",
				AssertionLocation: "tests/test_ops.py:20",
				AssertionType:     "error_check",
				TargetFunction:    "divide",
				TargetPackage:     "math_utils",
				SideEffectType:    "ErrorReturn",
				Confidence:        75,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.TestMappingResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Mappings) != 2 {
		t.Fatalf("mappings count = %d, want 2", len(decoded.Mappings))
	}
	for i, m := range decoded.Mappings {
		orig := original.Mappings[i]
		if m.TestFunction != orig.TestFunction {
			t.Errorf("mappings[%d].test_function = %q, want %q", i, m.TestFunction, orig.TestFunction)
		}
		if m.TargetFunction != orig.TargetFunction {
			t.Errorf("mappings[%d].target_function = %q, want %q", i, m.TargetFunction, orig.TargetFunction)
		}
		if m.SideEffectType != orig.SideEffectType {
			t.Errorf("mappings[%d].side_effect_type = %q, want %q", i, m.SideEffectType, orig.SideEffectType)
		}
		if m.Confidence != orig.Confidence {
			t.Errorf("mappings[%d].confidence = %d, want %d", i, m.Confidence, orig.Confidence)
		}
	}
}

// TestRoundTrip_ClassifySignalsResult verifies JSON marshal/unmarshal
// round-trip for ClassifySignalsResult.
func TestRoundTrip_ClassifySignalsResult(t *testing.T) {
	original := protocol.ClassifySignalsResult{
		Signals: []protocol.ClassifySignalData{
			{
				Function:       "divide",
				Package:        "math_utils",
				SideEffectType: "ErrorReturn",
				Source:         "docstring",
				Weight:         15,
				Reasoning:      "docstring mentions ZeroDivisionError",
			},
			{
				Function:       "multiply",
				Package:        "math_utils",
				SideEffectType: "ReturnValue",
				Source:         "type_annotation",
				Weight:         10,
				Reasoning:      "return type annotation present",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded protocol.ClassifySignalsResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Signals) != 2 {
		t.Fatalf("signals count = %d, want 2", len(decoded.Signals))
	}
	for i, s := range decoded.Signals {
		orig := original.Signals[i]
		if s.Function != orig.Function {
			t.Errorf("signals[%d].function = %q, want %q", i, s.Function, orig.Function)
		}
		if s.Source != orig.Source {
			t.Errorf("signals[%d].source = %q, want %q", i, s.Source, orig.Source)
		}
		if s.Weight != orig.Weight {
			t.Errorf("signals[%d].weight = %d, want %d", i, s.Weight, orig.Weight)
		}
		if s.Reasoning != orig.Reasoning {
			t.Errorf("signals[%d].reasoning = %q, want %q", i, s.Reasoning, orig.Reasoning)
		}
	}
}

// TestRoundTrip_Request verifies JSON marshal/unmarshal round-trip
// for the Request type.
func TestRoundTrip_Request(t *testing.T) {
	original := protocol.Request{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "analyze",
		Params: protocol.AnalyzeParams{
			RootPath: "/tmp/project",
			Patterns: []string{"./..."},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the JSON contains expected fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["jsonrpc"]; !ok {
		t.Error("JSON missing 'jsonrpc' field")
	}
	if _, ok := raw["id"]; !ok {
		t.Error("JSON missing 'id' field")
	}
	if _, ok := raw["method"]; !ok {
		t.Error("JSON missing 'method' field")
	}
	if _, ok := raw["params"]; !ok {
		t.Error("JSON missing 'params' field")
	}
}

// TestRoundTrip_Response verifies JSON marshal/unmarshal round-trip
// for the Response type with both success and error cases.
func TestRoundTrip_Response(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result, _ := json.Marshal(map[string]string{"status": "ok"})
		original := protocol.Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  result,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded protocol.Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID != 1 {
			t.Errorf("id = %d, want 1", decoded.ID)
		}
		if decoded.Error != nil {
			t.Errorf("error = %v, want nil", decoded.Error)
		}
		if decoded.Result == nil {
			t.Fatal("result is nil, want non-nil")
		}
	})

	t.Run("error", func(t *testing.T) {
		original := protocol.Response{
			JSONRPC: "2.0",
			ID:      2,
			Error: &protocol.Error{
				Code:    -32600,
				Message: "Invalid request",
				Data:    "missing method field",
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded protocol.Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID != 2 {
			t.Errorf("id = %d, want 2", decoded.ID)
		}
		if decoded.Error == nil {
			t.Fatal("error is nil, want non-nil")
		}
		if decoded.Error.Code != -32600 {
			t.Errorf("error.code = %d, want -32600", decoded.Error.Code)
		}
		if decoded.Error.Message != "Invalid request" {
			t.Errorf("error.message = %q, want %q", decoded.Error.Message, "Invalid request")
		}
	})
}

// TestRoundTrip_DocCoverageResult verifies JSON marshal/unmarshal
// round-trip for DocCoverageResult, Capabilities with DocCoverage,
// and SymbolDocStatus omitempty behavior.
func TestRoundTrip_DocCoverageResult(t *testing.T) {
	t.Run("full_result", func(t *testing.T) {
		original := protocol.DocCoverageResult{
			Symbols: []protocol.SymbolDocStatus{
				{
					Name:       "Calculator",
					Package:    "math_utils",
					File:       "math_utils/calc.py",
					Line:       10,
					Kind:       "class",
					Documented: true,
					DocSnippet: "A simple calculator class.",
				},
				{
					Name:       "add",
					Package:    "math_utils",
					File:       "math_utils/calc.py",
					Line:       25,
					Kind:       "method",
					Documented: true,
					DocSnippet: "Add two numbers.",
				},
				{
					Name:       "MAX_RETRIES",
					Package:    "math_utils",
					File:       "math_utils/constants.py",
					Line:       3,
					Kind:       "constant",
					Documented: false,
					DocSnippet: "",
				},
				{
					Name:       "divide",
					Package:    "math_utils",
					File:       "math_utils/calc.py",
					Line:       40,
					Kind:       "function",
					Documented: true,
					DocSnippet: "Divide a by b.",
				},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded protocol.DocCoverageResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if len(decoded.Symbols) != len(original.Symbols) {
			t.Fatalf("symbols count = %d, want %d", len(decoded.Symbols), len(original.Symbols))
		}

		for i, s := range decoded.Symbols {
			orig := original.Symbols[i]
			if s.Name != orig.Name {
				t.Errorf("symbols[%d].name = %q, want %q", i, s.Name, orig.Name)
			}
			if s.Package != orig.Package {
				t.Errorf("symbols[%d].package = %q, want %q", i, s.Package, orig.Package)
			}
			if s.File != orig.File {
				t.Errorf("symbols[%d].file = %q, want %q", i, s.File, orig.File)
			}
			if s.Line != orig.Line {
				t.Errorf("symbols[%d].line = %d, want %d", i, s.Line, orig.Line)
			}
			if s.Kind != orig.Kind {
				t.Errorf("symbols[%d].kind = %q, want %q", i, s.Kind, orig.Kind)
			}
			if s.Documented != orig.Documented {
				t.Errorf("symbols[%d].documented = %v, want %v", i, s.Documented, orig.Documented)
			}
			if s.DocSnippet != orig.DocSnippet {
				t.Errorf("symbols[%d].doc_snippet = %q, want %q", i, s.DocSnippet, orig.DocSnippet)
			}
		}
	})

	t.Run("capabilities_doc_coverage", func(t *testing.T) {
		original := protocol.Capabilities{
			Discover:        true,
			TestMapping:     false,
			ClassifySignals: false,
			Streaming:       false,
			DocCoverage:     true,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded protocol.Capabilities
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.DocCoverage != true {
			t.Errorf("doc_coverage = %v, want true", decoded.DocCoverage)
		}
		if decoded.Discover != true {
			t.Errorf("discover = %v, want true", decoded.Discover)
		}
		if decoded.TestMapping != false {
			t.Errorf("test_mapping = %v, want false", decoded.TestMapping)
		}
	})

	t.Run("omitempty_doc_snippet", func(t *testing.T) {
		// Undocumented symbol — DocSnippet should be omitted from JSON.
		undocumented := protocol.SymbolDocStatus{
			Name:       "helper",
			Package:    "utils",
			File:       "utils/helper.py",
			Line:       1,
			Kind:       "function",
			Documented: false,
			DocSnippet: "",
		}

		data, err := json.Marshal(undocumented)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		jsonStr := string(data)
		if strings.Contains(jsonStr, "doc_snippet") {
			t.Errorf("JSON contains 'doc_snippet' for undocumented symbol, want omitted; got %s", jsonStr)
		}

		// Documented symbol — DocSnippet should be present.
		documented := protocol.SymbolDocStatus{
			Name:       "Calculator",
			Package:    "utils",
			File:       "utils/calc.py",
			Line:       5,
			Kind:       "class",
			Documented: true,
			DocSnippet: "A calculator.",
		}

		data, err = json.Marshal(documented)
		if err != nil {
			t.Fatalf("marshal documented: %v", err)
		}

		jsonStr = string(data)
		if !strings.Contains(jsonStr, "doc_snippet") {
			t.Errorf("JSON missing 'doc_snippet' for documented symbol; got %s", jsonStr)
		}

		// Round-trip the documented symbol.
		var decoded protocol.SymbolDocStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.DocSnippet != "A calculator." {
			t.Errorf("doc_snippet = %q, want %q", decoded.DocSnippet, "A calculator.")
		}
	})
}
