# Gaze Analyzer Protocol Specification

**Version**: 1.1.0
**Transport**: JSON-RPC 2.0 over stdin/stdout

## Overview

The Gaze Analyzer Protocol enables external language analyzers to provide side effect detection, complexity analysis, and coverage data to Gaze. Gaze spawns the analyzer as a subprocess, communicates via JSON-RPC 2.0 over stdin/stdout, and uses the responses to compute CRAP scores, GazeCRAP scores, quadrant classifications, and fix strategies.

This protocol follows the same transport model as the Language Server Protocol (LSP): line-delimited JSON over stdin/stdout, with stderr reserved for diagnostics.

## Lifecycle

```text
gaze crap --analyzer snake-eyes ./src
|
+-- 1. Discover analyzer binary (CLI flag / .gaze.yaml / PATH)
+-- 2. Spawn: snake-eyes --stdio
+-- 3. initialize --> capabilities handshake
+-- 4. discover --> find source + test files (optional)
+-- 5. analyze --> detect side effects per function
+-- 6. complexity --> cyclomatic complexity per function
+-- 7. coverage --> line coverage per function
+-- 8. test_mapping --> map assertions to effects (optional)
+-- 9. classify_signals --> classification signals (optional)
+-- 10. doc_coverage --> documentation coverage (optional)
+-- 11. shutdown --> clean exit
+-- 12. Gaze computes CRAP, GazeCRAP, quadrants, fix strategies
```

### Discovery

Gaze finds analyzer binaries using a three-tier mechanism:

1. **CLI flag**: `--analyzer <binary>` -- explicit binary name or path
2. **Config file**: `.gaze.yaml` `analyzers.<language>.command`
3. **PATH convention**: `gaze-analyzer-<language>` on PATH

The `--language` flag determines which language key to look up in tiers 2 and 3. When `--analyzer` is specified directly, `--language` is optional.

## Message Format

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "analyze",
  "params": {
    "root_path": "/path/to/project",
    "patterns": ["./..."]
  }
}
```

### Response (success)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { ... }
}
```

