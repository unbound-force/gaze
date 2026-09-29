package classify_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// testdataDir returns the absolute path to the testdata/src directory.
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "src")
}

// loadTestPackages loads the test fixture packages for classification
// testing. Returns all packages in the testdata module.
func loadTestPackages(t *testing.T, patterns ...string) []*packages.Package {
	t.Helper()
	dir := testdataDir()

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:   dir,
		Tests: false,
	}

	// Load all packages to enable cross-package analysis.
	allPatterns := []string{"./..."}
	if len(patterns) > 0 {
		allPatterns = patterns
	}

	pkgs, err := packages.Load(cfg, allPatterns...)
	if err != nil {
		t.Fatalf("loading test packages: %v", err)
	}

	var valid []*packages.Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) == 0 {
			valid = append(valid, pkg)
		}
	}

	if len(valid) == 0 {
		t.Fatal("no valid test packages loaded")
	}

	return valid
}

// findPackage finds a package by suffix in the loaded packages.
func findPackage(pkgs []*packages.Package, suffix string) *packages.Package {
	for _, pkg := range pkgs {
		if len(pkg.PkgPath) >= len(suffix) &&
			pkg.PkgPath[len(pkg.PkgPath)-len(suffix):] == suffix {
			return pkg
		}
	}
	return nil
}

// TestNamingSignal_ContractualPrefixes tests naming convention
// detection for contractual function names.
func TestNamingSignal_ContractualPrefixes(t *testing.T) {
	tests := []struct {
		name       string
		funcName   string
		effectType taxonomy.SideEffectType
		wantWeight int
	}{
		{"GetData returns", "GetData", taxonomy.ReturnValue, 10},
		{"SaveRecord mutation", "SaveRecord", taxonomy.ErrorReturn, 10},
		{"FetchConfig returns", "FetchConfig", taxonomy.ReturnValue, 10},
		{"DeleteItem error", "DeleteItem", taxonomy.ErrorReturn, 10},
		{"HandleRequest any", "HandleRequest", taxonomy.ReceiverMutation, 10},
		// New/NewXxx constructor prefix — implies ReturnValue and ErrorReturn.
		{"New_ReturnValue", "NewClient", taxonomy.ReturnValue, 10},
		{"New_ExactMatch", "New", taxonomy.ReturnValue, 10},
		{"New_ErrorReturn", "NewStore", taxonomy.ErrorReturn, 10},
		{"New_NoMatchMutation", "NewClient", taxonomy.ReceiverMutation, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := classify.AnalyzeNamingSignal(tt.funcName, tt.effectType)
			if s.Weight != tt.wantWeight {
				t.Errorf("AnalyzeNamingSignal(%q, %s) weight = %d, want %d",
					tt.funcName, tt.effectType, s.Weight, tt.wantWeight)
			}
		})
	}
}

// TestNamingSignal_IncidentalPrefixes tests naming convention
// detection for incidental function names with I/O effect types.
func TestNamingSignal_IncidentalPrefixes(t *testing.T) {
	tests := []struct {
		funcName string
	}{
		{"logError"},
		{"LogInfo"},
		{"debugTrace"},
		{"Debug"},
		{"traceRequest"},
		{"printResult"},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			// I/O effect types should be penalized by incidental prefixes.
			s := classify.AnalyzeNamingSignal(tt.funcName, taxonomy.LogWrite)
			if s.Weight >= 0 {
				t.Errorf("AnalyzeNamingSignal(%q, LogWrite) weight = %d, want negative",
					tt.funcName, s.Weight)
			}
		})
	}
}

