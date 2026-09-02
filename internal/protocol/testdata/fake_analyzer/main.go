// Command fake_analyzer is a test double for external analyzer
// binaries used in protocol client integration tests.
//
// It reads JSON-RPC 2.0 requests from stdin (line-delimited JSON)
// and writes canned JSON-RPC responses to stdout. Supports all 9
// protocol methods with deterministic test data matching the spec:
//
//   - complexity: 3 functions (add/2, multiply/3, divide/5)
//   - coverage: 3 functions (add/90%, multiply/60%, divide/0%)
//   - analyze: divide has ReturnValue+ErrorReturn, multiply has ReturnValue
//   - doc_coverage: 3 symbols (divide/documented, multiply/documented, add/undocumented)
//
// Flags:
//
//	--stdio           Required flag (matches real analyzer convention)
//	--crash-after=M   Exit immediately after responding to method M
//	--hang            Sleep forever after responding to initialize
//	--malformed-json  Return invalid JSON for the first non-initialize request
//	--error-response  Return a JSON-RPC error for the first non-initialize request
//	--no-doc-coverage Disable doc_coverage capability in initialize response
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func main() {
	stdio := flag.Bool("stdio", false, "run in stdio mode")
	crashAfter := flag.String("crash-after", "", "crash after responding to this method")
	hang := flag.Bool("hang", false, "hang after initialize")
	hangStream := flag.Bool("hang-stream", false, "enable streaming mode and write JSONL for analyze/stream")
	malformedJSON := flag.Bool("malformed-json", false, "return malformed JSON for first non-initialize request")
	errorResponse := flag.Bool("error-response", false, "return JSON-RPC error for first non-initialize request")
	noDocCoverage := flag.Bool("no-doc-coverage", false, "disable doc_coverage capability in initialize response")
	flag.Parse()

	if !*stdio {
		fmt.Fprintln(os.Stderr, "fake_analyzer: --stdio flag is required")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Increase scanner buffer for large requests.
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	pastInitialize := false

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fake_analyzer: failed to parse request: %v\n", err)
			continue
		}

		// Handle --malformed-json: return garbage for first non-initialize request.
		if *malformedJSON && pastInitialize {
			fmt.Fprintln(os.Stdout, "{invalid json!!!")
			os.Stdout.Sync()
			*malformedJSON = false // only once
			continue
		}

		// Handle --error-response: return JSON-RPC error for first non-initialize request.
		if *errorResponse && pastInitialize {
			resp := response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &rpcError{
					Code:    -32603,
					Message: "internal error: simulated failure",
					Data:    "test error data",
				},
			}
			writeResponse(resp)
			*errorResponse = false // only once
			continue
		}

		// Handle analyze/stream: write JSONL lines instead of a single response.
		if req.Method == "analyze/stream" {
			writeStreamResponse(req.ID)
			if *crashAfter == "analyze/stream" {
				os.Exit(0)
			}
			continue
		}

		resp := handleRequest(req, *hangStream, *noDocCoverage)
		writeResponse(resp)

		if req.Method == "initialize" {
			pastInitialize = true
		}

		// Handle --crash-after: exit after responding to the specified method.
		if *crashAfter != "" && req.Method == *crashAfter {
			os.Exit(1)
		}

		// Handle --hang: sleep forever after initialize.
		if *hang && req.Method == "initialize" {
			time.Sleep(24 * time.Hour)
		}
	}
}

