package report

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// htmlSampleResults returns analysis results with two functions and
// multiple side effects across tiers for comprehensive HTML rendering
// tests.
func htmlSampleResults() []taxonomy.AnalysisResult {
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
					ID:          "se-ret-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    "store.go:42:49",
					Description: "returns int64 at position 0",
				},
				{
					ID:          "se-err-002",
					Type:        taxonomy.ErrorReturn,
					Tier:        taxonomy.TierP0,
					Location:    "store.go:42:56",
					Description: "returns error at position 1",
				},
			},
		},
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/store",
				Function:  "Delete",
				Signature: "func Delete(id string) error",
				Location:  "store.go:80:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-fs-003",
					Type:        taxonomy.FileSystemDelete,
					Tier:        taxonomy.TierP2,
					Location:    "store.go:85:3",
					Description: "removes file from disk",
				},
			},
		},
	}
}

// TestWriteHTML_CompleteDocument verifies that the HTML formatter
// produces a complete HTML document with doctype, html root, head,
// body, and the supplied version in metadata.
func TestWriteHTML_CompleteDocument(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), "1.2.3"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	required := []struct {
		substr string
		desc   string
	}{
		{"<!DOCTYPE html>", "doctype declaration"},
		{"<html lang=\"en\">", "html root element with lang"},
		{"<head>", "head element"},
		{"</head>", "head closing tag"},
		{"<body>", "body element"},
		{"</body>", "body closing tag"},
		{"</html>", "html closing tag"},
		{"<meta charset=\"utf-8\">", "charset meta"},
		{"<title>Gaze Analysis Report</title>", "title element"},
	}

	for _, r := range required {
		if !strings.Contains(output, r.substr) {
			t.Errorf("missing %s: expected %q in output", r.desc, r.substr)
		}
	}
}

// TestWriteHTML_VersionInDocument verifies that the supplied version
// string appears in the document metadata.
func TestWriteHTML_VersionInDocument(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), "1.2.3"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// Version appears in the generator meta tag.
	if !strings.Contains(output, `content="gaze 1.2.3"`) {
		t.Error("expected version in generator meta tag")
	}
	// Version appears in the visible version paragraph.
	if !strings.Contains(output, "gaze 1.2.3") {
		t.Error("expected version in visible version text")
	}
}

// TestWriteHTML_DevVersionFallback verifies that an empty version
// string defaults to "dev".
func TestWriteHTML_DevVersionFallback(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), ""); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, `content="gaze dev"`) {
		t.Error("expected 'gaze dev' in generator meta tag for empty version")
	}
	if !strings.Contains(output, "gaze dev") {
		t.Error("expected 'gaze dev' in visible version text for empty version")
	}
}

// TestWriteHTML_EmptyResults verifies that an empty result set
// produces a valid complete HTML document with a clear empty-state
// message.
func TestWriteHTML_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, nil, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("empty results should still produce a complete HTML document")
	}
	if !strings.Contains(output, "No functions were analyzed.") {
		t.Error("expected empty-state message 'No functions were analyzed.'")
	}
	// Should not contain function sections or summary counts.
	if strings.Contains(output, "<details>") {
		t.Error("empty results should not contain <details> sections")
	}
	if strings.Contains(output, "function(s) analyzed") {
		t.Error("empty results should not contain summary count text")
	}
}

