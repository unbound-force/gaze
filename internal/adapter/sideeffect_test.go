package adapter

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/protocol"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TestConvertAnalysisResults_UniversalType verifies that a universal
// taxonomy type (GeneratorYield) is converted without a warning and
// receives the correct tier assignment (P1).
func TestConvertAnalysisResults_UniversalType(t *testing.T) {
	funcs := []protocol.AnalyzedFunction{
		{
			Name:    "generate_values",
			Package: "generators",
			File:    "generators/gen.py",
			Line:    5,
			SideEffects: []protocol.AnalyzedSideEffect{
				{
					Type:        "GeneratorYield",
					Description: "yields computed value",
					Location:    "generators/gen.py:10:5",
					Target:      "value",
				},
			},
		},
	}

	var stderr bytes.Buffer
	results := convertAnalysisResults(funcs, &stderr)

	// No warning should be logged for a known universal type.
	if stderr.Len() != 0 {
		t.Errorf("unexpected warning for known universal type: %s", stderr.String())
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if len(results[0].SideEffects) != 1 {
		t.Fatalf("got %d effects, want 1", len(results[0].SideEffects))
	}

	effect := results[0].SideEffects[0]
	if effect.Type != taxonomy.GeneratorYield {
		t.Errorf("effect type = %q, want %q", effect.Type, taxonomy.GeneratorYield)
	}
	if effect.Tier != taxonomy.TierP1 {
		t.Errorf("effect tier = %q, want %q", effect.Tier, taxonomy.TierP1)
	}
}

// TestConvertAnalysisResults_UnknownTypeWarning verifies that a
// truly unknown type produces a warning (regression test for the
// isKnownP4 → IsKnownType migration).
func TestConvertAnalysisResults_UnknownTypeWarning(t *testing.T) {
	funcs := []protocol.AnalyzedFunction{
		{
			Name:    "do_something",
			Package: "pkg",
			File:    "pkg/mod.py",
			Line:    1,
			SideEffects: []protocol.AnalyzedSideEffect{
				{
					Type:        "CompletelyMadeUpEffect",
					Description: "not a real type",
				},
			},
		},
	}

	var stderr bytes.Buffer
	results := convertAnalysisResults(funcs, &stderr)

	// A warning should be logged for the unknown type.
	if !strings.Contains(stderr.String(), "CompletelyMadeUpEffect") {
		t.Errorf("expected warning about unknown type, got: %q", stderr.String())
	}

	// The effect should still be included with P4 default tier.
	if len(results) != 1 || len(results[0].SideEffects) != 1 {
		t.Fatalf("expected 1 result with 1 effect, got %d results", len(results))
	}
	if results[0].SideEffects[0].Tier != taxonomy.TierP4 {
		t.Errorf("unknown type tier = %q, want %q", results[0].SideEffects[0].Tier, taxonomy.TierP4)
	}
}

// TestConvertAnalysisResults_DetailPassthrough verifies that the
// Detail metadata map from the protocol response is preserved
// through conversion to taxonomy.SideEffect.
func TestConvertAnalysisResults_DetailPassthrough(t *testing.T) {
	detail := map[string]any{
		"language_type": "RaiseException",
		"confidence":    0.95,
		"framework":     "django",
	}

	funcs := []protocol.AnalyzedFunction{
		{
			Name:    "handle_request",
			Package: "views",
			File:    "views/api.py",
			Line:    42,
			SideEffects: []protocol.AnalyzedSideEffect{
				{
					Type:        "ErrorSignal",
					Description: "raises HTTP 404",
					Location:    "views/api.py:50:9",
					Target:      "Http404",
					Detail:      detail,
				},
			},
		},
	}

	var stderr bytes.Buffer
	results := convertAnalysisResults(funcs, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("unexpected warning: %s", stderr.String())
	}

	if len(results) != 1 || len(results[0].SideEffects) != 1 {
		t.Fatalf("expected 1 result with 1 effect")
	}

	effect := results[0].SideEffects[0]

	// Verify Detail is non-nil and contains expected keys.
	if effect.Detail == nil {
		t.Fatal("effect.Detail is nil, want non-nil")
	}

	// Verify string value.
	langType, ok := effect.Detail["language_type"]
	if !ok {
		t.Fatal("Detail missing key \"language_type\"")
	}
	if langType != "RaiseException" {
		t.Errorf("Detail[\"language_type\"] = %v, want %q", langType, "RaiseException")
	}

	// Verify numeric value (float64 in Go's map[string]any).
	conf, ok := effect.Detail["confidence"]
	if !ok {
		t.Fatal("Detail missing key \"confidence\"")
	}
	confFloat, ok := conf.(float64)
	if !ok {
		t.Fatalf("Detail[\"confidence\"] type = %T, want float64", conf)
	}
	if confFloat != 0.95 {
		t.Errorf("Detail[\"confidence\"] = %g, want 0.95", confFloat)
	}

	// Verify third key.
	fw, ok := effect.Detail["framework"]
	if !ok {
		t.Fatal("Detail missing key \"framework\"")
	}
	if fw != "django" {
		t.Errorf("Detail[\"framework\"] = %v, want %q", fw, "django")
	}
}

// TestConvertAnalysisResults_NilDetail verifies that a nil Detail
// map is preserved as nil (not converted to an empty map).
func TestConvertAnalysisResults_NilDetail(t *testing.T) {
	funcs := []protocol.AnalyzedFunction{
		{
			Name:    "simple_func",
			Package: "pkg",
			File:    "pkg/mod.py",
			Line:    1,
			SideEffects: []protocol.AnalyzedSideEffect{
				{
					Type:        "ReturnValue",
					Description: "returns result",
					// Detail intentionally omitted (nil).
				},
			},
		},
	}

	results := convertAnalysisResults(funcs, nil)

	if len(results) != 1 || len(results[0].SideEffects) != 1 {
		t.Fatalf("expected 1 result with 1 effect")
	}
	if results[0].SideEffects[0].Detail != nil {
		t.Errorf("Detail = %v, want nil", results[0].SideEffects[0].Detail)
	}
}

// TestParseSideEffectStream_ValidJSONL verifies that valid JSONL
// input produces the expected AnalyzedFunction slice.
func TestParseSideEffectStream_ValidJSONL(t *testing.T) {
	input := `{"package":"math_utils","name":"divide","file":"math.py","line":1,"side_effects":[{"type":"ErrorReturn","description":"division error"}]}
{"package":"math_utils","name":"multiply","file":"math.py","line":10,"side_effects":[]}
{"package":"strings","name":"trim","file":"str.py","line":5,"side_effects":[{"type":"ReturnValue","description":"trimmed string"}]}
`
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	funcs, err := parseSideEffectStream(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(funcs) != 3 {
		t.Fatalf("got %d functions, want 3", len(funcs))
	}

	// Verify package and name fields for each function.
	want := []struct{ pkg, name string }{
		{"math_utils", "divide"},
		{"math_utils", "multiply"},
		{"strings", "trim"},
	}
	for i, w := range want {
		if funcs[i].Package != w.pkg {
			t.Errorf("funcs[%d].Package = %q, want %q", i, funcs[i].Package, w.pkg)
		}
		if funcs[i].Name != w.name {
			t.Errorf("funcs[%d].Name = %q, want %q", i, funcs[i].Name, w.name)
		}
	}
}

// TestParseSideEffectStream_EmptyStream verifies that an empty
// reader produces no results and no error.
func TestParseSideEffectStream_EmptyStream(t *testing.T) {
	scanner := bufio.NewScanner(bytes.NewReader([]byte{}))
	funcs, err := parseSideEffectStream(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(funcs) != 0 {
		t.Errorf("got %d functions, want 0", len(funcs))
	}
}

// TestParseSideEffectStream_EmptyLinesSkipped verifies that empty
// lines interspersed with valid JSONL are skipped.
func TestParseSideEffectStream_EmptyLinesSkipped(t *testing.T) {
	input := "\n\n" + `{"package":"a","name":"b","file":"a.py","line":1}` + "\n\n"
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	funcs, err := parseSideEffectStream(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(funcs) != 1 {
		t.Fatalf("got %d functions, want 1", len(funcs))
	}
	if funcs[0].Package != "a" {
		t.Errorf("Package = %q, want %q", funcs[0].Package, "a")
	}
	if funcs[0].Name != "b" {
		t.Errorf("Name = %q, want %q", funcs[0].Name, "b")
	}
}

// TestParseSideEffectStream_MalformedJSON verifies that a malformed
// JSON line produces a fail-fast error with line number context.
func TestParseSideEffectStream_MalformedJSON(t *testing.T) {
	input := `{"package":"ok","name":"f1","file":"a.py","line":1}
{bad
`
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	_, err := parseSideEffectStream(scanner)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "malformed JSONL on line 2") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "malformed JSONL on line 2")
	}
}

// TestParseSideEffectStream_LongLineTruncated verifies that a
// malformed JSON line exceeding 200 bytes is truncated in the
// error message.
func TestParseSideEffectStream_LongLineTruncated(t *testing.T) {
	// Build a malformed JSON line longer than 200 bytes.
	long := "{" + strings.Repeat("x", 250)
	scanner := bufio.NewScanner(bytes.NewReader([]byte(long + "\n")))
	_, err := parseSideEffectStream(scanner)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "...") {
		t.Errorf("error = %q, want it to contain truncation marker \"...\"", err.Error())
	}
}

// errReader is an io.Reader that returns io.ErrUnexpectedEOF after
// delivering its initial data.
type errReader struct {
	data []byte
	pos  int
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestParseSideEffectStream_ScannerError verifies that an underlying
// reader error is surfaced with the expected error message.
func TestParseSideEffectStream_ScannerError(t *testing.T) {
	// Provide a valid first line so the scanner consumes it, then
	// fail with io.ErrUnexpectedEOF on the next read. The valid
	// line must be followed by a newline so the scanner yields it
	// successfully before attempting another read.
	validLine := `{"package":"a","name":"b","file":"a.py","line":1}` + "\n"
	reader := &errReader{data: []byte(validLine)}
	scanner := bufio.NewScanner(reader)
	_, err := parseSideEffectStream(scanner)
	if err == nil {
		t.Fatal("expected error from scanner, got nil")
	}
	if !strings.Contains(err.Error(), "reading analyze/stream response") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "reading analyze/stream response")
	}
}
