package report

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jflowers/gaze/internal/taxonomy"
	"github.com/santhosh-tekuri/jsonschema/v6"
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
				GazeVersion: "test",
				GoVersion:   "go1.24.0",
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
		`"signature"`, `"gaze_version"`, `"go_version"`,
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
					"gaze_version": "0.1.0",
					"go_version":   "go1.24",
					"duration_ms":  100,
				},
			},
		},
		"quality_summary": map[string]interface{}{
			"total_tests":                    1,
			"average_contract_coverage":      80.0,
			"total_over_specifications":      1,
			"assertion_detection_confidence": 95,
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
