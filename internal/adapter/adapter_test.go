package adapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/protocol"
)

// fakeBinaryPath is the path to the compiled fake_analyzer binary.
// Built once in TestMain from the protocol testdata.
var fakeBinaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "gaze-adapter-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	fakeBinaryPath = filepath.Join(tmpDir, "fake_analyzer")

	// Build the fake analyzer from the protocol testdata directory.
	cmd := exec.Command("go", "build", "-o", fakeBinaryPath,
		"./testdata/fake_analyzer/")
	cmd.Dir = filepath.Join("..", "protocol")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building fake_analyzer: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// TestExternalComplexityProvider verifies that the complexity adapter
// correctly translates protocol responses to crap.FunctionComplexity.
func TestExternalComplexityProvider(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	mustInitialize(t, client)

	provider := adapter.NewExternalComplexityProvider(client)
	results, err := provider.Analyze([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d functions, want 3", len(results))
	}

	// Verify canned data: add(2), multiply(3), divide(5).
	want := map[string]int{
		"add":      2,
		"multiply": 3,
		"divide":   5,
	}
	for _, r := range results {
		expected, ok := want[r.Function]
		if !ok {
			t.Errorf("unexpected function %q", r.Function)
			continue
		}
		if r.Complexity != expected {
			t.Errorf("%s complexity = %d, want %d", r.Function, r.Complexity, expected)
		}
		if r.Package != "math_utils" {
			t.Errorf("%s package = %q, want %q", r.Function, r.Package, "math_utils")
		}
	}
}

// TestExternalLineCoverageProvider verifies that the coverage adapter
// correctly translates protocol responses to crap.FuncCoverage.
func TestExternalLineCoverageProvider(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	mustInitialize(t, client)

	provider := adapter.NewExternalLineCoverageProvider(client)
	results, err := provider.Coverage([]string{"./..."}, "/tmp/project", "")
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d functions, want 3", len(results))
	}

	// Verify canned data: add(90%), multiply(60%), divide(0%).
	want := map[string]float64{
		"add":      90.0,
		"multiply": 60.0,
		"divide":   0.0,
	}
	for _, r := range results {
		expected, ok := want[r.FuncName]
		if !ok {
			t.Errorf("unexpected function %q", r.FuncName)
			continue
		}
		if r.Percentage != expected {
			t.Errorf("%s coverage = %g%%, want %g%%", r.FuncName, r.Percentage, expected)
		}
	}
}

// TestExternalSideEffectAnalyzer verifies that the side effect
// adapter correctly translates protocol responses to
// taxonomy.AnalysisResult with Classification attached.
func TestExternalSideEffectAnalyzer(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	var stderr bytes.Buffer
	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, &stderr, nil,
	)

	results, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Canned data: divide (2 effects), multiply (1 effect), add (0 effects).
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Find divide — should have ReturnValue and ErrorReturn.
	var divideEffects int
	var multiplyEffects int
	for _, r := range results {
		switch r.Target.Function {
		case "divide":
			divideEffects = len(r.SideEffects)
			// Verify classification is attached.
			for _, e := range r.SideEffects {
				if e.Classification == nil {
					t.Errorf("divide effect %s has nil classification", e.Type)
				}
			}
		case "multiply":
			multiplyEffects = len(r.SideEffects)
		case "add":
			if len(r.SideEffects) != 0 {
				t.Errorf("add has %d effects, want 0", len(r.SideEffects))
			}
		}
	}

	if divideEffects != 2 {
		t.Errorf("divide has %d effects, want 2", divideEffects)
	}
	if multiplyEffects != 1 {
		t.Errorf("multiply has %d effects, want 1", multiplyEffects)
	}
}

// TestExternalSideEffectAnalyzer_FiltersByPackage verifies that
// Analyze filters results by package path.
func TestExternalSideEffectAnalyzer_FiltersByPackage(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, nil, nil,
	)

	// Query a non-existent package — should return empty.
	results, err := analyzer.Analyze("nonexistent_pkg")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results for nonexistent package, want 0", len(results))
	}
}