### Response (error)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": "optional details"
  }
}
```

### Error Codes

Standard JSON-RPC 2.0 error codes:

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

## Methods

### Required Methods

Every analyzer **must** implement these 5 methods.

---

### `initialize`

Handshake method. Must be the first method called. Returns the analyzer's capabilities and identity.

**Timeout**: 30 seconds

**Request params**:

```json
{
  "root_path": "/absolute/path/to/project",
  "config": {}
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root_path` | string | yes | Absolute path to the project root |
| `config` | object | no | Analyzer-specific config from `.gaze.yaml` |

**Response result**:

```json
{
  "capabilities": {
    "discover": true,
    "test_mapping": true,
    "classify_signals": false,
    "streaming": false,
    "doc_coverage": false
  },
  "protocol_version": "1.1.0",
  "analyzer_name": "snake-eyes",
  "language": "python",
  "language_version": "3.12.0"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `capabilities.discover` | boolean | Supports the `discover` method |
| `capabilities.test_mapping` | boolean | Supports the `test_mapping` method |
| `capabilities.classify_signals` | boolean | Supports the `classify_signals` method |
| `capabilities.streaming` | boolean | Supports the `analyze/stream` method |
| `capabilities.doc_coverage` | boolean | Supports the `doc_coverage` method |
| `protocol_version` | string | Protocol version (semver) |
| `analyzer_name` | string | Human-readable analyzer name |
| `language` | string | Primary language (e.g., "python", "rust") |
| `language_version` | string | Runtime/compiler version (optional) |

---

### `analyze`

Detect side effects for all functions in the project.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "name": "divide",
      "package": "math_utils",
      "file": "math_utils/ops.py",
      "line": 20,
      "side_effects": [
        {
          "type": "ReturnValue",
          "description": "returns division result",
          "location": "math_utils/ops.py:25:5",
          "target": "result",
          "detail": {
            "python_type": "float"
          },
          "classification": {
            "label": "contractual",
            "confidence": 90
          }
        },
        {
          "type": "ErrorReturn",
          "description": "raises ZeroDivisionError",
          "location": "math_utils/ops.py:22:9",
          "target": "ZeroDivisionError",
          "detail": {
            "exception_class": "ZeroDivisionError"
          },
          "classification": {
            "label": "contractual",
            "confidence": 85
          }
        }
      ]
    }
  ]
}
```

#### Side Effect Types

Analyzers map their language concepts to Gaze's universal taxonomy of 48 canonical side effect types. Each type belongs to exactly one priority tier (P0–P4). Unknown types default to P4 tier with a warning.

##### P0 — Must Detect (6 types)

| Type | Tier | Definition | Go Mapping | Python Mapping |
|------|------|-----------|------------|----------------|
| `ReturnValue` | P0 | Function returns a value | `return x` | `return x` |
| `ErrorReturn` | P0 | Function signals an error | `return err` | `raise Exception` |
| `SentinelError` | P0 | Named error constant/value callers match against | `var ErrFoo = errors.New(...)` | `class FooError(Exception)` |
| `ReceiverMutation` | P0 | Mutates the receiver/self | `s.field = x` | `self.field = x` |
| `PointerArgMutation` | P0 | Mutates a parameter passed by reference | `*arg = x` | mutable arg mutation |
| `ErrorSignal` | P0 | Language-neutral error signaling mechanism | `return err` | `raise`, `throw` |

##### P1 — High Value (11 types)

| Type | Tier | Definition | Go Mapping | Python Mapping |
|------|------|-----------|------------|----------------|
| `SliceMutation` | P1 | Mutates a dynamic array/list parameter | `slice[i] = x` | `list[i] = x` |
| `MapMutation` | P1 | Mutates a dictionary/hash map parameter | `m[k] = v` | `dict[k] = v` |
| `GlobalMutation` | P1 | Writes to module-level / global state | `pkg.Var = x` | module-level write |
| `WriterOutput` | P1 | Writes to an injected output stream | `w.Write(data)` | `writer.write(data)` |
| `HTTPResponseWrite` | P1 | Writes to an HTTP response object | `w.WriteHeader(200)` | `response.write(data)` |
| `ChannelSend` | P1 | Sends a value to a concurrency channel/queue | `ch <- v` | `queue.put(v)` |
| `ChannelClose` | P1 | Closes a concurrency channel/queue | `close(ch)` | `queue.close()` |
| `DeferredReturnMutation` | P1 | Mutates a return value in a deferred/finally block | `defer func() { err = wrap(err) }()` | `finally: result = ...` |
| `GeneratorYield` | P1 | Yields a value from a generator/iterator | *(no direct equivalent)* | `yield x` |
| `ContainerMutation` | P1 | Mutates a generic container parameter (set, deque, etc.) | *(no direct equivalent)* | `set.add(x)`, `deque.append(x)` |
| `StreamOutput` | P1 | Writes to a generic output stream | `stream.Write(data)` | `stream.write(data)` |

##### P2 — Important (16 types)

| Type | Tier | Definition | Go Mapping | Python Mapping |
|------|------|-----------|------------|----------------|
| `FileSystemWrite` | P2 | Writes to filesystem | `os.WriteFile(...)` | `open().write()` |
| `FileSystemDelete` | P2 | Deletes a filesystem entry | `os.Remove(...)` | `os.remove(...)` |
| `FileSystemMeta` | P2 | Modifies filesystem metadata (permissions, timestamps) | `os.Chmod(...)` | `os.chmod(...)` |
| `DatabaseWrite` | P2 | Writes to a database | `db.Exec(...)` | `cursor.execute(...)` |
| `DatabaseTransaction` | P2 | Manages a database transaction | `tx.Commit()` | `conn.commit()` |
| `GoroutineSpawn` | P2 | Spawns a concurrent task | `go func(){}()` | `threading.Thread(...)` |
| `Panic` | P2 | Unrecoverable error / panic / abort | `panic(msg)` | `sys.exit(1)` |
| `CallbackInvocation` | P2 | Invokes a function parameter (callback, closure) | `fn()` | `callback()` |
| `LogWrite` | P2 | Writes to a logging system | `log.Printf(...)` | `logging.info(...)` |
| `ContextCancellation` | P2 | Cancels a context or cancellation token | `cancel()` | `event.set()` |
| `AsyncGeneratorYield` | P2 | Yields from an async generator | *(no direct equivalent)* | `async def gen(): yield x` |
| `MetaprogrammingMutation` | P2 | Mutates program structure at runtime | *(no direct equivalent)* | `setattr(obj, name, val)` |
| `DescriptorEffect` | P2 | Side effect via descriptor protocol | *(no direct equivalent)* | `__set__`, `__delete__` |
| `ResourceManagement` | P2 | Acquires or releases an external resource | `file.Close()` | `__enter__`/`__exit__` |
| `ImportSideEffect` | P2 | Module import triggers observable side effects | `import _ "pkg"` | `import module` (with side effects) |
| `MonkeyPatch` | P2 | Runtime replacement of functions/methods | *(no direct equivalent)* | `module.func = mock` |

##### P3 — Nice to Have (9 types)

| Type | Tier | Definition | Go Mapping | Python Mapping |
|------|------|-----------|------------|----------------|
| `StdoutWrite` | P3 | Writes to standard output | `fmt.Println(...)` | `print(...)` |
| `StderrWrite` | P3 | Writes to standard error | `fmt.Fprintln(os.Stderr, ...)` | `print(..., file=sys.stderr)` |
| `EnvVarMutation` | P3 | Modifies environment variables | `os.Setenv(k, v)` | `os.environ[k] = v` |
| `MutexOp` | P3 | Acquires or releases a mutex/lock | `mu.Lock()` | `lock.acquire()` |
| `WaitGroupOp` | P3 | Operates on a synchronization barrier | `wg.Add(1)` | `barrier.wait()` |
| `AtomicOp` | P3 | Performs an atomic memory operation | `atomic.AddInt64(...)` | *(no direct equivalent)* |
| `TimeDependency` | P3 | Depends on wall-clock time | `time.Now()` | `time.time()` |
| `ProcessExit` | P3 | Terminates the process | `os.Exit(1)` | `sys.exit(1)` |
| `RecoverBehavior` | P3 | Recovers from a panic/exception | `recover()` | `except Exception:` |

##### P4 — Exotic (6 types)

| Type | Tier | Definition | Go Mapping | Python Mapping |
|------|------|-----------|------------|----------------|
| `ReflectionMutation` | P4 | Mutates state via reflection | `reflect.ValueOf(&x).Elem().Set(...)` | `setattr(obj, name, val)` |
| `UnsafeMutation` | P4 | Mutates state via unsafe pointer operations | `unsafe.Pointer` | `ctypes` pointer writes |
| `CgoCall` | P4 | Calls foreign function interface (FFI) | `C.function()` | `ctypes.cdll.func()` |
| `FinalizerRegistration` | P4 | Registers a finalizer/destructor callback | `runtime.SetFinalizer(...)` | `weakref.finalize(...)` |
| `SyncPoolOp` | P4 | Operates on an object pool | `sync.Pool.Get/Put` | *(no direct equivalent)* |
| `ClosureCaptureMutation` | P4 | Mutates a variable captured by a closure | captured var write | captured var write |

##### Language-Neutral Aliases

The following aliases allow analyzers to use language-neutral names that resolve to the same canonical type:

| Alias | Resolves To | Use When |
|-------|------------|----------|
| `AsyncTaskSpawn` | `GoroutineSpawn` | Spawning async tasks, threads, coroutines |
| `AsyncMessageSend` | `ChannelSend` | Sending to queues, mailboxes, actors |
| `AsyncChannelClose` | `ChannelClose` | Closing queues, mailboxes, streams |
| `BarrierOp` | `WaitGroupOp` | Barriers, latches, countdown events |
| `PanicRecovery` | `RecoverBehavior` | Exception handling, panic recovery |
| `FFICall` | `CgoCall` | Foreign function interface calls (ctypes, napi, JNI) |
| `ObjectPoolOp` | `SyncPoolOp` | Object/connection pool operations |
| `DeferredMutation` | `DeferredReturnMutation` | Finally blocks, scope guards, RAII cleanup |
| `ArgumentMutation` | `PointerArgMutation` | Mutating parameters passed by reference |
| `ProcessTermination` | `Panic` | Process abort, unrecoverable errors |
| `SentinelErrorDecl` | `SentinelError` | Named error type/constant declarations |

Aliases are accepted in `analyze` responses and resolve to the canonical type for scoring and classification. The full taxonomy is defined in `internal/taxonomy/types.go`.

##### The `detail` Field

Each side effect in the `analyze` response may include an optional `detail` object containing language-specific metadata. This field is **opaque to Gaze** — it is passed through to JSON output and AI report pipelines without interpretation. Gaze does not use `detail` for scoring, classification, or CRAP computation.

Use `detail` to provide context that helps AI report generators or downstream tooling understand the effect in language-specific terms (e.g., exception class names, decorator metadata, type annotations).

> **Size guidance**: Keep `detail` payloads small — under 1KB per effect. Large payloads increase JSON output size and may degrade AI report quality. Include only the metadata needed for downstream interpretation.

#### Classification

Each side effect may include a `classification` object:

| Field | Type | Description |
|-------|------|-------------|
| `label` | string | `"contractual"`, `"incidental"`, or `"ambiguous"` |
| `confidence` | integer | 0-100 confidence score |

When `classification` is null, Gaze uses default classification based on the effect type's tier.

**Language-neutral side effect type aliases**: External analyzers may use either Go-specific type names (e.g., `GoroutineSpawn`, `ChannelSend`, `CgoCall`) or language-neutral aliases (e.g., `AsyncTaskSpawn`, `AsyncMessageSend`, `FFICall`). The aliases map to the same string values as the Go-specific names: `AsyncTaskSpawn` = `GoroutineSpawn`, `AsyncMessageSend` = `ChannelSend`, `AsyncChannelClose` = `ChannelClose`, `BarrierOp` = `WaitGroupOp`, `PanicRecovery` = `RecoverBehavior`, `FFICall` = `CgoCall`, `ObjectPoolOp` = `SyncPoolOp`.

---

### `analyze/stream` (optional)

Streaming alternative to the batch `analyze` method. When the analyzer declares `capabilities.streaming: true` in the `initialize` response, Gaze calls `analyze/stream` instead of `analyze`.

**Timeout**: 5 minutes

**Request params**: Same as `analyze`.

**Response format**: Instead of a single JSON-RPC response, the analyzer writes one JSON object per line (JSONL) to stdout. Each line represents one `AnalyzedFunction` — the same schema as the `functions` array elements in the batch response.

```jsonl
{"name":"add","package":"math_utils","file":"math_utils/ops.py","line":1,"side_effects":[]}
{"name":"multiply","package":"math_utils","file":"math_utils/ops.py","line":10,"side_effects":[{"type":"ReturnValue","description":"returns result","location":"ops.py:12:5","target":"result"}]}
{"name":"divide","package":"math_utils","file":"math_utils/ops.py","line":20,"side_effects":[{"type":"ReturnValue","description":"returns result","location":"ops.py:25:5","target":"result"},{"type":"ErrorReturn","description":"raises ZeroDivisionError","location":"ops.py:22:9","target":"ZeroDivisionError"}]}
```

**Error handling**: If a line contains invalid JSON, Gaze stops processing (fail-fast) and reports the error with the line number and content.

**When to use streaming**: Streaming is beneficial for large codebases (10K+ functions) where a single batch response would be multi-megabyte. For small codebases, the batch `analyze` method is simpler and sufficient.

---

### `complexity`

Compute cyclomatic complexity for all functions.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "name": "divide",
      "package": "math_utils",
      "file": "math_utils/ops.py",
      "line": 20,
      "complexity": 5
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Function or method name |
| `package` | string | Package/module path |
| `file` | string | Source file path (relative or absolute) |
| `line` | integer | Line number of function declaration |
| `complexity` | integer | Cyclomatic complexity value |

---

### `coverage`

Produce per-function line coverage data. The analyzer is responsible for running tests and collecting coverage internally.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "file": "math_utils/ops.py",
      "function": "divide",
      "start_line": 20,
      "end_line": 30,
      "covered_stmts": 0,
      "total_stmts": 10,
      "percentage": 0.0
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Source file path |
| `function` | string | Function or method name |
| `start_line` | integer | Function declaration start line |
| `end_line` | integer | Function body end line |
| `covered_stmts` | integer | Number of statements covered by tests |
| `total_stmts` | integer | Total number of statements |
| `percentage` | number | Coverage percentage (0.0-100.0) |

---

### `shutdown`

Request the analyzer to shut down cleanly. The analyzer should finish any pending work and exit.

**Timeout**: 10 seconds

**Request params**: none (or `null`)

**Response result**: `{}`

After receiving the shutdown response, Gaze closes stdin and waits for the process to exit.

---

### Optional Methods

These methods are declared via capabilities in the `initialize` response. Gaze only calls them when the corresponding capability is `true`.

---

### `discover` (optional)

Find source and test files in the project. Reserved for future use -- currently not consumed by Gaze's provider interfaces.

**Capability**: `discover`

**Request params**:

```json
{
  "root_path": "/path/to/project"
}
```

**Response result**:

```json
{
  "source_files": ["math_utils/ops.py", "math_utils/helpers.py"],
  "test_files": ["tests/test_ops.py"],
  "framework": "pytest"
}
```

---

### `test_mapping` (optional)

Map test assertions to side effects. When supported, this enables GazeCRAP scoring (contract coverage).

**Capability**: `test_mapping`

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "mappings": [
    {
      "test_function": "test_multiply",
      "test_file": "tests/test_ops.py",
      "assertion_location": "tests/test_ops.py:10",
      "assertion_type": "equality",
      "target_function": "multiply",
      "target_package": "math_utils",
      "side_effect_type": "ReturnValue",
      "confidence": 80
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `test_function` | string | Test function name |
| `test_file` | string | Test file path |
| `assertion_location` | string | Source position of the assertion |
| `assertion_type` | string | Kind of assertion (e.g., "equality", "error_check") |
| `target_function` | string | Function under test |
| `target_package` | string | Package of the function under test |
| `side_effect_type` | string | Type of side effect being asserted on |
| `confidence` | integer | Mapping confidence (0-100) |

---

### `classify_signals` (optional)

Provide raw classification signals for Gaze's scoring engine. When supported, these signals are fed into `classify.ComputeScore` alongside Gaze's built-in signals.

**Capability**: `classify_signals`

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "signals": [
    {
      "function": "divide",
      "package": "math_utils",
      "side_effect_type": "ErrorReturn",
      "source": "docstring",
      "weight": 15,
      "reasoning": "docstring mentions ZeroDivisionError"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `function` | string | Function name |
| `package` | string | Package path |
| `side_effect_type` | string | Side effect type this signal relates to |
| `source` | string | Signal type (e.g., "docstring", "type_annotation", "decorator") |
| `weight` | integer | Numeric contribution to confidence score |
| `reasoning` | string | (optional) Explanation of why this signal was applied |

### `doc_coverage` (optional)

Report documentation coverage for public symbols. When supported, Gaze uses the analyzer's native documentation analysis instead of heuristic backtick scanning.

**Capability**: `doc_coverage`

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root_path` | string | yes | Absolute path to the project root |
| `patterns` | string[] | yes | Package patterns to analyze |

