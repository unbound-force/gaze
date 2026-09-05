package protocol_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/unbound-force/gaze/internal/protocol"
)

// fakeBinaryPath is the path to the compiled fake_analyzer binary.
// Built once in TestMain.
var fakeBinaryPath string

func TestMain(m *testing.M) {
	// Build the fake analyzer binary into a temp directory.
	tmpDir, err := os.MkdirTemp("", "gaze-protocol-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	fakeBinaryPath = filepath.Join(tmpDir, "fake_analyzer")
	cmd := exec.Command("go", "build", "-o", fakeBinaryPath, "./testdata/fake_analyzer/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building fake_analyzer: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// TestFullSession verifies a successful full protocol session:
// initialize → analyze → complexity → coverage → shutdown.
func TestFullSession(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// 1. Initialize
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	var initResult protocol.InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if initResult.AnalyzerName != "fake-analyzer" {
		t.Errorf("analyzer_name = %q, want %q", initResult.AnalyzerName, "fake-analyzer")
	}
	if initResult.Language != "python" {
		t.Errorf("language = %q, want %q", initResult.Language, "python")
	}
	if initResult.ProtocolVersion != "1.1.0" {
		t.Errorf("protocol_version = %q, want %q", initResult.ProtocolVersion, "1.1.0")
	}
	if !initResult.Capabilities.Discover {
		t.Error("capabilities.discover = false, want true")
	}
	if !initResult.Capabilities.TestMapping {
		t.Error("capabilities.test_mapping = false, want true")
	}
	if !initResult.Capabilities.ClassifySignals {
		t.Error("capabilities.classify_signals = false, want true")
	}

	// 2. Analyze
	resp, err = client.Call(ctx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("analyze returned error: %s", resp.Error.Message)
	}

	var analyzeResult protocol.AnalyzeResult
	if err := json.Unmarshal(resp.Result, &analyzeResult); err != nil {
		t.Fatalf("unmarshal analyze result: %v", err)
	}
	if len(analyzeResult.Functions) != 3 {
		t.Fatalf("analyze returned %d functions, want 3", len(analyzeResult.Functions))
	}

	// Verify divide has 2 side effects (ReturnValue + ErrorReturn).
	divideFunc := findFunction(analyzeResult.Functions, "divide")
	if divideFunc == nil {
		t.Fatal("analyze: divide function not found")
	}
	if len(divideFunc.SideEffects) != 2 {
		t.Errorf("divide has %d side effects, want 2", len(divideFunc.SideEffects))
	}

	// Verify multiply has 1 side effect (ReturnValue).
	multiplyFunc := findFunction(analyzeResult.Functions, "multiply")
	if multiplyFunc == nil {
		t.Fatal("analyze: multiply function not found")
	}
	if len(multiplyFunc.SideEffects) != 1 {
		t.Errorf("multiply has %d side effects, want 1", len(multiplyFunc.SideEffects))
	}

	// 3. Complexity
	resp, err = client.Call(ctx, protocol.MethodComplexity, protocol.ComplexityParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("complexity: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("complexity returned error: %s", resp.Error.Message)
	}

	var complexityResult protocol.ComplexityResult
	if err := json.Unmarshal(resp.Result, &complexityResult); err != nil {
		t.Fatalf("unmarshal complexity result: %v", err)
	}
	if len(complexityResult.Functions) != 3 {
		t.Fatalf("complexity returned %d functions, want 3", len(complexityResult.Functions))
	}

	// Verify expected complexity values.
	wantComplexity := map[string]int{"add": 2, "multiply": 3, "divide": 5}
	for _, f := range complexityResult.Functions {
		if want, ok := wantComplexity[f.Name]; ok {
			if f.Complexity != want {
				t.Errorf("complexity(%s) = %d, want %d", f.Name, f.Complexity, want)
			}
		}
	}

	// 4. Coverage
	resp, err = client.Call(ctx, protocol.MethodCoverage, protocol.CoverageParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("coverage returned error: %s", resp.Error.Message)
	}

	var coverageResult protocol.CoverageResult
	if err := json.Unmarshal(resp.Result, &coverageResult); err != nil {
		t.Fatalf("unmarshal coverage result: %v", err)
	}
	if len(coverageResult.Functions) != 3 {
		t.Fatalf("coverage returned %d functions, want 3", len(coverageResult.Functions))
	}

	// Verify expected coverage values.
	wantCoverage := map[string]float64{"add": 90.0, "multiply": 60.0, "divide": 0.0}
	for _, f := range coverageResult.Functions {
		if want, ok := wantCoverage[f.Function]; ok {
			if f.Percentage != want {
				t.Errorf("coverage(%s) = %.1f%%, want %.1f%%", f.Function, f.Percentage, want)
			}
		}
	}

	// 5. Shutdown (via Close)
	if err := client.Close(); err != nil {
		// The fake analyzer exits with code 0 on shutdown, but
		// the process may have already exited. Accept either.
		t.Logf("Close returned: %v (acceptable)", err)
	}
}

// TestBinaryNotFound verifies that NewClient returns a clear error
// when the analyzer binary does not exist.
func TestBinaryNotFound(t *testing.T) {
	_, err := protocol.NewClient("nonexistent-analyzer-binary-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
	// The error should mention the binary name.
	if got := err.Error(); !containsAll(got, "nonexistent-analyzer-binary-xyz", "not found") {
		t.Errorf("error = %q, want it to mention binary name and 'not found'", got)
	}
}

// TestCrashMidSession verifies that the client detects when the
// analyzer subprocess exits unexpectedly after responding to
// initialize.
func TestCrashMidSession(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--crash-after=initialize")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Initialize should succeed.
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	// Next call should fail because the analyzer crashed.
	_, err = client.Call(ctx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err == nil {
		t.Fatal("expected error after analyzer crash, got nil")
	}
	t.Logf("crash error: %v", err)
}

// TestMalformedJSON verifies that the client returns a parse error
// when the analyzer returns invalid JSON.
func TestMalformedJSON(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--malformed-json")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Initialize should succeed.
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	// Next call should fail with a JSON parse error.
	_, err = client.Call(ctx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
	if got := err.Error(); !containsAll(got, "parsing", "response JSON") {
		t.Errorf("error = %q, want it to mention JSON parsing", got)
	}
	t.Logf("malformed JSON error: %v", err)
}

// TestErrorResponse verifies that JSON-RPC error responses are
// correctly propagated through the Response.Error field.
func TestErrorResponse(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--error-response")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Initialize should succeed.
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	// Next call should return a JSON-RPC error response.
	resp, err = client.Call(ctx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("expected JSON-RPC error in response, got transport error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error in response, got nil")
	}
	if resp.Error.Code != -32603 {
		t.Errorf("error code = %d, want -32603", resp.Error.Code)
	}
	if resp.Error.Message != "internal error: simulated failure" {
		t.Errorf("error message = %q, want %q", resp.Error.Message, "internal error: simulated failure")
	}
}

// TestTimeout verifies that the client kills the subprocess and
// returns a timeout error when the context deadline expires.
func TestTimeout(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--hang")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Initialize should succeed (hang happens after initialize).
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	// Next call with a short timeout should fail.
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = client.Call(timeoutCtx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if got := err.Error(); !containsAll(got, "context deadline exceeded") {
		t.Errorf("error = %q, want it to mention context deadline exceeded", got)
	}
	t.Logf("timeout error: %v", err)
}

// TestDiscoverAndTestMapping verifies that optional methods
// (discover, test_mapping) work correctly.
func TestDiscoverAndTestMapping(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Initialize.
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %s", resp.Error.Message)
	}

	// Discover.
	resp, err = client.Call(ctx, protocol.MethodDiscover, protocol.DiscoverParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("discover returned error: %s", resp.Error.Message)
	}

	var discoverResult protocol.DiscoverResult
	if err := json.Unmarshal(resp.Result, &discoverResult); err != nil {
		t.Fatalf("unmarshal discover result: %v", err)
	}
	if len(discoverResult.SourceFiles) != 2 {
		t.Errorf("discover returned %d source files, want 2", len(discoverResult.SourceFiles))
	}
	if discoverResult.Framework != "pytest" {
		t.Errorf("framework = %q, want %q", discoverResult.Framework, "pytest")
	}

	// TestMapping.
	resp, err = client.Call(ctx, protocol.MethodTestMapping, protocol.TestMappingParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("test_mapping: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("test_mapping returned error: %s", resp.Error.Message)
	}

	var mappingResult protocol.TestMappingResult
	if err := json.Unmarshal(resp.Result, &mappingResult); err != nil {
		t.Fatalf("unmarshal test_mapping result: %v", err)
	}
	if len(mappingResult.Mappings) != 3 {
		t.Fatalf("test_mapping returned %d mappings, want 3", len(mappingResult.Mappings))
	}
	if mappingResult.Mappings[0].TestFunction != "test_multiply" {
		t.Errorf("mapping[0] test_function = %q, want %q", mappingResult.Mappings[0].TestFunction, "test_multiply")
	}
}

// findFunction searches for a function by name in the analyze result.
func findFunction(functions []protocol.AnalyzedFunction, name string) *protocol.AnalyzedFunction {
	for i := range functions {
		if functions[i].Name == name {
			return &functions[i]
		}
	}
	return nil
}

// TestCallStream_ReturnsScanner verifies that CallStream sends the
// request and returns a scanner for reading JSONL lines from the
// analyzer's stdout. Uses --crash-after=analyze/stream to ensure
// the analyzer exits after writing stream data, causing EOF.
func TestCallStream_ReturnsScanner(t *testing.T) {
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--hang-stream", "--crash-after=analyze/stream")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Initialize first.
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

	// Verify streaming capability is declared.
	var initResult protocol.InitializeResult
	if uerr := json.Unmarshal(resp.Result, &initResult); uerr != nil {
		t.Fatalf("unmarshal: %v", uerr)
	}
	if !initResult.Capabilities.Streaming {
		t.Fatal("expected streaming=true with --hang-stream flag")
	}

	// Call analyze/stream.
	scanner, err := client.CallStream(ctx, protocol.MethodAnalyzeStream, protocol.AnalyzeParams{
		RootPath: "/tmp/project",
		Patterns: []string{"./..."},
	})
	if err != nil {
		t.Fatalf("CallStream: %v", err)
	}

	// Read JSONL lines. The fake analyzer exits after writing
	// stream data (--crash-after=analyze/stream), so scanner.Scan()
	// will return false at EOF.
	var lines int
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Verify each line is valid JSON.
		var obj map[string]interface{}
		if uerr := json.Unmarshal(line, &obj); uerr != nil {
			t.Errorf("line %d: invalid JSON: %v", lines+1, uerr)
		}
		lines++
	}
	if lines == 0 {
		t.Error("expected at least one JSONL line from streaming response")
	}
	if lines != 3 {
		t.Errorf("expected 3 JSONL lines (add, multiply, divide), got %d", lines)
	}
}

// containsAll checks that s contains all of the given substrings.
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
