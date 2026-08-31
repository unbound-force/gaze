package adapter

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/unbound-force/gaze/internal/protocol"
)

// callTestBinaryPath is the path to the compiled fake_analyzer binary used
// by the internal call_test.go tests. It is built once, lazily, because the
// external-package TestMain (in adapter_test.go, package adapter_test) builds
// its own copy in an unexported var that this internal test package cannot
// reference.
var (
	callTestBinaryPath string
	callTestBuildOnce  sync.Once
	callTestBuildErr   error
)

// buildCallTestFakeAnalyzer builds the fake analyzer binary once and returns
// its path. Subsequent calls return the cached path.
func buildCallTestFakeAnalyzer(t *testing.T) string {
	t.Helper()
	callTestBuildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "gaze-call-test-*")
		if err != nil {
			callTestBuildErr = err
			return
		}
		binPath := filepath.Join(tmpDir, "fake_analyzer")
		cmd := exec.Command("go", "build", "-o", binPath, "./testdata/fake_analyzer/")
		cmd.Dir = filepath.Join("..", "protocol")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			callTestBuildErr = err
			return
		}
		callTestBinaryPath = binPath
	})
	if callTestBuildErr != nil {
		t.Fatalf("building fake_analyzer: %v", callTestBuildErr)
	}
	return callTestBinaryPath
}

// newCallTestClient starts the fake analyzer with the given extra args (after
// --stdio) and returns an initialized client. The --error-response and
// --malformed-json fake modes only fire on the first non-initialize request,
// so callers must drive the helper with a non-initialize method.
func newCallTestClient(t *testing.T, extraArgs ...string) *protocol.Client {
	t.Helper()
	binPath := buildCallTestFakeAnalyzer(t)
	args := append([]string{"--stdio"}, extraArgs...)
	client, err := protocol.NewClient(binPath, args...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Perform the initialize handshake so the fake analyzer is "past
	// initialize" and the --error-response / --malformed-json guards apply
	// to the subsequent helper call.
	ctx := context.Background()
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		_ = client.Close()
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		_ = client.Close()
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}
	return client
}

// mismatchedResult intentionally types the "functions" field as a string so
// that unmarshalling a real complexity result (where "functions" is a JSON
// array) into it fails inside the helper's json.Unmarshal — exercising the
// result-unmarshal error path while keeping the JSON-RPC envelope valid.
type mismatchedResult struct {
	Functions string `json:"functions"`
}