func handleRequest(req request, streaming, noDocCoverage bool) response {
	switch req.Method {
	case "initialize":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"capabilities": map[string]any{
					"discover":         true,
					"test_mapping":     true,
					"classify_signals": true,
					"streaming":        streaming,
					"doc_coverage":     !noDocCoverage,
				},
				"protocol_version":  "1.1.0",
				"analyzer_name":     "fake-analyzer",
				"language":          "python",
				"language_version":  "3.12.0",
			},
		}

	case "analyze":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"functions": []map[string]any{
					{
						"name":    "divide",
						"package": "math_utils",
						"file":    "math_utils/ops.py",
						"line":    20,
						"side_effects": []map[string]any{
							{
								"type":        "ReturnValue",
								"description": "returns division result",
								"location":    "math_utils/ops.py:25:5",
								"target":      "result",
								"classification": map[string]any{
									"label":      "contractual",
									"confidence": 90,
								},
							},
							{
								"type":        "ErrorReturn",
								"description": "raises ZeroDivisionError",
								"location":    "math_utils/ops.py:22:9",
								"target":      "ZeroDivisionError",
								"classification": map[string]any{
									"label":      "contractual",
									"confidence": 85,
								},
							},
						},
					},
					{
						"name":    "multiply",
						"package": "math_utils",
						"file":    "math_utils/ops.py",
						"line":    10,
						"side_effects": []map[string]any{
							{
								"type":        "ReturnValue",
								"description": "returns multiplication result",
								"location":    "math_utils/ops.py:12:5",
								"target":      "result",
								"classification": map[string]any{
									"label":      "contractual",
									"confidence": 95,
								},
							},
						},
					},
					{
						"name":         "add",
						"package":      "math_utils",
						"file":         "math_utils/ops.py",
						"line":         1,
						"side_effects": []map[string]any{},
					},
				},
			},
		}

	case "complexity":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"functions": []map[string]any{
					{"name": "add", "package": "math_utils", "file": "math_utils/ops.py", "line": 1, "complexity": 2},
					{"name": "multiply", "package": "math_utils", "file": "math_utils/ops.py", "line": 10, "complexity": 3},
					{"name": "divide", "package": "math_utils", "file": "math_utils/ops.py", "line": 20, "complexity": 5},
				},
			},
		}

	case "coverage":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"functions": []map[string]any{
					{"file": "math_utils/ops.py", "function": "add", "start_line": 1, "end_line": 3, "covered_stmts": 9, "total_stmts": 10, "percentage": 90.0},
					{"file": "math_utils/ops.py", "function": "multiply", "start_line": 10, "end_line": 15, "covered_stmts": 6, "total_stmts": 10, "percentage": 60.0},
					{"file": "math_utils/ops.py", "function": "divide", "start_line": 20, "end_line": 30, "covered_stmts": 0, "total_stmts": 10, "percentage": 0.0},
				},
			},
		}

	case "discover":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"source_files": []string{"math_utils/ops.py", "math_utils/helpers.py"},
				"test_files":   []string{"tests/test_ops.py"},
				"framework":    "pytest",
			},
		}

	case "test_mapping":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"mappings": []map[string]any{
					{
						"test_function":    "test_multiply",
						"test_file":        "tests/test_ops.py",
						"assertion_location": "tests/test_ops.py:10",
						"assertion_type":   "equality",
						"target_function":  "multiply",
						"target_package":   "math_utils",
						"side_effect_type": "ReturnValue",
						"confidence":       80,
					},
				},
			},
		}

	case "classify_signals":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"signals": []map[string]any{
					{
						"function":         "divide",
						"package":          "math_utils",
						"side_effect_type": "ErrorReturn",
						"source":           "docstring",
						"weight":           20,
						"reasoning":        "docstring mentions ZeroDivisionError",
					},
					{
						"function":         "multiply",
						"package":          "math_utils",
						"side_effect_type": "ReturnValue",
						"source":           "naming_convention",
						"weight":           -15,
						"reasoning":        "function name implies pure computation",
					},
				},
			},
		}

	case "doc_coverage":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"symbols": []map[string]any{
					{"name": "divide", "package": "math_utils", "file": "math_utils/ops.py", "line": 20, "kind": "function", "documented": true, "doc_snippet": "Divides two numbers."},
					{"name": "multiply", "package": "math_utils", "file": "math_utils/ops.py", "line": 10, "kind": "function", "documented": true, "doc_snippet": "Multiplies two numbers."},
					{"name": "add", "package": "math_utils", "file": "math_utils/ops.py", "line": 1, "kind": "function", "documented": false, "doc_snippet": ""},
				},
			},
		}

	case "shutdown":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}

	default:
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

// writeStreamResponse writes JSONL lines for analyze/stream.
// Each line is one AnalyzedFunction as a standalone JSON object.
func writeStreamResponse(id int64) {
	_ = id // not used in JSONL mode — each line is independent
	funcs := []map[string]any{
		{
			"name":    "divide",
			"package": "math_utils",
			"file":    "math_utils/ops.py",
			"line":    20,
			"side_effects": []map[string]any{
				{
					"type":        "ReturnValue",
					"description": "returns division result",
					"location":    "math_utils/ops.py:25:5",
					"target":      "result",
					"classification": map[string]any{
						"label":      "contractual",
						"confidence": 90,
					},
				},
				{
					"type":        "ErrorReturn",
					"description": "raises ZeroDivisionError",
					"location":    "math_utils/ops.py:22:9",
					"target":      "ZeroDivisionError",
					"classification": map[string]any{
						"label":      "contractual",
						"confidence": 85,
					},
				},
			},
		},
		{
			"name":    "multiply",
			"package": "math_utils",
			"file":    "math_utils/ops.py",
			"line":    10,
			"side_effects": []map[string]any{
				{
					"type":        "ReturnValue",
					"description": "returns multiplication result",
					"location":    "math_utils/ops.py:12:5",
					"target":      "result",
					"classification": map[string]any{
						"label":      "contractual",
						"confidence": 95,
					},
				},
			},
		},
		{
			"name":         "add",
			"package":      "math_utils",
			"file":         "math_utils/ops.py",
			"line":         1,
			"side_effects": []map[string]any{},
		},
	}

	for _, fn := range funcs {
		data, err := json.Marshal(fn)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fake_analyzer: failed to marshal stream line: %v\n", err)
			continue
		}
		fmt.Fprintln(os.Stdout, string(data))
		os.Stdout.Sync()
	}
}

func writeResponse(resp response) {
	data, err := json.Marshal(resp)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fake_analyzer: failed to marshal response: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(data))
	os.Stdout.Sync()
}
