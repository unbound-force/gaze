## ADDED Requirements

### Requirement: FR-001 Pre-Write Compile Gate

The gaze-test-generator agent MUST run a compile verification
check before any Write or Edit tool call that modifies a Go
source file or test file. The check MUST use the bash tool to
execute `go build ./path/to/package/...` scoped to the target
package. If the compile check exits with a non-zero exit code,
the agent MUST NOT proceed with the Write or Edit tool call.
The agent MUST report the compilation error and continue
processing remaining target functions.

#### Scenario: Successful compile before write

- **GIVEN** the agent has generated a test function for
  package `internal/foo`
- **WHEN** the agent runs `go build ./internal/foo/...` and
  the command exits with code 0
- **THEN** the agent proceeds with the Write tool call to
  create or append to the test file

#### Scenario: Failed compile before write

- **GIVEN** the agent has generated a test function for
  package `internal/foo`
- **WHEN** the agent runs `go build ./internal/foo/...` and
  the command exits with a non-zero code
- **THEN** the agent MUST NOT execute the Write tool call
- **AND** the agent reports the compilation error output
- **AND** the agent continues processing remaining target
  functions (does not abort the entire batch)

#### Scenario: Compile gate for doc improvements (Edit tool)

- **GIVEN** the agent has generated a GoDoc improvement for
  a function in package `internal/bar`
- **WHEN** the agent runs `go build ./internal/bar/...` after
  composing the doc comment change and the command exits with
  code 0
- **THEN** the agent proceeds with the Edit tool call to
  modify the source file

#### Scenario: Failed compile before Edit tool call

- **GIVEN** the agent has composed a GoDoc improvement for
  a function in package `internal/bar`
- **WHEN** the agent runs `go build ./internal/bar/...` and
  the command exits with a non-zero code
- **THEN** the agent MUST NOT execute the Edit tool call
- **AND** the original file content is preserved unchanged
- **AND** the agent reports the compilation error output
- **AND** the agent continues processing remaining target
  functions

### Requirement: FR-002 Compile Gate Protocol Specification

The compile gate MUST be specified as a numbered protocol in
the agent's "Important Constraints" section, not as advisory
prose. The protocol MUST include:
1. The exact bash command to run
2. The exit code check
3. The explicit halt condition (MUST NOT write)

This ensures the gate is not skippable under context
compression, following the T3 remediation pattern from
issue #346.

#### Scenario: Protocol is structurally verifiable

- **GIVEN** the agent file at
  `internal/scaffold/assets/agents/gaze-test-generator.md`
- **WHEN** the Important Constraints section is inspected
- **THEN** it contains a numbered list with at least 3 steps
- **AND** the list uses RFC 2119 "MUST NOT" language for the
  halt condition
- **AND** the list includes a concrete `go build` command

## MODIFIED Requirements

### Requirement: FR-003 Output Format Verification Order

Previously: "After generating all code, run: `go build`..."

The Output Format section's post-generation compile check
SHOULD remain as a final batch verification, but the pre-write
gate (FR-001) MUST take precedence. The Output Format section
MUST note that individual files have already passed compilation
via the pre-write gate, and the final batch check serves as a
full-package integrity verification.

#### Scenario: Batch verification after individual gates

- **GIVEN** the agent has written 3 test files, each passing
  the pre-write compile gate individually
- **WHEN** the agent reaches the Output Format verification
  step
- **THEN** the agent runs `go build ./...` as a final
  integrity check across all modified packages
- **AND** reports the aggregate compilation status

### Requirement: FR-004 Important Constraints Section Update

Previously: "ALWAYS verify generated code compiles before
reporting success"

The bullet point reading "ALWAYS verify generated code compiles
before reporting success" (line 275 in the scaffold copy) MUST
be replaced with the concrete compile gate protocol (FR-002).
The replacement text MUST use MUST NOT language for the halt
condition, not ALWAYS advisory language.

**Coverage strategy**: This change modifies agent Markdown
files only — no new Go source code is introduced. Verification
is structural: the agent file contains the numbered protocol
with MUST NOT language, and both copies are byte-identical.
Drift detection uses explicit `diff` commands.

#### Scenario: Constraint replacement

- **GIVEN** the current text reads "ALWAYS verify generated
  code compiles before reporting success"
- **WHEN** the change is applied
- **THEN** the text is replaced with a numbered pre-write
  protocol specifying the bash command, exit code check, and
  MUST NOT halt condition

## REMOVED Requirements

None — no existing requirements are removed. The advisory
prose is replaced with a stronger, concrete version.
