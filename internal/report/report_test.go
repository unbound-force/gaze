package report

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

func sampleResults() []taxonomy.AnalysisResult {
	return []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/store",
				Function:  "Save",
				Receiver:  "*Store",
				Signature: "func (s *Store) Save(item Item) (int64, error)",
				Location:  "store.go:42:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-abc12345",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    "store.go:42:49",
					Description: "returns int64 at position 0",
					Target:      "int64",
				},
				{
					ID:          "se-def67890",
					Type:        taxonomy.ErrorReturn,
					Tier:        taxonomy.TierP0,
					Location:    "store.go:42:56",
					Description: "returns error at position 1",
					Target:      "error",
				},
				{
					ID:          "se-ghi11111",
					Type:        taxonomy.ReceiverMutation,
					Tier:        taxonomy.TierP0,
					Location:    "store.go:55:2",
					Description: "mutates receiver field 'lastSaved'",
					Target:      "lastSaved",
				},
			},
			Metadata: taxonomy.Metadata{
				GazeVersion:     "test",
				Language:        "go",
				LanguageVersion: "go1.24.0",
			},
		},
	}
}

func TestWriteJSON_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, sampleResults(), "0.1.0")
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Must be valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestWriteJSON_HasVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults(), "0.1.0"); err != nil {
		t.Fatal(err)
	}

	var report JSONReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}

	if report.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestWriteJSON_HasResults(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults(), "0.1.0"); err != nil {
		t.Fatal(err)
	}

	var report JSONReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}

	if len(report.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(report.Results))
	}
	if len(report.Results[0].SideEffects) != 3 {
		t.Errorf("expected 3 side effects, got %d",
			len(report.Results[0].SideEffects))
	}
}

func TestWriteJSON_ContainsAllFields(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults(), "0.1.0"); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	requiredFields := []string{
		`"version"`, `"results"`, `"target"`, `"side_effects"`,
		`"id"`, `"type"`, `"tier"`, `"location"`,
		`"description"`, `"package"`, `"function"`,
		`"signature"`, `"gaze_version"`, `"language"`, `"language_version"`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(output, field) {
			t.Errorf("JSON output missing field %s", field)
		}
	}
}

func TestWriteText_HasFunctionName(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleResults()); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "(*Store).Save") {
		t.Error("text output missing function name '(*Store).Save'")
	}
}

func TestWriteText_HasSideEffects(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleResults()); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "ReturnValue") {
		t.Error("text output missing ReturnValue")
	}
	if !strings.Contains(output, "ErrorReturn") {
		t.Error("text output missing ErrorReturn")
	}
	if !strings.Contains(output, "ReceiverMutation") {
		t.Error("text output missing ReceiverMutation")
	}
}

func TestWriteText_HasSummary(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleResults()); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "1 function(s) analyzed") {
		t.Error("text output missing function count summary")
	}
	if !strings.Contains(output, "3 side effect(s) detected") {
		t.Error("text output missing side effect count summary")
	}
}

func TestWriteText_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 function(s) analyzed") {
		t.Error("text output should show 0 functions for empty results")
	}
}

func TestWriteText_NoSideEffects(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Pure",
				Signature: "func Pure()",
				Location:  "pure.go:1:1",
			},
			SideEffects: nil,
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, results); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "No side effects detected") {
		t.Error("expected 'No side effects detected' for pure function")
	}
}

func TestWriteJSON_ValidAgainstSchema(t *testing.T) {
	// Compile the embedded JSON Schema.
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(Schema))
	if err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", sch); err != nil {
		t.Fatalf("failed to add schema resource: %v", err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	// Generate JSON output from sample data.
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Parse and validate against schema.
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("JSON output does not conform to schema:\n%v", err)
	}
}

