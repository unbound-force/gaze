package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/protocol"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// mustInternalFakeBinary reuses the lazy fake analyzer binary from call_test.go.
// Both files are package adapter (internal) so they share the same namespace.
func mustInternalFakeBinary(t *testing.T) string {
	t.Helper()
	return buildCallTestFakeAnalyzer(t)
}

func mustInternalClient(t *testing.T, args ...string) *protocol.Client {
	t.Helper()
	bin := mustInternalFakeBinary(t)
	allArgs := append([]string{"--stdio"}, args...)
	client, err := protocol.NewClient(bin, allArgs...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func mustInternalInitialize(t *testing.T, client *protocol.Client) protocol.Capabilities {
	t.Helper()
	ctx := context.Background()
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}
	var initResult protocol.InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	return initResult.Capabilities
}

// --- fetchClassifySignals tests ---

func TestFetchClassifySignals_Success(t *testing.T) {
	client := mustInternalClient(t)
	defer func() { _ = client.Close() }()
	mustInternalInitialize(t, client)

	var stderr bytes.Buffer
	signals := fetchClassifySignals(client, "/tmp/project", []string{"./..."}, &stderr)

	// The fake analyzer returns at least one signal (it has a
	// classify_signals handler even when the capability is false).
	if len(signals) == 0 {
		t.Fatal("expected at least one signal, got none")
	}

	// Verify the canned signal data.
	found := false
	for _, s := range signals {
		if s.Function == "divide" && s.SideEffectType == "ErrorReturn" {
			found = true
			if s.Source != "docstring" {
				t.Errorf("signal source = %q, want %q", s.Source, "docstring")
			}
			if s.Weight != 20 {
				t.Errorf("signal weight = %d, want 20", s.Weight)
			}
		}
	}
	if !found {
		t.Error("expected signal for divide/ErrorReturn")
	}
}

func TestFetchClassifySignals_TransportError(t *testing.T) {
	// Use --crash-after=initialize to make the analyzer exit after
	// initialize, so the classify_signals call hits a transport error.
	client := mustInternalClient(t, "--crash-after=initialize")
	defer func() { _ = client.Close() }()
	mustInternalInitialize(t, client)

	var stderr bytes.Buffer
	signals := fetchClassifySignals(client, "/tmp/project", []string{"./..."}, &stderr)

	// Graceful degradation: nil signals, warning on stderr.
	if signals != nil {
		t.Errorf("expected nil signals, got %d", len(signals))
	}
	if !strings.Contains(stderr.String(), "warning: classify_signals failed:") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
}

func TestFetchClassifySignals_ProtocolError(t *testing.T) {
	// Use --error-response to make the analyzer return JSON-RPC errors.
	client := mustInternalClient(t, "--error-response")
	defer func() { _ = client.Close() }()
	mustInternalInitialize(t, client)

	var stderr bytes.Buffer
	signals := fetchClassifySignals(client, "/tmp/project", []string{"./..."}, &stderr)

	// Graceful degradation: nil signals, warning on stderr.
	if signals != nil {
		t.Errorf("expected nil signals, got %d", len(signals))
	}
	if !strings.Contains(stderr.String(), "warning: classify_signals failed:") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
}

func TestFetchClassifySignals_NilStderr(t *testing.T) {
	// Use --crash-after=initialize to trigger an error, with nil stderr.
	// This verifies no panic occurs when stderr is nil.
	client := mustInternalClient(t, "--crash-after=initialize")
	defer func() { _ = client.Close() }()
	mustInternalInitialize(t, client)

	signals := fetchClassifySignals(client, "/tmp/project", []string{"./..."}, nil)
	if signals != nil {
		t.Errorf("expected nil signals with nil stderr on error, got %d", len(signals))
	}
}

// --- mergeClassifications tests ---

func TestMergeClassifications_MatchingSignals(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
					// No classification — should be filled.
				},
			},
		},
	}

	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         20,
			Reasoning:      "function documents error return",
		},
	}

	mergeClassifications(cached, signals, nil)

	if cached[0].SideEffects[0].Classification == nil {
		t.Fatal("expected classification to be attached")
	}

	c := cached[0].SideEffects[0].Classification
	// ErrorReturn is P0 (tier boost +25), base 50 + 25 + 20 = 95.
	// Default contractual threshold is 80, so label should be contractual.
	if c.Label != taxonomy.Contractual {
		t.Errorf("label = %q, want %q", c.Label, taxonomy.Contractual)
	}
	if c.Confidence != 95 {
		t.Errorf("confidence = %d, want 95", c.Confidence)
	}
}

func TestMergeClassifications_NoMatchingSignals(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
				},
			},
		},
	}

	// Signals for a different function — should not match.
	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Bar",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         20,
		},
	}

	mergeClassifications(cached, signals, nil)

	if cached[0].SideEffects[0].Classification != nil {
		t.Error("expected classification to remain nil (no matching signals)")
	}
}