// TestSessionLifecycle verifies the full session lifecycle:
// spawn → initialize → providers ready → close.
func TestSessionLifecycle(t *testing.T) {
	var stderr bytes.Buffer
	session := adapter.NewSession(
		fakeBinaryPath, []string{"--stdio"},
		"/tmp/project", []string{"./..."},
		&stderr, nil,
	)

	providers, err := session.Initialize()
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer func() { _ = session.Close() }()

	if providers.AnalyzerName != "fake-analyzer" {
		t.Errorf("AnalyzerName = %q, want %q", providers.AnalyzerName, "fake-analyzer")
	}
	if providers.Language != "python" {
		t.Errorf("Language = %q, want %q", providers.Language, "python")
	}
	if providers.Complexity == nil {
		t.Error("Complexity provider is nil")
	}
	if providers.LineCoverage == nil {
		t.Error("LineCoverage provider is nil")
	}
	// Fake analyzer has test_mapping: true, so ContractCoverage
	// should be non-nil.
	if providers.ContractCoverage == nil {
		t.Error("ContractCoverage provider is nil (test_mapping is true)")
	}
	if !providers.Capabilities.TestMapping {
		t.Error("Capabilities.TestMapping = false, want true")
	}
	if !providers.Capabilities.Discover {
		t.Error("Capabilities.Discover = false, want true")
	}
	if !providers.Capabilities.ClassifySignals {
		t.Error("Capabilities.ClassifySignals = false, want true")
	}
}