// TestWriteHTML_AllFunctionsAndEffects verifies that every function
// and every side effect from the input appears exactly once in the
// output.
func TestWriteHTML_AllFunctionsAndEffects(t *testing.T) {
	var buf bytes.Buffer
	results := htmlSampleResults()
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// Verify summary counts.
	if !strings.Contains(output, "2 function(s) analyzed") {
		t.Error("expected '2 function(s) analyzed' in summary")
	}
	if !strings.Contains(output, "3 side effect(s) detected") {
		t.Error("expected '3 side effect(s) detected' in summary")
	}

	// Every function name must appear.
	if !strings.Contains(output, "(*Store).Save") {
		t.Error("missing function name '(*Store).Save'")
	}
	if !strings.Contains(output, "Delete") {
		t.Error("missing function name 'Delete'")
	}

	// Every function location must appear.
	if !strings.Contains(output, "store.go:42:1") {
		t.Error("missing function location 'store.go:42:1'")
	}
	if !strings.Contains(output, "store.go:80:1") {
		t.Error("missing function location 'store.go:80:1'")
	}

	// Every side effect type must appear.
	if !strings.Contains(output, "ReturnValue") {
		t.Error("missing side effect type 'ReturnValue'")
	}
	if !strings.Contains(output, "ErrorReturn") {
		t.Error("missing side effect type 'ErrorReturn'")
	}
	if !strings.Contains(output, "FileSystemDelete") {
		t.Error("missing side effect type 'FileSystemDelete'")
	}

	// Every side effect tier must appear.
	if count := strings.Count(output, "P0"); count < 2 {
		t.Errorf("expected at least 2 P0 tier occurrences, got %d", count)
	}
	if !strings.Contains(output, "P2") {
		t.Error("missing tier 'P2'")
	}

	// Every side effect description must appear.
	if !strings.Contains(output, "returns int64 at position 0") {
		t.Error("missing description 'returns int64 at position 0'")
	}
	if !strings.Contains(output, "returns error at position 1") {
		t.Error("missing description 'returns error at position 1'")
	}
	if !strings.Contains(output, "removes file from disk") {
		t.Error("missing description 'removes file from disk'")
	}

	// Every side effect location must appear.
	if !strings.Contains(output, "store.go:42:49") {
		t.Error("missing effect location 'store.go:42:49'")
	}
	if !strings.Contains(output, "store.go:85:3") {
		t.Error("missing effect location 'store.go:85:3'")
	}
}