// stripANSI removes ANSI escape sequences from text for width measurement.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestWriteText_FitsIn80Columns(t *testing.T) {
	// SC-007: Human-readable output fits in an 80-column terminal
	// without horizontal scrolling for typical results.
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleResults()); err != nil {
		t.Fatal(err)
	}

	const maxWidth = 80
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines {
		plain := stripANSI(line)
		width := utf8.RuneCountInString(plain)
		if width > maxWidth {
			t.Errorf("line %d exceeds %d columns (%d runes): %q",
				i+1, maxWidth, width, plain)
		}
	}
}

// sampleClassifiedResults returns sample results with classification
// populated for testing the classify output path.
func sampleClassifiedResults() []taxonomy.AnalysisResult {
	results := sampleResults()
	for i := range results {
		for j := range results[i].SideEffects {
			results[i].SideEffects[j].Classification = &taxonomy.Classification{
				Label:      taxonomy.Contractual,
				Confidence: 85,
				Signals: []taxonomy.Signal{
					{
						Source:    "interface",
						Weight:    30,
						Reasoning: "implements io.Writer",
					},
					{
						Source: "naming",
						Weight: 10,
					},
				},
			}
		}
	}
	return results
}

func TestWriteTextOptions_ClassifyColumn(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTextOptions(&buf, sampleClassifiedResults(), TextOptions{Classify: true})
	if err != nil {
		t.Fatalf("WriteTextOptions failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CLASSIFICATION") {
		t.Error("expected CLASSIFICATION column header in classified text output")
	}
	if !strings.Contains(output, "contractual") {
		t.Error("expected 'contractual' label in classified text output")
	}
	if !strings.Contains(output, "85%") {
		t.Error("expected confidence '85%' in classified text output")
	}
}

func TestWriteTextOptions_VerboseSignalBreakdown(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTextOptions(&buf, sampleClassifiedResults(), TextOptions{
		Classify: true,
		Verbose:  true,
	})
	if err != nil {
		t.Fatalf("WriteTextOptions verbose failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "interface") {
		t.Error("expected signal source 'interface' in verbose output")
	}
	if !strings.Contains(output, "implements io.Writer") {
		t.Error("expected signal reasoning in verbose output")
	}
}

func TestWriteTextOptions_ClassifyFitsIn80Columns(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTextOptions(&buf, sampleClassifiedResults(), TextOptions{Classify: true}); err != nil {
		t.Fatal(err)
	}

	const maxWidth = 80
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines {
		plain := stripANSI(line)
		width := utf8.RuneCountInString(plain)
		if width > maxWidth {
			t.Errorf("classify line %d exceeds %d columns (%d runes): %q",
				i+1, maxWidth, width, plain)
		}
	}
}

func TestWriteJSON_ClassifiedOutput_ValidAgainstSchema(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(Schema))
	if err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", sch); err != nil {
		t.Fatalf("failed to add schema resource: %v", err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleClassifiedResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("classified JSON output does not conform to schema:\n%v", err)
	}
}

func TestWriteJSON_ClassifiedOutput_ContainsClassification(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleClassifiedResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"classification"`) {
		t.Error("classified JSON output missing 'classification' field")
	}
	if !strings.Contains(output, `"contractual"`) {
		t.Error("classified JSON output missing 'contractual' label")
	}
	if !strings.Contains(output, `"confidence"`) {
		t.Error("classified JSON output missing 'confidence' field")
	}
	if !strings.Contains(output, `"signals"`) {
		t.Error("classified JSON output missing 'signals' field")
	}
}

func TestClassificationStyle(_ *testing.T) {
	s := DefaultStyles()

	// Just verify the function returns without panic for all labels.
	labels := []string{"contractual", "incidental", "ambiguous", "unknown", ""}
	for _, label := range labels {
		style := s.ClassificationStyle(label)
		// Render something to ensure no panic.
		_ = style.Render("test")
	}
}

func TestWriteJSON_EmptyResults_ValidAgainstSchema(t *testing.T) {
	// Empty results should also validate.
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(Schema))
	if err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", sch); err != nil {
		t.Fatalf("failed to add schema resource: %v", err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil, "0.1.0"); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("empty JSON output does not conform to schema:\n%v", err)
	}
}

// TestQualitySchema_Compiles verifies the QualitySchema constant is
// valid JSON Schema that can be compiled without errors. This also
// exercises the QualitySchema constant to prevent it from being
// orphaned dead code (Zero-Waste Mandate).
func TestQualitySchema_Compiles(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(QualitySchema))
	if err != nil {
		t.Fatalf("failed to parse QualitySchema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("quality-schema.json", sch); err != nil {
		t.Fatalf("failed to add quality schema resource: %v", err)
	}
	_, err = compiler.Compile("quality-schema.json")
	if err != nil {
		t.Fatalf("failed to compile QualitySchema: %v", err)
	}
}

// TestQualitySchema_ValidatesSampleOutput validates sample quality
// JSON output against the QualitySchema.
func TestQualitySchema_ValidatesSampleOutput(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(QualitySchema))
	if err != nil {
		t.Fatalf("failed to parse QualitySchema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("quality-schema.json", sch); err != nil {
		t.Fatalf("failed to add quality schema resource: %v", err)
	}
	compiled, err := compiler.Compile("quality-schema.json")
	if err != nil {
		t.Fatalf("failed to compile QualitySchema: %v", err)
	}

	// Construct a sample quality report JSON.
	sample := map[string]interface{}{
		"quality_reports": []map[string]interface{}{
			{
				"test_function": "TestFoo",
				"test_location": "foo_test.go:10",
				"target_function": map[string]interface{}{
					"package":   "pkg",
					"function":  "Foo",
					"signature": "func Foo() error",
					"location":  "foo.go:5",
				},
				"contract_coverage": map[string]interface{}{
					"percentage":        80.0,
					"covered_count":     4,
					"total_contractual": 5,
				},
				"over_specification": map[string]interface{}{
					"count": 1,
					"ratio": 0.2,
				},
				"assertion_detection_confidence": 95,
				"metadata": map[string]interface{}{
					"gaze_version":     "0.1.0",
					"language":         "go",
					"language_version": "go1.24",
					"duration_ms":      100,
				},
			},
		},
		"quality_summary": map[string]interface{}{
			"total_tests":                    1,
			"average_contract_coverage":      80.0,
			"total_over_specifications":      1,
			"assertion_detection_confidence": 95,
			"ssa_degraded":                   false,
		},
	}

	sampleJSON, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal sample: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(sampleJSON))
	if err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("sample quality JSON does not conform to QualitySchema:\n%v", err)
	}
}

// TestQualitySchema_ValidatesDegradedOutput validates that quality
// JSON output with ssa_degraded: true conforms to the QualitySchema.
func TestQualitySchema_ValidatesDegradedOutput(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(QualitySchema))
	if err != nil {
		t.Fatalf("failed to parse QualitySchema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("quality-schema.json", sch); err != nil {
		t.Fatalf("failed to add quality schema resource: %v", err)
	}
	compiled, err := compiler.Compile("quality-schema.json")
	if err != nil {
		t.Fatalf("failed to compile QualitySchema: %v", err)
	}

	// Construct a degraded quality report JSON — SSA failed, so
	// target, coverage, and mapping are zero-valued.
	sample := map[string]interface{}{
		"quality_reports": []map[string]interface{}{
			{
				"test_function": "TestFoo",
				"test_location": "foo_test.go:10",
				"target_function": map[string]interface{}{
					"package":   "",
					"function":  "",
					"signature": "",
					"location":  "",
				},
				"contract_coverage": map[string]interface{}{
					"percentage":        0.0,
					"covered_count":     0,
					"total_contractual": 0,
				},
				"over_specification": map[string]interface{}{
					"count": 0,
					"ratio": 0.0,
				},
				"assertion_detection_confidence": 80,
				"metadata": map[string]interface{}{
					"gaze_version":     "0.1.0",
					"language":         "go",
					"language_version": "go1.25",
					"duration_ms":      5,
				},
			},
		},
		"quality_summary": map[string]interface{}{
			"total_tests":                    1,
			"average_contract_coverage":      0.0,
			"total_over_specifications":      0,
			"assertion_detection_confidence": 80,
			"ssa_degraded":                   true,
			"ssa_degraded_packages":          []string{"github.com/example/pkg/chrome"},
		},
	}

	sampleJSON, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal sample: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(sampleJSON))
	if err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("degraded quality JSON does not conform to QualitySchema:\n%v", err)
	}
}

// TestQualitySchema_ValidatesWithoutSSADegradedField verifies that
// JSON output that omits ssa_degraded entirely still validates
// against the schema (backward compatibility).
func TestQualitySchema_ValidatesWithoutSSADegradedField(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(QualitySchema))
	if err != nil {
		t.Fatalf("failed to parse QualitySchema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("quality-schema.json", sch); err != nil {
		t.Fatalf("failed to add quality schema resource: %v", err)
	}
	compiled, err := compiler.Compile("quality-schema.json")
	if err != nil {
		t.Fatalf("failed to compile QualitySchema: %v", err)
	}

	// Construct a sample without ssa_degraded — must still validate
	// since the field is not in the required array.
	sample := map[string]interface{}{
		"quality_reports": []map[string]interface{}{
			{
				"test_function": "TestBar",
				"test_location": "bar_test.go:5",
				"target_function": map[string]interface{}{
					"package":   "pkg",
					"function":  "Bar",
					"signature": "func Bar() int",
					"location":  "bar.go:10",
				},
				"contract_coverage": map[string]interface{}{
					"percentage":        100.0,
					"covered_count":     2,
					"total_contractual": 2,
				},
				"over_specification": map[string]interface{}{
					"count": 0,
					"ratio": 0.0,
				},
				"assertion_detection_confidence": 100,
				"metadata": map[string]interface{}{
					"gaze_version":     "0.1.0",
					"language":         "go",
					"language_version": "go1.24",
					"duration_ms":      50,
				},
			},
		},
		"quality_summary": map[string]interface{}{
			"total_tests":                    1,
			"average_contract_coverage":      100.0,
			"total_over_specifications":      0,
			"assertion_detection_confidence": 100,
			// ssa_degraded intentionally omitted — backward compat
		},
	}

	sampleJSON, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal sample: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(sampleJSON))
	if err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}
	if err := compiled.Validate(inst); err != nil {
		t.Errorf("legacy quality JSON (without ssa_degraded) does not conform to QualitySchema:\n%v", err)
	}
}

// TestQualitySchema_ValidatesClassificationCounts verifies that quality
// JSON carrying a classification_counts object validates, and that JSON
// omitting the field (Go-native output) still validates.
func TestQualitySchema_ValidatesClassificationCounts(t *testing.T) {
	sch, err := jsonschema.UnmarshalJSON(strings.NewReader(QualitySchema))
	if err != nil {
		t.Fatalf("failed to parse QualitySchema JSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("quality-schema.json", sch); err != nil {
		t.Fatalf("failed to add quality schema resource: %v", err)
	}
	compiled, err := compiler.Compile("quality-schema.json")
	if err != nil {
		t.Fatalf("failed to compile QualitySchema: %v", err)
	}

	buildSummary := func(withCounts bool) map[string]interface{} {
		summary := map[string]interface{}{
			"total_tests":                    1,
			"average_contract_coverage":      50.0,
			"total_over_specifications":      0,
			"assertion_detection_confidence": 90,
		}
		if withCounts {
			summary["classification_counts"] = map[string]interface{}{
				"contractual": 563,
				"incidental":  342,
				"ambiguous":   1186,
			}
		}
		return summary
	}

	buildSample := func(withCounts bool) map[string]interface{} {
		return map[string]interface{}{
			"quality_reports": []map[string]interface{}{
				{
					"test_function": "TestFoo",
					"test_location": "foo_test.go:10",
					"target_function": map[string]interface{}{
						"package":   "pkg",
						"function":  "Foo",
						"signature": "func Foo() int",
						"location":  "foo.go:10",
					},
					"contract_coverage": map[string]interface{}{
						"percentage":        50.0,
						"covered_count":     1,
						"total_contractual": 2,
					},
					"over_specification": map[string]interface{}{
						"count": 0,
						"ratio": 0.0,
					},
					"assertion_detection_confidence": 90,
					"metadata": map[string]interface{}{
						"gaze_version":     "0.1.0",
						"language":         "python",
						"language_version": "3.12",
						"duration_ms":      5,
					},
				},
			},
			"quality_summary": buildSummary(withCounts),
		}
	}

	t.Run("classification_counts present validates", func(t *testing.T) {
		sampleJSON, err := json.Marshal(buildSample(true))
		if err != nil {
			t.Fatalf("failed to marshal sample: %v", err)
		}
		inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(sampleJSON))
		if err != nil {
			t.Fatalf("failed to parse sample JSON: %v", err)
		}
		if err := compiled.Validate(inst); err != nil {
			t.Errorf("quality JSON with classification_counts does not conform to QualitySchema:\n%v", err)
		}
	})

	t.Run("classification_counts omitted validates", func(t *testing.T) {
		sampleJSON, err := json.Marshal(buildSample(false))
		if err != nil {
			t.Fatalf("failed to marshal sample: %v", err)
		}
		inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(sampleJSON))
		if err != nil {
			t.Fatalf("failed to parse sample JSON: %v", err)
		}
		if err := compiled.Validate(inst); err != nil {
			t.Errorf("quality JSON without classification_counts does not conform to QualitySchema:\n%v", err)
		}
	})
}

// TestTierStyle_AllTiers verifies that TierStyle returns the correct
// style for each tier string and the default for unknown tiers.
// Lipgloss degrades to no-color in non-TTY mode, so we compare style
// identity via the GetForeground() color value rather than rendered
// output.
func TestTierStyle_AllTiers(t *testing.T) {
	s := DefaultStyles()

	tests := []struct {
		tier string
		want lipgloss.Style
		desc string
	}{
		{"P0", s.TierP0, "TierP0"},
		{"P1", s.TierP1, "TierP1"},
		{"P2", s.TierP2, "TierP2"},
		{"P3", s.TierP3, "TierP3"},
		{"P4", s.TierP4, "TierP4"},
		{"unknown", s.Muted, "Muted (default)"},
		{"", s.Muted, "Muted (empty)"},
	}

	for _, tt := range tests {
		got := s.TierStyle(tt.tier)
		// Compare via foreground color — this works even in
		// no-color mode because the style struct retains the
		// configured color value.
		gotFg := got.GetForeground()
		wantFg := tt.want.GetForeground()
		if gotFg != wantFg {
			t.Errorf("TierStyle(%q) foreground = %v, want %v (%s)",
				tt.tier, gotFg, wantFg, tt.desc)
		}
	}
}

// --- writeEffectRows tests ---

func TestWriteEffectRows_ClassifyMode(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Tier:        taxonomy.TierP0,
			Type:        taxonomy.ReturnValue,
			Description: "returns int64",
			Classification: &taxonomy.Classification{
				Label:      taxonomy.Contractual,
				Confidence: 85,
			},
		},
	}
	rows := writeEffectRows(effects, 26, true)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 4 {
		t.Fatalf("expected 4 columns in classify mode, got %d", len(rows[0]))
	}
	if rows[0][0] != "P0" {
		t.Errorf("tier = %q, want %q", rows[0][0], "P0")
	}
	if rows[0][1] != "ReturnValue" {
		t.Errorf("type = %q, want %q", rows[0][1], "ReturnValue")
	}
	if rows[0][2] != "returns int64" {
		t.Errorf("desc = %q, want %q", rows[0][2], "returns int64")
	}
	if rows[0][3] != "contractual/85%" {
		t.Errorf("class = %q, want %q", rows[0][3], "contractual/85%")
	}
}

func TestWriteEffectRows_NonClassifyMode(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Tier:        taxonomy.TierP1,
			Type:        taxonomy.MapMutation,
			Description: "mutates map",
		},
	}
	rows := writeEffectRows(effects, 42, false)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 3 {
		t.Fatalf("expected 3 columns in non-classify mode, got %d", len(rows[0]))
	}
	if rows[0][0] != "P1" {
		t.Errorf("tier = %q, want %q", rows[0][0], "P1")
	}
	if rows[0][2] != "mutates map" {
		t.Errorf("desc = %q, want %q", rows[0][2], "mutates map")
	}
}

func TestWriteEffectRows_DescriptionTruncation(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Tier:        taxonomy.TierP0,
			Type:        taxonomy.ReturnValue,
			Description: "this is a very long description that exceeds the maximum",
		},
	}
	// maxDesc = 20: should truncate to 17 chars + "..."
	rows := writeEffectRows(effects, 20, false)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	desc := rows[0][2]
	if len(desc) != 20 {
		t.Errorf("truncated desc length = %d, want 20", len(desc))
	}
	if !strings.HasSuffix(desc, "...") {
		t.Errorf("truncated desc should end with '...', got %q", desc)
	}
}

func TestWriteEffectRows_NilClassification(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Tier:           taxonomy.TierP0,
			Type:           taxonomy.ReturnValue,
			Description:    "returns error",
			Classification: nil,
		},
	}
	rows := writeEffectRows(effects, 26, true)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][3] != "—" {
		t.Errorf("nil classification should produce dash cell, got %q", rows[0][3])
	}
}

func TestWriteEffectRows_ClassificationTruncation(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Tier:        taxonomy.TierP0,
			Type:        taxonomy.ReturnValue,
			Description: "returns",
			Classification: &taxonomy.Classification{
				Label:      taxonomy.ClassificationLabel("verylonglabel"),
				Confidence: 100,
			},
		},
	}
	rows := writeEffectRows(effects, 26, true)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	classCell := rows[0][3]
	// "verylonglabel/100%" = 18 chars > maxClassify (16), should truncate
	if len(classCell) > 16 {
		t.Errorf("classification cell should be truncated to 16 chars, got %d: %q", len(classCell), classCell)
	}
	if !strings.HasSuffix(classCell, "...") {
		t.Errorf("truncated classification should end with '...', got %q", classCell)
	}
}

// --- writeVerboseSignals tests ---

func TestWriteVerboseSignals_WithSignals(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Type:     taxonomy.ReturnValue,
			Location: "store.go:42",
			Classification: &taxonomy.Classification{
				Label:      taxonomy.Contractual,
				Confidence: 85,
				Signals: []taxonomy.Signal{
					{
						Source:    "interface",
						Weight:    30,
						Reasoning: "implements io.Writer",
					},
					{
						Source:     "naming",
						Weight:     10,
						SourceFile: "store.go",
						Excerpt:    "Save(item)",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	writeVerboseSignals(&buf, effects)
	output := buf.String()

	if !strings.Contains(output, "Signals for ReturnValue (store.go:42):") {
		t.Error("missing signal header")
	}
	if !strings.Contains(output, "interface: +30") {
		t.Error("missing signal source and weight")
	}
	if !strings.Contains(output, "implements io.Writer") {
		t.Error("missing signal reasoning")
	}
	if !strings.Contains(output, "source: store.go") {
		t.Error("missing signal source file")
	}
	if !strings.Contains(output, `excerpt: "Save(item)"`) {
		t.Error("missing signal excerpt")
	}
}

func TestWriteVerboseSignals_NoSignals(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{
			Type:           taxonomy.ReturnValue,
			Location:       "store.go:42",
			Classification: nil,
		},
		{
			Type:     taxonomy.ErrorReturn,
			Location: "store.go:43",
			Classification: &taxonomy.Classification{
				Label:   taxonomy.Contractual,
				Signals: nil, // no signals
			},
		},
	}

	var buf bytes.Buffer
	writeVerboseSignals(&buf, effects)
	output := buf.String()

	if output != "" {
		t.Errorf("expected empty output for effects without signals, got %q", output)
	}
}
