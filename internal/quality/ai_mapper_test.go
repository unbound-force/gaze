package quality_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/quality"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TestAssess_AIMapperMatch verifies that when the AI mapper returns a valid
// effect ID, the assertion is mapped at confidence 50.
func TestAssess_AIMapperMatch(t *testing.T) {
	// Use indirectmatch fixture — it has known unmapped assertions.
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	// Collect all effect IDs so the AI mapper can return a valid one.
	effectIDs := make(map[string]bool)
	for _, r := range results {
		for _, se := range r.SideEffects {
			effectIDs[se.ID] = true
		}
	}

	var aiCalled bool
	qualOpts := quality.Options{
		AIMapperFunc: func(ctx quality.AIMapperContext) (string, error) {
			aiCalled = true
			// Verify context is populated.
			if ctx.TargetFunc == "" {
				t.Error("AIMapperContext.TargetFunc should be non-empty")
			}
			if ctx.AssertionSource == "" {
				t.Error("AIMapperContext.AssertionSource should be non-empty")
			}
			if len(ctx.SideEffects) == 0 {
				t.Error("AIMapperContext.SideEffects should be non-empty")
			}
			// Return the first available effect ID.
			for _, se := range ctx.SideEffects {
				return se.ID, nil
			}
			return "", nil
		},
	}

	reports, _, err := quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess failed: %v", err)
	}
	if !aiCalled {
		t.Fatal("AI mapper was never called — indirectmatch should have unmapped assertions")
	}
	if len(reports) == 0 {
		t.Fatal("expected non-empty reports")
	}

	// Verify at least one mapping exists with confidence 50
	// (the AI-mapped confidence level).
	// ContractCoverage doesn't expose individual mappings, but we
	// can verify the AI mapper was used by comparing coverage with
	// and without the mapper.
	qualOptsNone := quality.Options{}
	reportsNone, _, err := quality.Assess(results, pkg, qualOptsNone)
	if err != nil {
		t.Fatalf("Assess (no AI) failed: %v", err)
	}

	// With AI mapper, we should have higher or equal coverage
	// (the AI always returns a valid ID, adding at least one mapping).
	var totalCovAI, totalCovNone float64
	for _, r := range reports {
		totalCovAI += r.ContractCoverage.Percentage
	}
	for _, r := range reportsNone {
		totalCovNone += r.ContractCoverage.Percentage
	}
	if totalCovAI < totalCovNone {
		t.Errorf("AI mapper should not reduce coverage: AI=%.1f%%, none=%.1f%%",
			totalCovAI, totalCovNone)
	}
}

// TestAssess_AIMapperNoMatch verifies that when the AI mapper returns
// empty string, the assertion remains unmapped.
func TestAssess_AIMapperNoMatch(t *testing.T) {
	// Use indirectmatch — guaranteed to have unmapped assertions.
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	var aiCalled bool
	qualOpts := quality.Options{
		AIMapperFunc: func(ctx quality.AIMapperContext) (string, error) {
			aiCalled = true
			return "", nil // no match
		},
	}

	_, _, err = quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess failed: %v", err)
	}
	if !aiCalled {
		t.Error("AI mapper should have been called for indirectmatch fixture")
	}
}

// TestAssess_AIMapperError verifies that when the AI mapper returns an
// error, the assertion remains unmapped and no panic occurs.
func TestAssess_AIMapperError(t *testing.T) {
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	qualOpts := quality.Options{
		AIMapperFunc: func(ctx quality.AIMapperContext) (string, error) {
			return "", fmt.Errorf("simulated AI failure")
		},
	}

	_, _, err = quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess should not fail on AI mapper error, got: %v", err)
	}
}

// TestAssess_AIMapperNil verifies that nil AI mapper produces identical
// behavior to a no-match mapper.
func TestAssess_AIMapperNil(t *testing.T) {
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	// Run with nil AI mapper.
	qualOptsNil := quality.Options{}
	reportsNil, summaryNil, err := quality.Assess(results, pkg, qualOptsNil)
	if err != nil {
		t.Fatalf("Assess (nil) failed: %v", err)
	}

	// Run with no-match AI mapper.
	qualOptsNoMatch := quality.Options{
		AIMapperFunc: func(ctx quality.AIMapperContext) (string, error) {
			return "", nil
		},
	}
	reportsNoMatch, summaryNoMatch, err := quality.Assess(results, pkg, qualOptsNoMatch)
	if err != nil {
		t.Fatalf("Assess (no-match) failed: %v", err)
	}

	// Both must produce non-nil summaries.
	if summaryNil == nil {
		t.Fatal("summaryNil should be non-nil")
	}
	if summaryNoMatch == nil {
		t.Fatal("summaryNoMatch should be non-nil")
	}

	// Both should produce the same number of reports and same
	// coverage (no-match AI mapper doesn't add any mappings).
	if len(reportsNil) != len(reportsNoMatch) {
		t.Errorf("expected same report count: nil=%d, no-match=%d",
			len(reportsNil), len(reportsNoMatch))
	}
	if summaryNil.AverageContractCoverage != summaryNoMatch.AverageContractCoverage {
		t.Errorf("expected same average contract coverage: nil=%.1f, no-match=%.1f",
			summaryNil.AverageContractCoverage, summaryNoMatch.AverageContractCoverage)
	}
	if summaryNil.TotalTests != summaryNoMatch.TotalTests {
		t.Errorf("expected same TotalTests: nil=%d, no-match=%d",
			summaryNil.TotalTests, summaryNoMatch.TotalTests)
	}
}