// TestWriteHTML_ClassificationAndDetailMetadata verifies that
// classification labels, confidence, and detail metadata are rendered
// when present.
func TestWriteHTML_ClassificationAndDetailMetadata(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Process",
				Signature: "func Process() error",
				Location:  "proc.go:10:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-class-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    "proc.go:10:20",
					Description: "returns error",
					Classification: &taxonomy.Classification{
						Label:      taxonomy.Contractual,
						Confidence: 92,
					},
					Detail: map[string]any{
						"alpha": "first",
						"beta":  42,
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// Classification column header must be present.
	if !strings.Contains(output, "Classification") {
		t.Error("expected 'Classification' column header")
	}
	// Classification value must appear.
	if !strings.Contains(output, "contractual (92%)") {
		t.Error("expected 'contractual (92%)' classification value")
	}

	// Detail column header must be present.
	if !strings.Contains(output, "Detail") {
		t.Error("expected 'Detail' column header")
	}
	// Detail must be JSON-serialized with sorted keys. html/template
	// contextually escapes quotes to &#34; -- the escaped form is the
	// only correct output because Detail is a plain string field, not
	// a trusted template.HTML value.
	if !strings.Contains(output, `{&#34;alpha&#34;:&#34;first&#34;,&#34;beta&#34;:42}`) {
		t.Errorf("expected deterministic JSON detail with sorted keys and html/template-escaped quotes (&#34;); got output that does not contain the expected escaped form")
	}
}

// TestWriteHTML_DetailsSummarySections verifies that native <details>
// and <summary> elements are used for collapsible function sections.
func TestWriteHTML_DetailsSummarySections(t *testing.T) {
	var buf bytes.Buffer
	results := htmlSampleResults()
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// Count <details> elements -- one per function.
	detailsCount := strings.Count(output, "<details>")
	if detailsCount != 2 {
		t.Errorf("expected 2 <details> elements (one per function), got %d", detailsCount)
	}

	summaryCount := strings.Count(output, "<summary>")
	if summaryCount != 2 {
		t.Errorf("expected 2 <summary> elements (one per function), got %d", summaryCount)
	}

	// Verify closing tags match.
	closingDetails := strings.Count(output, "</details>")
	if closingDetails != 2 {
		t.Errorf("expected 2 </details> closing tags, got %d", closingDetails)
	}
}

// TestWriteHTML_NoExternalResources verifies that the HTML output
// contains no external resource references -- no <link>, <script src>,
// remote images, CSS imports, or URLs that cause network loading.
func TestWriteHTML_NoExternalResources(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// No external link elements (stylesheets, etc.).
	if strings.Contains(output, "<link ") {
		t.Error("output must not contain <link> elements")
	}

	// No script elements with src attribute.
	if strings.Contains(output, "<script") {
		t.Error("output must not contain <script> elements")
	}

	// No CSS @import rules.
	if strings.Contains(output, "@import") {
		t.Error("output must not contain CSS @import rules")
	}

	// No remote URLs (http:// or https://).
	if strings.Contains(output, "http://") || strings.Contains(output, "https://") {
		t.Error("output must not contain remote URLs")
	}

	// No external image references.
	if strings.Contains(output, "<img") {
		t.Error("output must not contain <img> elements")
	}
}

// TestWriteHTML_AdversarialEscaping verifies that source-derived
// values containing HTML markup, script tags, quotes, ampersands,
// and attribute-breaking characters are rendered as inert escaped
// text, not as executable content.
func TestWriteHTML_AdversarialEscaping(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/<script>alert(1)</script>",
				Function:  `Inject"onclick="alert(2)`,
				Receiver:  `*Foo&Bar<Baz>`,
				Signature: `func <img src=x onerror=alert(3)>()`,
				Location:  `file"with'quotes.go:1:1`,
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-xss-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    `evil.go:1&col=<script>`,
					Description: `<b>bold</b> & "quoted" & 'apos'`,
					Detail: map[string]any{
						"key": `<script>alert("xss")</script>`,
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// The raw script tag must not appear unescaped.
	if strings.Contains(output, "<script>alert") {
		t.Error("unescaped <script> tag found in output -- XSS vulnerability")
	}
	if strings.Contains(output, "<b>bold</b>") {
		t.Error("unescaped <b> tag found in output")
	}
	// The raw <img> tag must not appear as an actual element.
	// The text "onerror=alert" may appear as escaped text content
	// inside &lt;...&gt; -- that is safe. The vulnerability would be
	// an unescaped <img element.
	if strings.Contains(output, "<img src=x") {
		t.Error("unescaped <img> tag found in output -- XSS vulnerability")
	}
	if strings.Contains(output, `onclick="alert`) {
		t.Error("unescaped onclick handler found in output")
	}

	// Escaped forms must be present (html/template escapes < > " &).
	if !strings.Contains(output, "&lt;script&gt;") {
		t.Error("expected escaped <script> tag as &lt;script&gt;")
	}
	if !strings.Contains(output, "&amp;") {
		t.Error("expected escaped ampersand as &amp;")
	}
	if !strings.Contains(output, "&lt;b&gt;") {
		t.Error("expected escaped <b> tag as &lt;b&gt;")
	}
}

// TestWriteHTML_DeterministicBytes verifies that identical input
// produces byte-identical output across two independent calls.
func TestWriteHTML_DeterministicBytes(t *testing.T) {
	results := htmlSampleResults()
	version := "1.0.0"

	var buf1, buf2 bytes.Buffer
	if err := WriteHTML(&buf1, results, version); err != nil {
		t.Fatalf("first WriteHTML call failed: %v", err)
	}
	if err := WriteHTML(&buf2, results, version); err != nil {
		t.Fatalf("second WriteHTML call failed: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("two calls with identical input produced different output -- rendering is not deterministic")
		// Find first differing byte for debugging.
		b1, b2 := buf1.Bytes(), buf2.Bytes()
		minLen := len(b1)
		if len(b2) < minLen {
			minLen = len(b2)
		}
		for i := 0; i < minLen; i++ {
			if b1[i] != b2[i] {
				t.Errorf("first difference at byte %d: 0x%02x vs 0x%02x", i, b1[i], b2[i])
				break
			}
		}
		if len(b1) != len(b2) {
			t.Errorf("output lengths differ: %d vs %d", len(b1), len(b2))
		}
	}
}

// TestWriteHTML_DeterministicBytesWithDetail verifies deterministic
// output when detail metadata is present (map key ordering).
func TestWriteHTML_DeterministicBytesWithDetail(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Fn",
				Signature: "func Fn()",
				Location:  "fn.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-det-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    "fn.go:2:1",
					Description: "returns value",
					Detail: map[string]any{
						"zebra":   "last",
						"alpha":   "first",
						"middle":  "center",
						"numeric": 42,
					},
				},
			},
		},
	}

	var buf1, buf2 bytes.Buffer
	if err := WriteHTML(&buf1, results, "1.0.0"); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if err := WriteHTML(&buf2, results, "1.0.0"); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("detail metadata with multiple keys produced non-deterministic output")
	}
}

