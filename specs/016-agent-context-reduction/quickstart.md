# Quickstart: Agent Context Reduction

**Feature**: 016-agent-context-reduction
**Date**: 2026-03-02

## Prerequisites

- Go 1.24+ installed
- Gaze repository cloned and on the `016-agent-context-reduction` branch
- OpenCode installed (for manual verification of `/gaze` command)

## Implementation Steps

### Step 1: Create reference files

Create two new files under `.opencode/references/` and their scaffold copies under `internal/scaffold/assets/references/`:

1. `.opencode/references/example-report.md` — extract the `## Example Output` section from the current agent prompt
2. `.opencode/references/doc-scoring-model.md` — extract the `### Document-Enhanced Classification` subsection from `## Full Mode`

### Step 2: Update the agent prompt

Edit `.opencode/agents/gaze-reporter.md`:

1. Remove the `## Example Output` section (lines 382-453)
2. Replace with a `## Reference Files` section containing read instructions
3. Remove the `### Document-Enhanced Classification` subsection (lines 201-250)
4. Replace with an instruction to read `doc-scoring-model.md` in full mode
5. Replace the quadrant label listing in CRAP Mode (lines 123-127) with a cross-reference

### Step 3: Update the scaffold copy

Copy the updated `.opencode/agents/gaze-reporter.md` to `internal/scaffold/assets/agents/gaze-reporter.md` to keep them byte-identical.

### Step 4: Update scaffold logic

Edit `internal/scaffold/scaffold.go` to add overwrite-on-diff behavior for `references/` files. Add `Updated` field to the `Result` struct.

### Step 5: Update tests

Update hardcoded file counts and expected file lists in:
- `internal/scaffold/scaffold_test.go`
- `cmd/gaze/main_test.go`

Add tests for the new overwrite-on-diff behavior.

## Verification

### Prompt Size Check

```bash
wc -c .opencode/agents/gaze-reporter.md
# Expected: ≤13,300 bytes (down from 17,775)
```

### Scaffold Sync Check

```bash
diff .opencode/agents/gaze-reporter.md internal/scaffold/assets/agents/gaze-reporter.md
diff .opencode/references/example-report.md internal/scaffold/assets/references/example-report.md
diff .opencode/references/doc-scoring-model.md internal/scaffold/assets/references/doc-scoring-model.md
# All should show no differences
```

### Build and Test

```bash
go build ./cmd/gaze
go test -race -count=1 -short ./...
```

### GoReleaser Validation

```bash
goreleaser check
```

### Manual Formatting Verification

Run `/gaze crap ./...` in OpenCode and verify:
- Report starts with `🔍 Gaze CRAP Report`
- CRAP Summary uses `📊` marker
- Quadrant distribution uses correct emoji-prefixed labels
- Tables are properly formatted with right-aligned numeric columns
- Metadata line shows project, branch, version, Go version, date

Run `/gaze` (full mode) and verify:
- All 5 sections present with correct emoji markers
- Document-Enhanced Classification applies correct signal weights
- Recommendations are severity-prefixed with correct emojis

### Scaffold Overwrite Verification

```bash
# Fresh scaffold
mkdir /tmp/test-scaffold && cd /tmp/test-scaffold
go mod init test && gaze init

# Verify reference files created
ls .opencode/references/
# Expected: example-report.md  doc-scoring-model.md

# Modify a reference file
echo "modified" > .opencode/references/example-report.md

# Re-run scaffold (without --force)
gaze init
# Expected: example-report.md listed as "updated" (overwritten)
# Expected: agent and command files listed as "skipped"

# Verify agent file was NOT overwritten
# (its content should still match the original scaffold)
```