// TestNamingSignal_IncidentalTypeGuard verifies that incidental naming
// prefixes only apply to I/O effect types, not to P0 effects like
// ReturnValue. See issue #105.
func TestNamingSignal_IncidentalTypeGuard(t *testing.T) {
	tests := []struct {
		name       string
		funcName   string
		effectType taxonomy.SideEffectType
		wantWeight int
	}{
		// P0 effects on incidental-prefixed functions → no penalty.
		{"ReturnValue/LogAndCompute", "LogAndCompute", taxonomy.ReturnValue, 0},
		{"ErrorReturn/DebugAndFetch", "DebugAndFetch", taxonomy.ErrorReturn, 0},
		{"ReceiverMutation/traceState", "traceState", taxonomy.ReceiverMutation, 0},
		{"PointerArgMutation/printToBuffer", "printToBuffer", taxonomy.PointerArgMutation, 0},
		{"SentinelError/LogError", "LogError", taxonomy.SentinelError, 0},

		// Non-I/O P1/P2 effects → no penalty (boundary: not in appliesTo).
		{"ChannelSend/LogEvents", "LogEvents", taxonomy.ChannelSend, 0},
		{"WriterOutput/LogAndWrite", "LogAndWrite", taxonomy.WriterOutput, 0},

		// I/O effects on incidental-prefixed functions → penalty applied.
		{"LogWrite/LogAndCompute", "LogAndCompute", taxonomy.LogWrite, -10},
		{"StdoutWrite/PrintSummary", "PrintSummary", taxonomy.StdoutWrite, -10},
		{"StderrWrite/logWarning", "logWarning", taxonomy.StderrWrite, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := classify.AnalyzeNamingSignal(tt.funcName, tt.effectType)
			if s.Weight != tt.wantWeight {
				t.Errorf("AnalyzeNamingSignal(%q, %s) weight = %d, want %d",
					tt.funcName, tt.effectType, s.Weight, tt.wantWeight)
			}
		})
	}
}

// TestNamingSignal_NoMatch tests that unknown names produce zero
// signal.
func TestNamingSignal_NoMatch(t *testing.T) {
	s := classify.AnalyzeNamingSignal("computeHash", taxonomy.ReturnValue)
	if s.Source != "" {
		t.Errorf("expected zero signal for %q, got source=%q weight=%d",
			"computeHash", s.Source, s.Weight)
	}
}

// TestTierBoost verifies that the tier-based confidence boost
// is correctly applied for P0, P1, and P2+ effect types.
func TestTierBoost(t *testing.T) {
	cases := []struct {
		name       string
		effectType taxonomy.SideEffectType
		wantBoost  int
	}{
		// P0 types → +25
		{"ReturnValue", taxonomy.ReturnValue, 25},
		{"ErrorReturn", taxonomy.ErrorReturn, 25},
		{"SentinelError", taxonomy.SentinelError, 25},
		{"ReceiverMutation", taxonomy.ReceiverMutation, 25},
		{"PointerArgMutation", taxonomy.PointerArgMutation, 25},
		// P1 types → +10 (all 8 P1 types)
		{"WriterOutput", taxonomy.WriterOutput, 10},
		{"SliceMutation", taxonomy.SliceMutation, 10},
		{"MapMutation", taxonomy.MapMutation, 10},
		{"GlobalMutation", taxonomy.GlobalMutation, 10},
		{"HTTPResponseWrite", taxonomy.HTTPResponseWrite, 10},
		{"ChannelSend", taxonomy.ChannelSend, 10},
		{"ChannelClose", taxonomy.ChannelClose, 10},
		{"DeferredReturnMutation", taxonomy.DeferredReturnMutation, 10},
		// P2+ types → 0
		{"GoroutineSpawn", taxonomy.GoroutineSpawn, 0},
		{"FileSystemWrite", taxonomy.FileSystemWrite, 0},
		// Empty/unknown type → 0
		{"empty", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// ComputeScore with no signals: base (50) + tierBoost.
			c := classify.ComputeScore(tc.effectType, nil, nil)
			wantConfidence := 50 + tc.wantBoost
			if c.Confidence != wantConfidence {
				t.Errorf("ComputeScore(%q, nil) confidence = %d, want %d",
					tc.effectType, c.Confidence, wantConfidence)
			}
		})
	}
}