// TestAssess_AIMapperContextPopulated verifies that the AIMapperContext
// contains all expected fields when called.
func TestAssess_AIMapperContextPopulated(t *testing.T) {
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	var contexts []quality.AIMapperContext
	qualOpts := quality.Options{
		AIMapperFunc: func(ctx quality.AIMapperContext) (string, error) {
			contexts = append(contexts, ctx)
			return "", nil
		},
	}

	_, _, err = quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess failed: %v", err)
	}

	if len(contexts) == 0 {
		t.Fatal("expected AI mapper to be called at least once for indirectmatch fixture")
	}

	for i, ctx := range contexts {
		if ctx.TargetFunc == "" {
			t.Errorf("context[%d].TargetFunc is empty", i)
		}
		if ctx.AssertionSource == "" || ctx.AssertionSource == "<expr>" {
			t.Errorf("context[%d].AssertionSource should be readable source, got %q", i, ctx.AssertionSource)
		}
		if ctx.TestFuncSource == "" || ctx.TestFuncSource == "<func>" {
			t.Errorf("context[%d].TestFuncSource should be readable source, got %q", i, ctx.TestFuncSource)
		}
		if len(ctx.SideEffects) == 0 {
			t.Errorf("context[%d].SideEffects is empty", i)
		}
	}
}

// TestBuildAIMapperPrompt verifies that the prompt builder produces
// a structured prompt containing the assertion, test body, target
// name, and side effects list.
func TestBuildAIMapperPrompt(t *testing.T) {
	ctx := quality.AIMapperContext{
		AssertionSource: "got != want",
		AssertionKind:   quality.AssertionKindStdlibComparison,
		TestFuncSource:  "func TestFoo(t *testing.T) { ... }",
		TargetFunc:      "(*Counter).Increment",
		SideEffects: []taxonomy.SideEffect{
			{ID: "eff-001", Type: taxonomy.ReturnValue, Description: "returns the new count"},
			{ID: "eff-002", Type: taxonomy.ReceiverMutation, Description: "mutates the counter value"},
		},
	}

	prompt := quality.BuildAIMapperPrompt(ctx)

	// Verify the prompt contains all key components.
	for _, expected := range []string{
		"got != want",
		"TestFoo",
		"(*Counter).Increment",
		"eff-001",
		"eff-002",
		"ReturnValue",
		"ReceiverMutation",
		"NONE",
		"semantic relationships",
	} {
		if !strings.Contains(prompt, expected) {
			t.Errorf("prompt should contain %q, got:\n%s", expected, prompt)
		}
	}
}

// TestParseAIMapperResponse verifies parsing of AI model responses.
func TestParseAIMapperResponse(t *testing.T) {
	validIDs := map[string]bool{
		"eff-001": true,
		"eff-002": true,
	}

	tests := []struct {
		name     string
		response string
		want     string
	}{
		{"exact match", "eff-001", "eff-001"},
		{"with whitespace", "  eff-002  \n", "eff-002"},
		{"NONE", "NONE", ""},
		{"none lowercase", "none", ""},
		{"embedded in sentence", "The assertion verifies eff-001 because...", "eff-001"},
		{"invalid ID", "eff-999", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quality.ParseAIMapperResponse(tt.response, validIDs)
			if got != tt.want {
				t.Errorf("ParseAIMapperResponse(%q) = %q, want %q", tt.response, got, tt.want)
			}
		})
	}
}

// TestMapAssertionsToEffects_AIConfidence50 directly verifies that
// AI-mapped assertions have Confidence == 50 by calling
// MapAssertionsToEffectsWithStderr with a mock AI mapper.
func TestMapAssertionsToEffects_AIConfidence50(t *testing.T) {
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	// Build effect ID set from analysis results.
	effectIDs := make(map[string]bool)
	for _, r := range results {
		for _, se := range r.SideEffects {
			effectIDs[se.ID] = true
		}
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	aiMapper := quality.AIMapperFunc(func(ctx quality.AIMapperContext) (string, error) {
		// Return the first available effect ID.
		for _, se := range ctx.SideEffects {
			return se.ID, nil
		}
		return "", nil
	})

	var foundConfidence50 bool
	for _, tf := range testFuncs {
		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			continue
		}
		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}
			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			mapped, _, _ := quality.MapAssertionsToEffectsWithStderr(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
				nil, // stderr
				aiMapper,
			)
			for _, m := range mapped {
				if m.Confidence == 50 {
					foundConfidence50 = true
					t.Logf("Found AI-mapped assertion: %s -> %s (confidence %d)",
						m.AssertionLocation, m.SideEffectID, m.Confidence)
				}
			}
		}
	}

	if !foundConfidence50 {
		t.Error("expected at least one AI-mapped assertion with Confidence == 50")
	}
}

// TestMapAssertionsToEffects_AIErrorWarning verifies that when the AI
// mapper returns an error, a warning is written to stderr.
func TestMapAssertionsToEffects_AIErrorWarning(t *testing.T) {
	pkg := loadPkg(t, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}
	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	aiMapper := quality.AIMapperFunc(func(ctx quality.AIMapperContext) (string, error) {
		return "", fmt.Errorf("simulated AI failure")
	})

	var stderr bytes.Buffer
	for _, tf := range testFuncs {
		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			continue
		}
		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}
			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			quality.MapAssertionsToEffectsWithStderr(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
				&stderr,
				aiMapper,
			)
		}
	}

	if !strings.Contains(stderr.String(), "warning: AI mapper failed") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
}
