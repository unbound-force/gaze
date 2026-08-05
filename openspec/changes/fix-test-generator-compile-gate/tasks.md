<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file --
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Update canonical scaffold agent file

- [x] 1.1 Replace advisory compile prose in Important Constraints
  - File: `internal/scaffold/assets/agents/gaze-test-generator.md`
  - Replace the bullet reading "ALWAYS verify generated code
    compiles before reporting success" (line 275) with the
    concrete pre-write compile gate protocol:
    ```
    - Before any Write or Edit tool call that modifies a Go
      source or test file, MUST run compile verification:
      1. Run via bash: `go build ./path/to/package/...`
         (scoped to the target package being modified)
      2. If the command exits with non-zero code, MUST NOT
         proceed with the Write or Edit call. Report the
         compilation error and continue to the next target.
      3. Only proceed with the Write or Edit call after a
         successful (exit code 0) compile check.
    ```

- [x] 1.2 Update Output Format section to reference pre-write gate
  - File: `internal/scaffold/assets/agents/gaze-test-generator.md`
  - In the Output Format section (around line 254), change
    "After generating all code, run:" to clarify that individual
    files have already been verified by the pre-write compile
    gate, and this final check is a full-package integrity
    verification.

## 2. Sync active copy from scaffold

- [x] 2.1 Copy updated scaffold to active runtime copy
  - Source: `internal/scaffold/assets/agents/gaze-test-generator.md`
  - Target: `.opencode/agents/gaze-test-generator.md`
  - Copy the full updated scaffold content to the active copy.
    Both files should be byte-identical after this step.

## 3. Verification

- [x] 3.1 Verify copy consistency with explicit diff
  - Run:
    ```bash
    diff internal/scaffold/assets/agents/gaze-test-generator.md \
         .opencode/agents/gaze-test-generator.md
    ```
  - Must produce no output (files are identical).

- [x] 3.2 Verify old advisory text is removed
  - Run:
    ```bash
    grep "ALWAYS verify generated code compiles" \
      internal/scaffold/assets/agents/gaze-test-generator.md
    ```
  - Must return no results.

- [x] 3.3 Verify new protocol is present
  - Run:
    ```bash
    grep "MUST NOT" \
      internal/scaffold/assets/agents/gaze-test-generator.md
    ```
  - Must return at least one result containing the halt
    condition.

- [x] 3.4 Run full build and test suite
  - Command: `go build ./... && go test -race -count=1 -short ./...`
  - Verify no regressions from the agent file changes.
  - Note: agent Markdown files are embedded via `embed.FS` in
    `internal/scaffold/`, so build verification confirms the
    embedded assets are valid.

- [x] 3.5 Verify constitution alignment
  - Confirm the change aligns with Principle III (Observable
    Quality) by adding a concrete verification step that
    produces observable output.
  - Confirm the change aligns with Principle IV (Testability)
    by enforcing verification of observable side effects before
    writing.
  - No new dependencies introduced (Principle II).

## 4. Documentation Gate

- [x] 4.1 Assess documentation impact
  - This change modifies internal agent behavior (the
    gaze-test-generator's pre-write verification protocol).
    It does not change user-facing CLI commands, workflows, or
    hero capabilities. Exempt from website documentation issue
    requirement per AGENTS.md: "Internal refactoring with no
    user-facing behavior change."

<!-- spec-review: passed -->
<!-- code-review: passed -->