// TestScoreComputation_BaseConfidence tests that zero signals on a
// P2+ effect produce a score of 50 (base confidence, no tier boost).
func TestScoreComputation_BaseConfidence(t *testing.T) {
	c := classify.ComputeScore("", nil, nil)
	if c.Confidence != 50 {
		t.Errorf("zero signals: confidence = %d, want 50", c.Confidence)
	}
	if c.Label != taxonomy.Ambiguous {
		t.Errorf("zero signals: label = %q, want %q", c.Label, taxonomy.Ambiguous)
	}
}

// TestScoreComputation_Contractual tests that strong positive
// signals produce a contractual classification.
func TestScoreComputation_Contractual(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 10},
	}

	c := classify.ComputeScore("", signals, nil)
	// 50 + 30 + 10 = 90 >= 80 = contractual.
	if c.Label != taxonomy.Contractual {
		t.Errorf("label = %q, want %q", c.Label, taxonomy.Contractual)
	}
	if c.Confidence != 90 {
		t.Errorf("confidence = %d, want 90", c.Confidence)
	}
}

// TestScoreComputation_Incidental tests that negative signals
// produce an incidental classification.
func TestScoreComputation_Incidental(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "naming", Weight: -10},
	}

	c := classify.ComputeScore("", signals, nil)
	// 50 - 10 = 40 < 50 = incidental.
	if c.Label != taxonomy.Incidental {
		t.Errorf("label = %q, want %q", c.Label, taxonomy.Incidental)
	}
	if c.Confidence != 40 {
		t.Errorf("confidence = %d, want 40", c.Confidence)
	}
}

// TestScoreComputation_Contradiction tests that contradicting
// signals apply a penalty.
func TestScoreComputation_Contradiction(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "naming", Weight: -10},
	}

	c := classify.ComputeScore("", signals, nil)
	// 50 + 30 - 10 - 20 (contradiction) = 50.
	if c.Confidence != 50 {
		t.Errorf("contradiction: confidence = %d, want 50", c.Confidence)
	}
	if c.Label != taxonomy.Ambiguous {
		t.Errorf("contradiction: label = %q, want %q", c.Label, taxonomy.Ambiguous)
	}
}

// TestScoreComputation_ClampToZero tests that very negative scores
// clamp to 0.
func TestScoreComputation_ClampToZero(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "naming", Weight: -10},
		{Source: "godoc", Weight: -15},
		{Source: "another", Weight: -30},
	}

	c := classify.ComputeScore("", signals, nil)
	if c.Confidence != 0 {
		t.Errorf("clamp: confidence = %d, want 0", c.Confidence)
	}
}

// TestScoreComputation_ClampTo100 tests that very positive scores
// clamp to 100.
func TestScoreComputation_ClampTo100(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 20},
		{Source: "caller", Weight: 15},
		{Source: "naming", Weight: 10},
		{Source: "godoc", Weight: 15},
	}

	c := classify.ComputeScore("", signals, nil)
	// 50 + 30 + 20 + 15 + 10 + 15 = 140, clamped to 100.
	if c.Confidence != 100 {
		t.Errorf("clamp: confidence = %d, want 100", c.Confidence)
	}
}

// TestScoreComputation_CustomThresholds tests that custom
// thresholds from config are respected.
func TestScoreComputation_CustomThresholds(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Classification.Thresholds.Contractual = 90
	cfg.Classification.Thresholds.Incidental = 40

	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 10},
	}

	c := classify.ComputeScore("", signals, cfg)
	// 50 + 30 + 10 = 90 >= 90 = contractual with custom threshold.
	if c.Label != taxonomy.Contractual {
		t.Errorf("custom threshold: label = %q, want %q",
			c.Label, taxonomy.Contractual)
	}
}

