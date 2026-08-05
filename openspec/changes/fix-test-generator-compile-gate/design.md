## Context

The `gaze-test-generator` agent generates Go test code, GoDoc
improvements, and assertion restructurings based on gaze quality
analysis data. It writes generated code to disk using the Write
and Edit tools. The agent's "Important Constraints" section
(line 275) says "ALWAYS verify generated code compiles before
reporting success" but provides no concrete protocol for how or
when to verify.

The parent audit (unbound-force/unbound-force#346) identified
that advisory prose gates are systematically bypassed under
context compression. The fix pattern established in that audit
is to replace advisory text with concrete tool call sequences
and explicit halt conditions.

A fix was already applied in
[unbound-force/unbound-force#404](https://github.com/unbound-force/unbound-force/pull/404)
but targeted the wrong repo. The gaze repo has two copies that
must stay in sync:

- `internal/scaffold/assets/agents/gaze-test-generator.md`
  (canonical scaffold, shipped via `gaze init` — 277 lines)
- `.opencode/agents/gaze-test-generator.md` (active runtime
  copy — currently identical to scaffold, 277 lines)

Both copies contain the old advisory prose and require the same
fix.

## Goals / Non-Goals

### Goals
- Replace advisory compile verification prose with a concrete
  pre-write gate protocol
- Specify the exact bash command, failure handling, and halt
  condition
- Maintain consistency between scaffold and active copies
- Follow the T3 remediation pattern from issue #346

### Non-Goals
- Modifying the `/gaze fix` command — it already has its own
  verification step in Step 4
- Adding compile gates to other agents — that is separate work
  tracked in the parent audit
- Changing the agent's tool permissions or mode
- Introducing the pre-flight skill dependency — the compile
  check is simple enough to inline as `go build`

## Decisions

### D1: Inline compile gate, not pre-flight skill delegation

The pre-flight skill is designed for CI-aware, multi-tool
execution with coverage matrices and baseline classification.
The gaze-test-generator needs a single `go build` check before
each write. Delegating to pre-flight would add unnecessary
complexity and make the agent harder to reason about.

Decision: Inline the compile check as a concrete 3-step protocol
in the agent's "Important Constraints" section.

### D2: Gate placement — before Write, not after all generation

The issue specifies "before any Write tool call." This is the
correct placement because:
- Writing non-compiling code to disk is the harmful action
- A post-generation check allows broken files to persist
- Per-write gating catches each file individually

Decision: The gate fires before each Write/Edit tool call, not
as a batch check after all generation.

### D3: Compile scope — package-level, not full repo

Running `go build ./...` (full repo) after each generated test
file is expensive and may surface pre-existing errors unrelated
to the generated code. Running `go build ./path/to/package/...`
scopes the check to the package being modified.

Decision: Use `go build ./path/to/package/...` scoped to the
target package. This matches the existing pattern in the Output
Format section (line 257).

### D4: Edit both copies consistently

The scaffold copy (`internal/scaffold/assets/agents/`) is the
canonical source. The active copy (`.opencode/agents/`) is
currently identical. Both must receive the same changes.

Decision: Apply the compile gate changes to the scaffold copy
first. Then sync the active copy to match. Verify with `diff`.

## Risks / Trade-offs

### Risk: False negatives from pre-existing build errors

If the target project already has compilation errors, the gate
will block all writes even though the generated code is correct.

Mitigation: The gate instruction specifies to report the error
context, allowing the user to identify pre-existing vs generated
errors. This is acceptable — writing into a non-compiling
codebase is itself a risky action.

### Risk: Agent may still skip the gate under extreme compression

No amount of prose can guarantee LLM compliance. However, the
concrete 3-step protocol with an explicit "MUST NOT" halt
condition is significantly more resistant to fast-path skipping
than advisory prose.

### Trade-off: Per-write compilation adds latency

Running `go build` before each write adds variable latency:
~1-3 seconds per file with a warm build cache, potentially
10-30+ seconds on a cold cache or for large dependency trees.
For batch operations processing 10+ functions, this adds
meaningful overhead. The Go build cache mitigates repeated
builds of the same package. This is acceptable because
correctness outweighs speed for code generation.

### Note: Overlap with /gaze fix verification

The `/gaze fix` command (`.opencode/commands/gaze-fix.md`,
Step 4) already has its own post-generation compile and test
verification. When `/gaze fix` invokes the gaze-test-generator
agent, generated code will be verified twice: once by the
agent's pre-write gate and once by `/gaze fix`'s Step 4. This
is intentional defense-in-depth — the pre-write gate catches
errors before they reach disk; the batch check provides a
final integrity verification across all modified packages.