// TestSessionLifecycle_NoTestMapping verifies that when test_mapping
// capability is false, ContractCoverage is nil.
func TestSessionLifecycle_NoTestMapping(t *testing.T) {
	// The fake analyzer has test_mapping: true by default.
	// We test the session's behavior by verifying the provider
	// construction logic. Since we can't easily change the fake's
	// capabilities, we verify the positive case above and test
	// the no-op path through the contract provider directly.

	// Create a client and get capabilities.
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	// Use capabilities with test_mapping: false.
	caps := protocol.Capabilities{
		Discover:        true,
		TestMapping:     false,
		ClassifySignals: false,
	}

	provider := adapter.NewExternalContractCoverageProvider(
		client, caps, nil, "/tmp/project", []string{"./..."}, nil,
	)

	lookup, degraded, err := provider.Build([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if degraded != nil {
		t.Errorf("degraded = %v, want nil", degraded)
	}

	// No-op lookup should return (zero, false).
	info, ok := lookup("math_utils", "divide")
	if ok {
		t.Error("no-op lookup returned ok=true, want false")
	}
	if info.Percentage != 0 {
		t.Errorf("no-op lookup percentage = %g, want 0", info.Percentage)
	}
}

// TestContractCoverageProvider_WithTestMapping verifies that when
// test_mapping is supported, the provider calls the method and
// computes contract coverage.
func TestContractCoverageProvider_WithTestMapping(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	sideEffects := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, nil, nil,
	)

	provider := adapter.NewExternalContractCoverageProvider(
		client, caps, sideEffects,
		"/tmp/project", []string{"./..."}, nil,
	)

	lookup, _, err := provider.Build([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// The fake analyzer maps test_multiply → multiply ReturnValue.
	// multiply has 1 contractual effect (ReturnValue), which is
	// mapped, so contract coverage should be 100%.
	info, ok := lookup("math_utils", "multiply")
	if !ok {
		t.Fatal("lookup returned ok=false for multiply")
	}
	if info.Percentage != 100.0 {
		t.Errorf("multiply contract coverage = %g%%, want 100%%", info.Percentage)
	}

	// divide has 2 contractual effects (ReturnValue and ErrorReturn).
	// The fake analyzer maps test_divide_basic → ReturnValue and
	// test_divide_error → ErrorReturn, so both effects are covered → 100%.
	info, ok = lookup("math_utils", "divide")
	if !ok {
		t.Fatal("lookup returned ok=false for divide")
	}
	if info.Percentage != 100.0 {
		t.Errorf("divide contract coverage = %g%%, want 100%%", info.Percentage)
	}
}

// TestSessionClose_BeforeInitialize verifies that Close is safe
// to call before Initialize.
func TestSessionClose_BeforeInitialize(t *testing.T) {
	session := adapter.NewSession(
		fakeBinaryPath, []string{"--stdio"},
		"/tmp/project", []string{"./..."}, nil, nil,
	)

	// Close without Initialize — should not panic or error.
	if err := session.Close(); err != nil {
		t.Errorf("Close before Initialize: %v", err)
	}
}

// TestSession_BinaryNotFound verifies error when analyzer binary
// does not exist.
func TestSession_BinaryNotFound(t *testing.T) {
	session := adapter.NewSession(
		"/nonexistent/analyzer", []string{"--stdio"},
		"/tmp/project", []string{"./..."}, nil, nil,
	)

	_, err := session.Initialize()
	if err == nil {
		t.Fatal("Initialize with nonexistent binary should fail")
	}
}

// TestErrorPropagation_ComplexityProtocolError verifies that
// protocol errors from required methods are propagated.
func TestErrorPropagation_ComplexityProtocolError(t *testing.T) {
	// Build a fake that returns errors after initialize.
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--error-response")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Initialize succeeds.
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

	// Complexity call should get the error response.
	provider := adapter.NewExternalComplexityProvider(client)
	_, err = provider.Analyze([]string{"./..."}, "/tmp/project")
	if err == nil {
		t.Fatal("Analyze should fail with error response")
	}
}

// --- helpers ---

func mustNewClient(t *testing.T) *protocol.Client {
	t.Helper()
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func mustInitialize(t *testing.T, client *protocol.Client) protocol.Capabilities {
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

// TestExternalSideEffectAnalyzer_ClassifySignals verifies that when
// classify_signals capability is true, the adapter calls classify_signals
// after analyze and merges the returned signals into cached effects.
// Effects with pre-existing classifications (from the analyze response)
// are preserved; effects without classification get classified via
// ComputeScore.
func TestExternalSideEffectAnalyzer_ClassifySignals(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)
	if !caps.ClassifySignals {
		t.Fatal("expected ClassifySignals=true from fake analyzer")
	}

	var stderr bytes.Buffer
	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, &stderr, nil,
	)

	// First call triggers loadAll → analyze + classify_signals.
	results, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Verify classify_signals was called (no stderr warnings about failure).
	if strings.Contains(stderr.String(), "classify_signals failed") {
		t.Errorf("unexpected classify_signals failure: %s", stderr.String())
	}

	// Find multiply — its ReturnValue effect has a pre-existing classification
	// from the analyze response (confidence 95, contractual). The classify_signals
	// response sends a signal with weight -15 for it, but since the effect is
	// already classified, mergeClassifications should preserve the original.
	for _, r := range results {
		if r.Target.Function == "multiply" {
			if len(r.SideEffects) != 1 {
				t.Fatalf("multiply has %d effects, want 1", len(r.SideEffects))
			}
			e := r.SideEffects[0]
			if e.Classification == nil {
				t.Fatal("multiply ReturnValue classification is nil after classify_signals")
			}
			// Pre-classified: should retain original confidence 95.
			if e.Classification.Confidence != 95 {
				t.Errorf("multiply ReturnValue confidence = %d, want 95 (pre-classified preserved)",
					e.Classification.Confidence)
			}
		}
	}
}

// TestExternalSideEffectAnalyzer_ClassifySignals_CacheConsistency verifies
// that a second Analyze() call returns the same classified data from the
// cache without re-calling classify_signals. The "no pre-classification"
// path (where classify_signals signals are scored through ComputeScore on
// unclassified effects) is tested via unit tests in classify_test.go, since
// the fake analyzer's analyze response always includes pre-classifications.
func TestExternalSideEffectAnalyzer_ClassifySignals_CacheConsistency(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	// Verify multi-call cache: a second Analyze() call returns the same
	// classified data without re-calling classify_signals.
	var stderr bytes.Buffer
	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, &stderr, nil,
	)

	// First call loads and classifies.
	results1, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("first Analyze: %v", err)
	}

	// Second call for same package — uses cache.
	results2, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("second Analyze: %v", err)
	}

	if len(results1) != len(results2) {
		t.Fatalf("result count mismatch: first=%d, second=%d", len(results1), len(results2))
	}

	// Verify classification state is identical (cached, not re-classified).
	for i := range results1 {
		if results1[i].Target.Function != results2[i].Target.Function {
			t.Errorf("result[%d] function mismatch: %q vs %q",
				i, results1[i].Target.Function, results2[i].Target.Function)
		}
		for j := range results1[i].SideEffects {
			c1 := results1[i].SideEffects[j].Classification
			c2 := results2[i].SideEffects[j].Classification
			if (c1 == nil) != (c2 == nil) {
				t.Errorf("result[%d] effect[%d] classification nil mismatch", i, j)
			}
			if c1 != nil && c2 != nil && c1.Confidence != c2.Confidence {
				t.Errorf("result[%d] effect[%d] confidence mismatch: %d vs %d",
					i, j, c1.Confidence, c2.Confidence)
			}
		}
	}
}