// TestScoreComputation_ReasoningContainsThreshold verifies that the
// Reasoning field is populated and references the classification
// threshold used (FR-014).
func TestScoreComputation_ReasoningContainsThreshold(t *testing.T) {
	// Contractual: score=90, threshold=80.
	contractual := classify.ComputeScore("", []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 10},
	}, nil)
	if contractual.Reasoning == "" {
		t.Error("contractual: expected non-empty Reasoning")
	}
	if !strings.Contains(contractual.Reasoning, "80") {
		t.Errorf("contractual: expected threshold '80' in Reasoning, got: %q",
			contractual.Reasoning)
	}

	// Incidental: score=40, threshold=50.
	incidental := classify.ComputeScore("", []taxonomy.Signal{
		{Source: "naming", Weight: -10},
	}, nil)
	if incidental.Reasoning == "" {
		t.Error("incidental: expected non-empty Reasoning")
	}
	if !strings.Contains(incidental.Reasoning, "50") {
		t.Errorf("incidental: expected threshold '50' in Reasoning, got: %q",
			incidental.Reasoning)
	}

	// Ambiguous: score=60, between 50 and 80.
	ambiguous := classify.ComputeScore("", []taxonomy.Signal{
		{Source: "naming", Weight: 10},
	}, nil)
	if ambiguous.Reasoning == "" {
		t.Error("ambiguous: expected non-empty Reasoning")
	}
	if !strings.Contains(ambiguous.Reasoning, "ambiguous") {
		t.Errorf("ambiguous: expected 'ambiguous' in Reasoning, got: %q",
			ambiguous.Reasoning)
	}
}

// TestScoreComputation_SignalsInResult verifies that the input
// signals are returned in the Classification.Signals field.
func TestScoreComputation_SignalsInResult(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 10},
	}

	c := classify.ComputeScore("", signals, nil)

	if len(c.Signals) == 0 {
		t.Fatal("expected non-empty Signals in Classification")
	}

	// Both input signals must appear in the result.
	sources := make(map[string]int)
	for _, s := range c.Signals {
		sources[s.Source] = s.Weight
	}

	if w, ok := sources["interface"]; !ok || w != 30 {
		t.Errorf("expected signal 'interface' weight=30, got ok=%v w=%d", ok, w)
	}
	if w, ok := sources["visibility"]; !ok || w != 10 {
		t.Errorf("expected signal 'visibility' weight=10, got ok=%v w=%d", ok, w)
	}
}

// TestScoreComputation_ContradictionSignalAdded verifies that when
// both positive and negative signals exist, a contradiction penalty
// signal is added to Classification.Signals (FR-007).
func TestScoreComputation_ContradictionSignalAdded(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "naming", Weight: -10},
	}

	c := classify.ComputeScore("", signals, nil)

	// Contradiction penalty must appear as a signal in the result.
	var contradictionFound bool
	for _, s := range c.Signals {
		if s.Source == "contradiction" {
			contradictionFound = true
			if s.Weight != -20 {
				t.Errorf("contradiction signal weight = %d, want -20", s.Weight)
			}
			if s.Reasoning == "" {
				t.Error("contradiction signal: expected non-empty Reasoning")
			}
		}
	}
	if !contradictionFound {
		t.Error("expected 'contradiction' signal in Classification.Signals")
	}

	// Reasoning must mention the contradiction.
	if !strings.Contains(c.Reasoning, "contradiction") {
		t.Errorf("expected 'contradiction' in Reasoning, got: %q", c.Reasoning)
	}
}

// TestScoreComputation_ZeroWeightSignalsFiltered verifies that zero-
// weight, empty-source signals are excluded from Classification.Signals.
func TestScoreComputation_ZeroWeightSignalsFiltered(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "", Weight: 0}, // should be filtered
	}

	c := classify.ComputeScore("", signals, nil)

	for _, s := range c.Signals {
		if s.Source == "" {
			t.Error("expected empty-source signal to be filtered from result")
		}
	}
}