// TestWriteHTML_UnsupportedDetailSerializationError verifies that
// unsupported detail values (e.g., channels, functions) cause
// WriteHTML to return an error rather than silently dropping data.
func TestWriteHTML_UnsupportedDetailSerializationError(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Bad",
				Signature: "func Bad()",
				Location:  "bad.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-bad-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Location:    "bad.go:2:1",
					Description: "returns value",
					Detail: map[string]any{
						"unserializable": math.Inf(1),
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := WriteHTML(&buf, results, "0.1.0")
	if err == nil {
		t.Fatal("expected error for unserializable detail value, got nil")
	}
	if !strings.Contains(err.Error(), "preparing HTML report data") {
		t.Errorf("expected error to contain 'preparing HTML report data', got: %v", err)
	}
	if !strings.Contains(err.Error(), "serializing detail") {
		t.Errorf("expected error to contain 'serializing detail', got: %v", err)
	}
}

// TestWriteHTML_WriterError verifies that template execution errors
// from a failing writer are propagated with context wrapping.
func TestWriteHTML_WriterError(t *testing.T) {
	w := &failWriter{failAfter: 10}
	err := WriteHTML(w, htmlSampleResults(), "0.1.0")
	if err == nil {
		t.Fatal("expected error from failing writer, got nil")
	}
	if !strings.Contains(err.Error(), "executing HTML report template") {
		t.Errorf("expected error to contain 'executing HTML report template', got: %v", err)
	}
}

// failWriter is an io.Writer that fails after writing a specified
// number of bytes.
type failWriter struct {
	failAfter int
	written   int
}

func (w *failWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.failAfter {
		remaining := w.failAfter - w.written
		if remaining > 0 {
			w.written += remaining
			return remaining, errWriteFailed
		}
		return 0, errWriteFailed
	}
	w.written += len(p)
	return len(p), nil
}

var errWriteFailed = &writeError{}

type writeError struct{}

func (e *writeError) Error() string { return "simulated write failure" }

// TestWriteHTML_FunctionWithNoEffects verifies that a function with
// zero side effects renders a "No side effects detected" message
// instead of an empty table.
func TestWriteHTML_FunctionWithNoEffects(t *testing.T) {
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
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "No side effects detected.") {
		t.Error("expected 'No side effects detected.' for function with no effects")
	}
	// Should still have a details/summary section for the function.
	if !strings.Contains(output, "<details>") {
		t.Error("expected <details> section even for function with no effects")
	}
	if !strings.Contains(output, "Pure") {
		t.Error("expected function name 'Pure' in output")
	}
}