// TestExternalSideEffectAnalyzer_ClassifySignals_Disabled verifies that
// when classify_signals capability is false, no classify_signals call is
// made and effects without pre-classification remain unclassified.
func TestExternalSideEffectAnalyzer_ClassifySignals_Disabled(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	// Initialize to get protocol handshake done, but override caps.
	mustInitialize(t, client)

	// Use caps with ClassifySignals=false.
	caps := protocol.Capabilities{
		Discover:        true,
		TestMapping:     true,
		ClassifySignals: false,
	}

	var stderr bytes.Buffer
	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, &stderr, nil,
	)

	results, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// With ClassifySignals=false, the effects still have pre-classifications
	// from the analyze response (they're baked into the protocol data).
	// But no classify_signals call should have been made.
	if strings.Contains(stderr.String(), "classify_signals") {
		t.Errorf("unexpected classify_signals activity in stderr: %s", stderr.String())
	}

	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Verify pre-classified confidence values match the analyze response
	// exactly — proving they came from the analyze response, not from
	// classify_signals → ComputeScore. We check each effect individually
	// since some functions have multiple effects.
	type wantEffect struct {
		seType     string
		confidence int
	}
	wantConfidence := map[string][]wantEffect{
		"multiply": {{"ReturnValue", 95}},
		"divide":   {{"ReturnValue", 90}, {"ErrorReturn", 85}},
		// add has no side effects
	}
	for _, r := range results {
		wants, ok := wantConfidence[r.Target.Function]
		if !ok {
			continue
		}
		for _, w := range wants {
			for _, e := range r.SideEffects {
				if string(e.Type) != w.seType {
					continue
				}
				if e.Classification == nil {
					t.Errorf("%s/%s: expected pre-classification from analyze response, got nil",
						r.Target.Function, w.seType)
					continue
				}
				if e.Classification.Confidence != w.confidence {
					t.Errorf("%s/%s: confidence = %d, want %d (from analyze response, not ComputeScore)",
						r.Target.Function, w.seType, e.Classification.Confidence, w.confidence)
				}
			}
		}
	}
}