// TestScoreComputation_Determinism verifies that identical inputs
// produce identical outputs (FR-011).
func TestScoreComputation_Determinism(t *testing.T) {
	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "naming", Weight: 10},
	}

	c1 := classify.ComputeScore("", signals, nil)
	c2 := classify.ComputeScore("", signals, nil)

	if c1.Label != c2.Label {
		t.Errorf("determinism: labels differ: %q vs %q", c1.Label, c2.Label)
	}
	if c1.Confidence != c2.Confidence {
		t.Errorf("determinism: confidence differs: %d vs %d",
			c1.Confidence, c2.Confidence)
	}
}

// TestClassify_ContractsPackage tests end-to-end classification on
// the contracts fixture package.
func TestClassify_ContractsPackage(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	// Analyze the contracts package first.
	opts := analysis.Options{
		IncludeUnexported: false,
	}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("no analysis results for contracts package")
	}

	// Classify.
	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	// Verify that all side effects have classifications.
	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				t.Errorf("function %s, effect %s: no classification",
					result.Target.Function, se.Type)
			}
		}
	}

	// Check that interface-implementing methods have high confidence.
	for _, result := range classified {
		if result.Target.Function == "Save" ||
			result.Target.Function == "Delete" ||
			result.Target.Function == "Write" {
			for _, se := range result.SideEffects {
				if se.Classification == nil {
					continue
				}
				if se.Classification.Confidence < 70 {
					t.Errorf("interface method %s.%s: confidence %d, want >= 70",
						result.Target.Function, se.Type,
						se.Classification.Confidence)
				}
			}
		}
	}
}

// TestClassify_IncidentalPackage tests that incidental effects
// are classified with low confidence.
func TestClassify_IncidentalPackage(t *testing.T) {
	allPkgs := loadTestPackages(t)
	incidentalPkg := findPackage(allPkgs, "incidental")
	if incidentalPkg == nil {
		t.Fatal("incidental package not found")
	}

	opts := analysis.Options{
		IncludeUnexported: true,
	}
	results, err := analysis.Analyze(incidentalPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      incidentalPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	// Verify that all side effects have classifications.
	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				t.Errorf("function %s, effect %s: no classification",
					result.Target.Function, se.Type)
				continue
			}
			// P0 effects (ReturnValue, ErrorReturn, ReceiverMutation,
			// etc.) receive a tier boost (+25, issue #71) that pushes
			// them toward contractual. The final confidence may be
			// below 75 if negative signals (naming, contradiction)
			// apply, but P0 effects are exempt from the "must not be
			// contractual" check because they are definitionally
			// part of the function's contract.
			if taxonomy.TierOf(se.Type) == taxonomy.TierP0 {
				t.Logf("P0 effect %s on %s: confidence %d (exempt from incidental check)",
					se.Type, result.Target.Function, se.Classification.Confidence)
				continue
			}
			if se.Classification.Label == taxonomy.Contractual {
				t.Errorf("incidental function %s, non-P0 effect %s: "+
					"classified as contractual (confidence %d)",
					result.Target.Function, se.Type,
					se.Classification.Confidence)
			}
		}
	}
}

// TestSC001_MechanicalContractualAccuracy verifies that the
// classification engine achieves >= 90% true-positive rate on the
// contracts fixture, which contains functions with known contractual
// side effects. This test operationalizes SC-001 from Spec 002:
// "Given known-contractual inputs, the engine labels >= 90% as
// Contractual or Ambiguous with Confidence > 60."
func TestSC001_MechanicalContractualAccuracy(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	total := 0
	truePositives := 0

	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				continue
			}
			total++
			// A true positive is Contractual or Ambiguous with
			// confidence > 60 (per SC-001 definition). All
			// contractual side effects in the contracts fixture are
			// expected to reach at least Contractual or high-
			// confidence Ambiguous via mechanical signals.
			label := se.Classification.Label
			conf := se.Classification.Confidence
			if label == taxonomy.Contractual ||
				(label == taxonomy.Ambiguous && conf > 60) {
				truePositives++
			}
		}
	}

	if total == 0 {
		t.Fatal("no classified side effects found — fixture may be empty")
	}

	rate := float64(truePositives) / float64(total)
	const minRate = 0.90

	if rate < minRate {
		t.Errorf("true-positive rate = %.1f%% (%d/%d), want >= %.0f%%",
			rate*100, truePositives, total, minRate*100)
	}
}