**Response result**:

```json
{
  "symbols": [
    {
      "name": "divide",
      "package": "math_utils",
      "kind": "function",
      "file": "math_utils/ops.py",
      "line": 20,
      "documented": true,
      "doc_snippet": "Divide two numbers, raising ZeroDivisionError for zero divisor."
    },
    {
      "name": "add",
      "package": "math_utils",
      "kind": "function",
      "file": "math_utils/ops.py",
      "line": 1,
      "documented": false
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `symbols[].name` | string | Symbol name (function, type, constant, etc.) |
| `symbols[].package` | string | Package/module path |
| `symbols[].kind` | string | Symbol kind: `"function"`, `"type"`, `"constant"`, `"variable"`, `"class"`, `"method"` |
| `symbols[].file` | string | Source file path |
| `symbols[].line` | integer | Line number of symbol declaration |
| `symbols[].documented` | boolean | Whether the symbol has documentation |
| `symbols[].doc_snippet` | string | (optional) First line or summary of the documentation. Omitted when empty. |

---

## Error Handling

### Required method errors

When a required method (`analyze`, `complexity`, `coverage`) returns a JSON-RPC error, Gaze propagates the error and exits non-zero. The error message is displayed to the user.

### Optional method errors

When an optional method (`discover`, `test_mapping`, `classify_signals`, `doc_coverage`) returns an error, Gaze logs a warning to stderr and degrades gracefully:

- `discover` error: no impact (not currently consumed)
- `test_mapping` error: GazeCRAP is unavailable
- `classify_signals` error: uses pre-classified effects from `analyze`
- `doc_coverage` error: falls back to heuristic documentation coverage from `analyze` output

### Process crashes

If the analyzer process exits unexpectedly (crash, segfault), Gaze detects the closed stdin/stdout pipes and returns an error with the process's stderr output for diagnostics.

### Timeouts

Each method has a default timeout (30s for handshake methods, 5min for analysis methods). When the timeout expires, Gaze kills the subprocess and returns a timeout error.

## Configuration

### `.gaze.yaml`

```yaml
analyzers:
  python:
    command: snake-eyes
    args: ["--stdio"]
  rust:
    command: /usr/local/bin/gaze-analyzer-rust
    args: ["--stdio", "--edition=2021"]
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `--analyzer <binary>` | Explicit analyzer binary name or path |
| `--language <lang>` | Target language for config/PATH discovery |