// TestCallAndUnmarshal exercises the generic helper against every error
// condition and the happy path, driven through the fake analyzer binary
// (the only way to construct a *protocol.Client, which spawns a subprocess).
func TestCallAndUnmarshal(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		client := newCallTestClient(t)
		defer func() { _ = client.Close() }()

		result, err := callAndUnmarshal[protocol.ComplexityResult](
			context.Background(), client, protocol.MethodComplexity,
			protocol.ComplexityParams{RootPath: "/tmp/project", Patterns: []string{"./..."}},
		)
		if err != nil {
			t.Fatalf("callAndUnmarshal: unexpected error: %v", err)
		}
		if len(result.Functions) != 3 {
			t.Fatalf("got %d functions, want 3", len(result.Functions))
		}
		// Assert specific field values from the canned data.
		want := map[string]int{"add": 2, "multiply": 3, "divide": 5}
		for _, fn := range result.Functions {
			exp, ok := want[fn.Name]
			if !ok {
				t.Errorf("unexpected function %q", fn.Name)
				continue
			}
			if fn.Complexity != exp {
				t.Errorf("%s complexity = %d, want %d", fn.Name, fn.Complexity, exp)
			}
		}
	})

	t.Run("TransportError", func(t *testing.T) {
		// --crash-after=complexity makes the subprocess exit after responding,
		// but the crash happens after the response is written. To force a
		// transport error we crash after initialize so the complexity Call
		// fails reading a response from a dead subprocess.
		client := newCallTestClient(t, "--crash-after=initialize")
		defer func() { _ = client.Close() }()

		_, err := callAndUnmarshal[protocol.ComplexityResult](
			context.Background(), client, protocol.MethodComplexity,
			protocol.ComplexityParams{RootPath: "/tmp/project", Patterns: []string{"./..."}},
		)
		if err == nil {
			t.Fatal("expected transport error, got nil")
		}
		if !strings.Contains(err.Error(), "complexity protocol call") {
			t.Errorf("error %q does not contain method prefix %q", err.Error(), "complexity protocol call")
		}
		// Transport errors wrap with %w — the wrapped error must be reachable.
		if errors.Unwrap(err) == nil {
			t.Errorf("transport error %q should wrap the underlying error with %%w", err.Error())
		}
	})

	t.Run("ProtocolError", func(t *testing.T) {
		client := newCallTestClient(t, "--error-response")
		defer func() { _ = client.Close() }()

		_, err := callAndUnmarshal[protocol.ComplexityResult](
			context.Background(), client, protocol.MethodComplexity,
			protocol.ComplexityParams{RootPath: "/tmp/project", Patterns: []string{"./..."}},
		)
		if err == nil {
			t.Fatal("expected protocol error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "complexity protocol error") {
			t.Errorf("error %q does not contain method prefix %q", msg, "complexity protocol error")
		}
		// The fake analyzer returns message "internal error: simulated failure"
		// with code -32603.
		if !strings.Contains(msg, "internal error: simulated failure") {
			t.Errorf("error %q does not contain protocol error message", msg)
		}
		if !strings.Contains(msg, "-32603") {
			t.Errorf("error %q does not contain protocol error code -32603", msg)
		}
	})

	t.Run("UnmarshalFailure", func(t *testing.T) {
		// The fake analyzer's --malformed-json mode corrupts the JSON-RPC
		// envelope itself, which fails inside client.Call (a transport error),
		// not inside the helper's result unmarshal. To exercise the helper's
		// own json.Unmarshal failure path we request a well-formed response
		// (complexity) but decode it into a type whose "functions" field is
		// incompatible (a string instead of an array), forcing json.Unmarshal
		// of resp.Result to fail while the envelope stays valid.
		client := newCallTestClient(t)
		defer func() { _ = client.Close() }()

		_, err := callAndUnmarshal[mismatchedResult](
			context.Background(), client, protocol.MethodComplexity,
			protocol.ComplexityParams{RootPath: "/tmp/project", Patterns: []string{"./..."}},
		)
		if err == nil {
			t.Fatal("expected unmarshal failure, got nil")
		}
		if !strings.Contains(err.Error(), "parsing complexity result") {
			t.Errorf("error %q does not contain method prefix %q", err.Error(), "parsing complexity result")
		}
		// Unmarshal errors wrap with %w.
		if errors.Unwrap(err) == nil {
			t.Errorf("unmarshal error %q should wrap the underlying error with %%w", err.Error())
		}
	})

	// GenericInstantiation exercises the helper with a second, distinct result
	// type to verify the generic is not coupled to a single concrete type.
	t.Run("GenericInstantiation", func(t *testing.T) {
		client := newCallTestClient(t)
		defer func() { _ = client.Close() }()

		result, err := callAndUnmarshal[protocol.CoverageResult](
			context.Background(), client, protocol.MethodCoverage,
			protocol.CoverageParams{RootPath: "/tmp/project", Patterns: []string{"./..."}},
		)
		if err != nil {
			t.Fatalf("callAndUnmarshal[CoverageResult]: unexpected error: %v", err)
		}
		if len(result.Functions) != 3 {
			t.Fatalf("got %d functions, want 3", len(result.Functions))
		}
		// Assert specific field values from the second type's canned data.
		want := map[string]float64{"add": 90.0, "multiply": 60.0, "divide": 0.0}
		for _, fn := range result.Functions {
			exp, ok := want[fn.Function]
			if !ok {
				t.Errorf("unexpected function %q", fn.Function)
				continue
			}
			if fn.Percentage != exp {
				t.Errorf("%s coverage = %g, want %g", fn.Function, fn.Percentage, exp)
			}
		}
	})
}