// TestSC002_DeterministicOutput verifies that repeated runs on the
// same input produce identical classification output (SC-002).
func TestSC002_DeterministicOutput(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results1, _ := analysis.Analyze(contractsPkg, opts)
	results2, _ := analysis.Analyze(contractsPkg, opts)

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
	}

	c1 := classify.Classify(results1, classifyOpts)
	c2 := classify.Classify(results2, classifyOpts)

	if len(c1) != len(c2) {
		t.Fatalf("SC-002 determinism: result count differs: %d vs %d",
			len(c1), len(c2))
	}

	for i := range c1 {
		for j := range c1[i].SideEffects {
			if j >= len(c2[i].SideEffects) {
				break
			}
			se1 := c1[i].SideEffects[j]
			se2 := c2[i].SideEffects[j]

			if se1.Classification == nil || se2.Classification == nil {
				continue
			}
			if se1.Classification.Label != se2.Classification.Label {
				t.Errorf("SC-002: function %s effect %s: labels differ: %q vs %q",
					c1[i].Target.Function, se1.Type,
					se1.Classification.Label, se2.Classification.Label)
			}
			if se1.Classification.Confidence != se2.Classification.Confidence {
				t.Errorf("SC-002: function %s effect %s: confidence differs: %d vs %d",
					c1[i].Target.Function, se1.Type,
					se1.Classification.Confidence, se2.Classification.Confidence)
			}
		}
	}
}

// TestSC003_IncidentalNotContractual verifies that the incidental
// fixture does not produce Contractual labels (SC-003).
func TestSC003_IncidentalNotContractual(t *testing.T) {
	allPkgs := loadTestPackages(t)
	incidentalPkg := findPackage(allPkgs, "incidental")
	if incidentalPkg == nil {
		t.Fatal("incidental package not found")
	}

	opts := analysis.Options{IncludeUnexported: true}
	results, err := analysis.Analyze(incidentalPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      incidentalPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				continue
			}
			// P0 effects receive a tier boost (+25, issue #71) and
			// are exempt from the incidental check — they are
			// definitionally part of a function's contract regardless
			// of the function's incidental nature. Negative signals
			// (naming, contradiction) may still bring the final
			// confidence below 75, so we do not assert on the value.
			if taxonomy.TierOf(se.Type) == taxonomy.TierP0 {
				t.Logf("SC-003: P0 effect %s on %s: confidence %d (exempt)",
					se.Type, result.Target.Function, se.Classification.Confidence)
				continue
			}
			if se.Classification.Label == taxonomy.Contractual {
				t.Errorf("SC-003: incidental function %s, non-P0 effect %s: "+
					"labeled Contractual (confidence %d)",
					result.Target.Function, se.Type,
					se.Classification.Confidence)
			}
		}
	}
}

// TestSC004_ConfigurableThresholds verifies that threshold overrides
// from config are respected (SC-004).
func TestSC004_ConfigurableThresholds(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Classification.Thresholds.Contractual = 95 // Very strict.
	cfg.Classification.Thresholds.Incidental = 40

	signals := []taxonomy.Signal{
		{Source: "interface", Weight: 30},
		{Source: "visibility", Weight: 10},
	}

	// 50 + 30 + 10 = 90. With threshold 95, should be Ambiguous.
	c := classify.ComputeScore("", signals, cfg)
	if c.Label == taxonomy.Contractual {
		t.Errorf("SC-004: with threshold=95, confidence=90 should not be Contractual; got %q",
			c.Label)
	}

	// Now with default threshold (80), same score = Contractual.
	cDefault := classify.ComputeScore("", signals, nil)
	if cDefault.Label != taxonomy.Contractual {
		t.Errorf("SC-004: with default threshold=80, confidence=90 should be Contractual; got %q",
			cDefault.Label)
	}
}