// TestExternalSideEffectAnalyzer_Streaming verifies that the streaming
// adapter produces the same results as the batch adapter.
func TestExternalSideEffectAnalyzer_Streaming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: spawns external process")
	}

	// Build batch results first.
	batchClient := mustNewClient(t)
	batchCaps := mustInitialize(t, batchClient)
	if batchCaps.Streaming {
		t.Fatal("expected batch mode (streaming=false) from default fake analyzer")
	}

	batchAnalyzer := adapter.NewExternalSideEffectAnalyzer(
		batchClient, batchCaps,
		"/tmp/project", []string{"./..."},
		&bytes.Buffer{}, nil,
	)
	batchResults, err := batchAnalyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("batch analyze: %v", err)
	}
	_ = batchClient.Close()

	// Build streaming results with --hang-stream flag.
	// --crash-after=analyze/stream ensures the analyzer exits after
	// writing JSONL, causing EOF so the scanner terminates.
	streamClient := startFakeAnalyzerWithArgs(t, "--hang-stream", "--crash-after=analyze/stream")
	streamCaps := mustInitialize(t, streamClient)
	if !streamCaps.Streaming {
		t.Fatal("expected streaming=true with --hang-stream flag")
	}

	streamAnalyzer := adapter.NewExternalSideEffectAnalyzer(
		streamClient, streamCaps,
		"/tmp/project", []string{"./..."},
		&bytes.Buffer{}, nil,
	)
	streamResults, err := streamAnalyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("streaming analyze: %v", err)
	}
	_ = streamClient.Close()

	// Compare: streaming should produce same results as batch.
	if len(batchResults) != len(streamResults) {
		t.Fatalf("result count mismatch: batch=%d, stream=%d",
			len(batchResults), len(streamResults))
	}
	for i := range batchResults {
		if batchResults[i].Target.Function != streamResults[i].Target.Function {
			t.Errorf("result[%d] function mismatch: batch=%q, stream=%q",
				i, batchResults[i].Target.Function, streamResults[i].Target.Function)
		}
		if len(batchResults[i].SideEffects) != len(streamResults[i].SideEffects) {
			t.Errorf("result[%d] side effects count mismatch: batch=%d, stream=%d",
				i, len(batchResults[i].SideEffects), len(streamResults[i].SideEffects))
		}
	}
}

// TestDetectionConfidence_Integration verifies that
// ExternalContractCoverageProvider.Build computes and stores
// per-target-function detection confidence from mapping data, and
// that the values are accessible via DetectionConfidence.
func TestDetectionConfidence_Integration(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	sideEffects := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, nil, nil,
	)

	provider := adapter.NewExternalContractCoverageProvider(
		client, caps, sideEffects,
		"/tmp/project", []string{"./..."}, nil,
	)

	_, _, err := provider.Build([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// The fake analyzer returns 3 mappings:
	//   multiply: 1 mapping with assertion_type="equality" → 100%
	//   divide:   2 mappings, 1 with assertion_type="equality", 1 with "" → 50%
	if got := provider.DetectionConfidence("math_utils", "multiply"); got != 100 {
		t.Errorf("multiply DetectionConfidence = %d, want 100", got)
	}
	if got := provider.DetectionConfidence("math_utils", "divide"); got != 50 {
		t.Errorf("divide DetectionConfidence = %d, want 50", got)
	}

	// Unknown function returns 0.
	if got := provider.DetectionConfidence("math_utils", "unknown"); got != 0 {
		t.Errorf("unknown DetectionConfidence = %d, want 0", got)
	}
}

// startFakeAnalyzerWithArgs starts the fake analyzer binary with
// extra command-line arguments appended after --stdio.
func startFakeAnalyzerWithArgs(t *testing.T, extraArgs ...string) *protocol.Client {
	t.Helper()
	args := append([]string{"--stdio"}, extraArgs...)
	client, err := protocol.NewClient(fakeBinaryPath, args...)
	if err != nil {
		t.Fatalf("starting fake analyzer with args %v: %v", extraArgs, err)
	}
	return client
}