func TestMergeClassifications_PreservesPreClassified(t *testing.T) {
	existing := &taxonomy.Classification{
		Label:      taxonomy.Incidental,
		Confidence: 30,
		Reasoning:  "inline from analyzer",
	}
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type:           taxonomy.ErrorReturn,
					Classification: existing,
				},
			},
		},
	}

	// Signals that would change classification if applied.
	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         20,
		},
	}

	mergeClassifications(cached, signals, nil)

	// Pre-classified effect should be preserved.
	if cached[0].SideEffects[0].Classification != existing {
		t.Error("pre-classified effect was overwritten")
	}
	if cached[0].SideEffects[0].Classification.Label != taxonomy.Incidental {
		t.Errorf("label = %q, want %q", cached[0].SideEffects[0].Classification.Label, taxonomy.Incidental)
	}
}

func TestMergeClassifications_NilConfig(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
				},
			},
		},
	}

	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         20,
		},
	}

	// Passing nil config — should use DefaultConfig internally.
	mergeClassifications(cached, signals, nil)

	if cached[0].SideEffects[0].Classification == nil {
		t.Fatal("expected classification with nil config (uses DefaultConfig)")
	}
}

func TestMergeClassifications_CustomConfig(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
				},
			},
		},
	}

	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         20,
		},
	}

	// Custom config with very high contractual threshold.
	cfg := config.DefaultConfig()
	cfg.Classification.Thresholds.Contractual = 99

	mergeClassifications(cached, signals, cfg)

	c := cached[0].SideEffects[0].Classification
	if c == nil {
		t.Fatal("expected classification to be attached")
	}
	// ErrorReturn P0: base 50 + tier 25 + weight 20 = 95, but threshold is 99.
	// Score 95 < 99, and 95 >= 50 (incidental threshold), so label should be ambiguous.
	if c.Label != taxonomy.Ambiguous {
		t.Errorf("label = %q, want %q (custom threshold 99)", c.Label, taxonomy.Ambiguous)
	}
}

func TestMergeClassifications_EmptySignals(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
				},
			},
		},
	}

	// Empty signals — should be a no-op.
	mergeClassifications(cached, nil, nil)

	if cached[0].SideEffects[0].Classification != nil {
		t.Error("expected classification to remain nil with nil signals")
	}

	mergeClassifications(cached, []protocol.ClassifySignalData{}, nil)

	if cached[0].SideEffects[0].Classification != nil {
		t.Error("expected classification to remain nil with empty signals")
	}
}

func TestMergeClassifications_SideEffectTypeMismatch(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ReturnValue,
				},
			},
		},
	}

	// Signal for a different SideEffectType — should be silently ignored.
	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ContainerMutation",
			Source:         "type_annotation",
			Weight:         30,
		},
	}

	mergeClassifications(cached, signals, nil)

	if cached[0].SideEffects[0].Classification != nil {
		t.Error("expected classification to remain nil (SideEffectType mismatch)")
	}
}

func TestMergeClassifications_MultipleSignalsSameKey(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type: taxonomy.ErrorReturn,
				},
			},
		},
	}

	// Two signals for the same (pkg, Foo, ErrorReturn) key with different
	// sources — both should be aggregated by ComputeScore.
	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "docstring",
			Weight:         10,
		},
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "ErrorReturn",
			Source:         "naming_convention",
			Weight:         15,
		},
	}

	mergeClassifications(cached, signals, nil)

	c := cached[0].SideEffects[0].Classification
	if c == nil {
		t.Fatal("expected classification from aggregated signals")
	}
	// ErrorReturn is P0 (tier boost +25): base 50 + 25 + 10 + 15 = 100,
	// clamped to 100. Label should be contractual.
	if c.Label != taxonomy.Contractual {
		t.Errorf("label = %q, want %q", c.Label, taxonomy.Contractual)
	}
	if c.Confidence != 100 {
		t.Errorf("confidence = %d, want 100 (clamped)", c.Confidence)
	}
}

func TestMergeClassifications_MultipleEffectsSameType(t *testing.T) {
	cached := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					Type:     taxonomy.MapMutation,
					Location: "file.go:10",
				},
				{
					Type:     taxonomy.MapMutation,
					Location: "file.go:20",
				},
			},
		},
	}

	signals := []protocol.ClassifySignalData{
		{
			Package:        "pkg",
			Function:       "Foo",
			SideEffectType: "MapMutation",
			Source:         "type_annotation",
			Weight:         30,
		},
	}

	mergeClassifications(cached, signals, nil)

	// Both effects of the same type should be classified.
	// MapMutation is P1 (tier boost +10): base 50 + 10 + weight 30 = 90.
	// Default contractual threshold is 80, so label should be contractual.
	for i, e := range cached[0].SideEffects {
		if e.Classification == nil {
			t.Fatalf("effect[%d] at %s: expected classification to be attached", i, e.Location)
		}
		if e.Classification.Label != taxonomy.Contractual {
			t.Errorf("effect[%d] label = %q, want %q", i, e.Classification.Label, taxonomy.Contractual)
		}
		if e.Classification.Confidence != 90 {
			t.Errorf("effect[%d] confidence = %d, want 90", i, e.Classification.Confidence)
		}
	}

	// Both should have the same label and confidence.
	if cached[0].SideEffects[0].Classification.Label != cached[0].SideEffects[1].Classification.Label {
		t.Errorf("effects have different labels: %q vs %q",
			cached[0].SideEffects[0].Classification.Label,
			cached[0].SideEffects[1].Classification.Label)
	}
}