// TestSC005_SignalReasoningPopulated verifies that verbose mode
// populates signal reasoning fields (SC-005).
func TestSC005_SignalReasoningPopulated(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	signalFound := false
	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				continue
			}
			for _, sig := range se.Classification.Signals {
				if sig.Reasoning != "" {
					signalFound = true
				}
			}
		}
	}

	if !signalFound {
		t.Error("SC-005: verbose mode produced no signal reasoning — expected at least one")
	}
}

// TestSC006_NonVerboseStripsDetails verifies that non-verbose mode
// strips SourceFile, Excerpt, and Reasoning from signals (SC-006).
func TestSC006_NonVerboseStripsDetails(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
		Verbose:        false, // Non-verbose.
	}

	classified := classify.Classify(results, classifyOpts)

	for _, result := range classified {
		for _, se := range result.SideEffects {
			if se.Classification == nil {
				continue
			}
			for _, sig := range se.Classification.Signals {
				if sig.Reasoning != "" {
					t.Errorf("SC-006: non-verbose mode: signal %q has reasoning %q",
						sig.Source, sig.Reasoning)
				}
				if sig.SourceFile != "" {
					t.Errorf("SC-006: non-verbose mode: signal %q has source_file %q",
						sig.Source, sig.SourceFile)
				}
				if sig.Excerpt != "" {
					t.Errorf("SC-006: non-verbose mode: signal %q has excerpt %q",
						sig.Source, sig.Excerpt)
				}
			}
		}
	}
}

// TestSC007_NewPrefixContractual verifies that a function named
// NewClient with a ReturnValue (P0) effect is classified as
// "contractual" with confidence >= 80 after adding the "New" prefix
// to the naming convention list (SC-001 from spec 035).
func TestSC007_NewPrefixContractual(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
		Verbose:        true,
	}

	classified := classify.Classify(results, classifyOpts)

	// Find NewClient and verify its ReturnValue effect is contractual.
	found := false
	for _, result := range classified {
		if result.Target.Function != "NewClient" {
			continue
		}
		found = true
		for _, se := range result.SideEffects {
			if se.Type != taxonomy.ReturnValue {
				continue
			}
			if se.Classification == nil {
				t.Fatal("SC-007: NewClient ReturnValue has no classification")
			}
			if se.Classification.Label != taxonomy.Contractual {
				t.Errorf("SC-007: NewClient ReturnValue label = %q, want %q",
					se.Classification.Label, taxonomy.Contractual)
			}
			if se.Classification.Confidence < 80 {
				t.Errorf("SC-007: NewClient ReturnValue confidence = %d, want >= 80",
					se.Classification.Confidence)
			}
		}
	}

	if !found {
		t.Fatal("SC-007: NewClient function not found in classified results")
	}
}

// TestClassify_Determinism verifies that classifying the same
// package twice produces identical results (FR-011).
func TestClassify_Determinism(t *testing.T) {
	allPkgs := loadTestPackages(t)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		t.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results1, _ := analysis.Analyze(contractsPkg, opts)
	results2, _ := analysis.Analyze(contractsPkg, opts)

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
	}

	c1 := classify.Classify(results1, classifyOpts)
	c2 := classify.Classify(results2, classifyOpts)

	if len(c1) != len(c2) {
		t.Fatalf("determinism: result count differs: %d vs %d",
			len(c1), len(c2))
	}

	for i := range c1 {
		for j := range c1[i].SideEffects {
			se1 := c1[i].SideEffects[j]
			se2 := c2[i].SideEffects[j]

			if se1.Classification == nil || se2.Classification == nil {
				continue
			}

			if se1.Classification.Label != se2.Classification.Label {
				t.Errorf("determinism: function %s effect %s: "+
					"labels differ: %q vs %q",
					c1[i].Target.Function, se1.Type,
					se1.Classification.Label,
					se2.Classification.Label)
			}
			if se1.Classification.Confidence != se2.Classification.Confidence {
				t.Errorf("determinism: function %s effect %s: "+
					"confidence differs: %d vs %d",
					c1[i].Target.Function, se1.Type,
					se1.Classification.Confidence,
					se2.Classification.Confidence)
			}
		}
	}
}