// TestWriteHTML_TierCSSClasses verifies that each tier value produces
// the correct CSS class in the rendered output.
func TestWriteHTML_TierCSSClasses(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Multi",
				Signature: "func Multi()",
				Location:  "multi.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{ID: "se-t0", Type: taxonomy.ReturnValue, Tier: taxonomy.TierP0, Description: "p0 effect"},
				{ID: "se-t1", Type: taxonomy.MapMutation, Tier: taxonomy.TierP1, Description: "p1 effect"},
				{ID: "se-t2", Type: taxonomy.FileSystemWrite, Tier: taxonomy.TierP2, Description: "p2 effect"},
				{ID: "se-t3", Type: taxonomy.StdoutWrite, Tier: taxonomy.TierP3, Description: "p3 effect"},
				{ID: "se-t4", Type: taxonomy.ReflectionMutation, Tier: taxonomy.TierP4, Description: "p4 effect"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	tierClasses := []string{"tier-p0", "tier-p1", "tier-p2", "tier-p3", "tier-p4"}
	for _, cls := range tierClasses {
		if !strings.Contains(output, cls) {
			t.Errorf("expected CSS class %q in output", cls)
		}
	}
}

// TestWriteHTML_ClassificationWithoutDetail verifies that when
// classification is present but detail is absent, only the
// classification column appears.
func TestWriteHTML_ClassificationWithoutDetail(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:   "example.com/pkg",
				Function:  "Classified",
				Signature: "func Classified()",
				Location:  "cls.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-cls-001",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Description: "returns value",
					Classification: &taxonomy.Classification{
						Label:      taxonomy.Contractual,
						Confidence: 75,
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteHTML(&buf, results, "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "contractual (75%)") {
		t.Error("expected classification value 'contractual (75%)'")
	}

	// Count "Detail" as a table header -- should not appear since
	// no effects have detail.
	if strings.Contains(output, "<th>Detail</th>") {
		t.Error("Detail column header should not appear when no effects have detail")
	}
}

// --- Unit tests for internal helpers ---

// TestTierCSSClass verifies the CSS class generation for each tier.
func TestTierCSSClass(t *testing.T) {
	tests := []struct {
		tier taxonomy.Tier
		want string
	}{
		{taxonomy.TierP0, "p0"},
		{taxonomy.TierP1, "p1"},
		{taxonomy.TierP2, "p2"},
		{taxonomy.TierP3, "p3"},
		{taxonomy.TierP4, "p4"},
		{taxonomy.Tier("Unknown"), "unknown"},
		{taxonomy.Tier(""), ""},
	}

	for _, tt := range tests {
		got := tierCSSClass(tt.tier)
		if got != tt.want {
			t.Errorf("tierCSSClass(%q) = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

// TestFormatClassification verifies the human-readable classification
// string format.
func TestFormatClassification(t *testing.T) {
	tests := []struct {
		label      taxonomy.ClassificationLabel
		confidence int
		want       string
	}{
		{taxonomy.Contractual, 85, "contractual (85%)"},
		{taxonomy.Incidental, 60, "incidental (60%)"},
		{taxonomy.Ambiguous, 50, "ambiguous (50%)"},
		{taxonomy.Contractual, 100, "contractual (100%)"},
		{taxonomy.Contractual, 0, "contractual (0%)"},
	}

	for _, tt := range tests {
		c := &taxonomy.Classification{
			Label:      tt.label,
			Confidence: tt.confidence,
		}
		got := formatClassification(c)
		if got != tt.want {
			t.Errorf("formatClassification(%s, %d) = %q, want %q",
				tt.label, tt.confidence, got, tt.want)
		}
	}
}

// TestSerializeDetail_SortedKeys verifies that detail map keys are
// sorted in the JSON output for deterministic rendering.
func TestSerializeDetail_SortedKeys(t *testing.T) {
	detail := map[string]any{
		"zebra": "last",
		"alpha": "first",
		"mid":   "center",
	}

	got, err := serializeDetail(detail)
	if err != nil {
		t.Fatalf("serializeDetail failed: %v", err)
	}

	want := `{"alpha":"first","mid":"center","zebra":"last"}`
	if got != want {
		t.Errorf("serializeDetail = %q, want %q", got, want)
	}
}

// TestSerializeDetail_Unserializable verifies that unserializable
// values produce an error.
func TestSerializeDetail_Unserializable(t *testing.T) {
	detail := map[string]any{
		"bad": math.Inf(1),
	}

	_, err := serializeDetail(detail)
	if err == nil {
		t.Fatal("expected error for unserializable detail value, got nil")
	}
}

// TestSerializeDetail_Empty verifies that an empty map serializes
// to "{}".
func TestSerializeDetail_Empty(t *testing.T) {
	got, err := serializeDetail(map[string]any{})
	if err != nil {
		t.Fatalf("serializeDetail failed: %v", err)
	}
	if got != "{}" {
		t.Errorf("serializeDetail(empty) = %q, want %q", got, "{}")
	}
}

// TestBuildHTMLData_VersionPassthrough verifies that the version
// string is passed through to the view model.
func TestBuildHTMLData_VersionPassthrough(t *testing.T) {
	data, err := buildHTMLData(htmlSampleResults(), "2.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if data.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", data.Version, "2.0.0")
	}
}

// TestBuildHTMLData_FunctionCount verifies the correct number of
// functions in the view model.
func TestBuildHTMLData_FunctionCount(t *testing.T) {
	data, err := buildHTMLData(htmlSampleResults(), "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if len(data.Functions) != 2 {
		t.Errorf("len(Functions) = %d, want 2", len(data.Functions))
	}
}

// TestBuildHTMLData_TotalEffects verifies the total effect count
// aggregation.
func TestBuildHTMLData_TotalEffects(t *testing.T) {
	data, err := buildHTMLData(htmlSampleResults(), "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if data.TotalEffects != 3 {
		t.Errorf("TotalEffects = %d, want 3", data.TotalEffects)
	}
}

// TestBuildHTMLData_HasClassificationFlag verifies the flag is set
// only when classification data is present.
func TestBuildHTMLData_HasClassificationFlag(t *testing.T) {
	// Without classification.
	data, err := buildHTMLData(htmlSampleResults(), "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if data.HasClassification {
		t.Error("HasClassification should be false when no effects have classification")
	}

	// With classification.
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package: "pkg", Function: "F", Signature: "func F()", Location: "f.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID: "se-1", Type: taxonomy.ReturnValue, Tier: taxonomy.TierP0,
					Classification: &taxonomy.Classification{
						Label: taxonomy.Contractual, Confidence: 80,
					},
				},
			},
		},
	}
	data, err = buildHTMLData(results, "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if !data.HasClassification {
		t.Error("HasClassification should be true when effects have classification")
	}
}

// TestBuildHTMLData_HasDetailFlag verifies the flag is set only when
// detail metadata is present.
func TestBuildHTMLData_HasDetailFlag(t *testing.T) {
	// Without detail.
	data, err := buildHTMLData(htmlSampleResults(), "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if data.HasDetail {
		t.Error("HasDetail should be false when no effects have detail")
	}

	// With detail.
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package: "pkg", Function: "F", Signature: "func F()", Location: "f.go:1:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID: "se-1", Type: taxonomy.ReturnValue, Tier: taxonomy.TierP0,
					Detail: map[string]any{"key": "value"},
				},
			},
		},
	}
	data, err = buildHTMLData(results, "1.0.0")
	if err != nil {
		t.Fatalf("buildHTMLData failed: %v", err)
	}
	if !data.HasDetail {
		t.Error("HasDetail should be true when effects have detail")
	}
}

// TestWriteHTML_InlineStylePresent verifies that the output contains
// inline CSS within a <style> element (self-contained styling).
func TestWriteHTML_InlineStylePresent(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<style>") {
		t.Error("expected <style> element for inline CSS")
	}
	if !strings.Contains(output, "</style>") {
		t.Error("expected </style> closing tag")
	}
}

// TestWriteHTML_SignatureInOutput verifies that function signatures
// appear in the rendered HTML.
func TestWriteHTML_SignatureInOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, htmlSampleResults(), "0.1.0"); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()

	// Signatures contain special characters that get escaped.
	// "func (s *Store) Save(item Item) (int64, error)" should be present.
	if !strings.Contains(output, "func (s *Store) Save(item Item) (int64, error)") {
		t.Error("expected function signature in output")
	}
	if !strings.Contains(output, "func Delete(id string) error") {
		t.Error("expected function signature for Delete in output")
	}
}