## Building an Analyzer

To build a Gaze-compatible analyzer:

1. **Accept `--stdio` flag**: Read JSON-RPC requests from stdin, write responses to stdout, diagnostics to stderr.
2. **Implement the 5 required methods**: `initialize`, `analyze`, `complexity`, `coverage`, `shutdown`.
3. **Declare capabilities**: In the `initialize` response, set `test_mapping: true` if you can map assertions to effects (enables GazeCRAP), and `doc_coverage: true` if you can report documentation status per symbol.
4. **Map to Gaze's taxonomy**: Use Gaze's `SideEffectType` constants for the `type` field in `analyze` responses.
5. **Follow naming convention**: Name your binary `gaze-analyzer-<language>` for automatic PATH discovery.

### Minimal Example (Python)

```python
#!/usr/bin/env python3
"""Minimal Gaze analyzer for demonstration."""
import json
import sys

def handle(request):
    method = request["method"]
    rid = request["id"]

    if method == "initialize":
        return {"jsonrpc": "2.0", "id": rid, "result": {
            "capabilities": {"discover": False, "test_mapping": False, "classify_signals": False, "streaming": False, "doc_coverage": False},
            "protocol_version": "1.1.0",
            "analyzer_name": "minimal-python",
            "language": "python"
        }}
    elif method == "analyze":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "complexity":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "coverage":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "shutdown":
        return {"jsonrpc": "2.0", "id": rid, "result": {}}
    else:
        return {"jsonrpc": "2.0", "id": rid, "error": {
            "code": -32601, "message": f"Method not found: {method}"
        }}

for line in sys.stdin:
    request = json.loads(line.strip())
    response = handle(request)
    print(json.dumps(response), flush=True)
    if request["method"] == "shutdown":
        break
```

### Testing Your Analyzer

Use Gaze's fake analyzer as a reference implementation: `internal/protocol/testdata/fake_analyzer/main.go`.

```bash
# Test with gaze crap
gaze crap --analyzer ./my-analyzer --language python ./src

# Test with JSON output (no AI adapter needed)
gaze report --analyzer ./my-analyzer --format=json ./src

# Test documentation coverage
gaze docscan --analyzer ./my-analyzer --language python ./src
```