// TestComputeScore_Issue105_LogAndComputeReturnValue is an integration
// test that verifies the full scoring path for the issue #105 scenario:
// a function named "LogAndCompute" with a ReturnValue effect, exported,
// with godoc containing only "logs" (no contractual keywords). Before
// the fix, incidental naming (-10) and godoc (-15) penalties plus the
// contradiction penalty (-20) overwhelmed the P0 tier boost (+25) and
// visibility (+8), yielding confidence 38 → incidental. After the fix,
// the naming and godoc incidental signals are skipped for ReturnValue,
// yielding confidence 83 → contractual.
func TestComputeScore_Issue105_LogAndComputeReturnValue(t *testing.T) {
	// Simulate the signals that classifySideEffect would produce for
	// an exported function "LogAndCompute" with ReturnValue effect and
	// godoc "LogAndCompute logs the request and computes a result."

	// After the fix:
	// - naming: "Log" prefix doesn't apply to ReturnValue → zero signal
	// - godoc: "logs" keyword doesn't apply to ReturnValue → zero signal
	// - visibility: exported function → +8
	// No naming/godoc incidental signals → no contradiction → no -20 penalty
	// Expected: base(50) + tierBoost(25) + visibility(8) = 83 → contractual
	signals := []taxonomy.Signal{
		// Visibility signal: exported function.
		{Source: "visibility", Weight: 8, Reasoning: "exported function"},
		// Naming: zero signal (type guard skipped incidental prefix).
		// Godoc: zero signal (type guard skipped incidental keyword).
	}

	result := classify.ComputeScore(taxonomy.ReturnValue, signals, nil)

	if result.Label != taxonomy.Contractual {
		t.Errorf("issue #105: LogAndCompute ReturnValue: label = %q, want %q",
			result.Label, taxonomy.Contractual)
	}
	if result.Confidence < 80 {
		t.Errorf("issue #105: LogAndCompute ReturnValue: confidence = %d, want >= 80",
			result.Confidence)
	}
	if result.Confidence != 83 {
		t.Errorf("issue #105: LogAndCompute ReturnValue: confidence = %d, want 83",
			result.Confidence)
	}

	// Verify no contradiction penalty was applied.
	for _, s := range result.Signals {
		if s.Source == "contradiction" {
			t.Error("issue #105: contradiction penalty should not be applied when " +
				"incidental signals are skipped by type guard")
		}
	}
}

// TestComputeScore_Issue105_LogAndComputeLogWrite is the counterpart:
// verify that I/O effects on the same function still get the incidental
// classification. LogWrite on "LogAndCompute" should still be penalized.
func TestComputeScore_Issue105_LogAndComputeLogWrite(t *testing.T) {
	// For LogWrite on LogAndCompute:
	// - naming: "Log" prefix applies to LogWrite → -10
	// - godoc: "logs" keyword applies to LogWrite → -15
	// - visibility: exported function → +8
	// Has both positive(+8) and negative(-10,-15) → contradiction(-20)
	// Expected: base(50) + tierBoost(0, P2) + 8 - 10 - 15 - 20 = 13 → incidental
	signals := []taxonomy.Signal{
		{Source: "visibility", Weight: 8, Reasoning: "exported function"},
		{Source: "naming", Weight: -10, Reasoning: "function name prefix Log*"},
		{Source: "godoc", Weight: -15, Reasoning: "godoc contains logs"},
	}

	result := classify.ComputeScore(taxonomy.LogWrite, signals, nil)

	if result.Label != taxonomy.Incidental {
		t.Errorf("LogAndCompute LogWrite: label = %q, want %q",
			result.Label, taxonomy.Incidental)
	}
}
