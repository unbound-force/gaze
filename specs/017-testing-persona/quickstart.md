# Quickstart: Testing Persona Integration

**Feature**: 017-testing-persona
**Date**: 2026-03-05

## Verification Guide

After implementation, use these steps to verify each deliverable works correctly.

### 1. Constitution Amendment

Verify that Principle IV exists and the Constitution Check recognizes it:

```bash
# Check the constitution contains Principle IV
grep -c "### IV. Testability" .specify/memory/constitution.md
# Expected: 1

# Check version was bumped
grep "Version.*1.1.0" .specify/memory/constitution.md
# Expected: match found
```

The Constitution Check in `/speckit.plan` automatically reads all principles from the constitution file. No command modification is needed — creating a new spec and running `/speckit.plan` on it will validate Principle IV alongside the existing three.

### 2. Reviewer-Testing Agent

Verify the agent file is correctly structured:

```bash
# Check frontmatter contains required fields
head -10 .opencode/agents/reviewer-testing.md
# Expected: mode: subagent, model: claude-sonnet-4-6, temperature: 0.1, read-only tools

# Check dual-mode sections exist
grep -c "## Code Review Mode" .opencode/agents/reviewer-testing.md
# Expected: 1
grep -c "## Spec Review Mode" .opencode/agents/reviewer-testing.md
# Expected: 1

# Check CRITICAL severity for missing coverage strategy
grep "CRITICAL" .opencode/agents/reviewer-testing.md | head -3
# Expected: reference to missing coverage strategy being CRITICAL
```

### 3. `/speckit.testreview` Command

In OpenCode, run:

```
/speckit.testreview
```

On a feature branch with completed spec.md, plan.md, and tasks.md. Verify:
- The command runs `check-prerequisites.sh` and resolves artifact paths
- The command delegates to `reviewer-testing` in Spec Review Mode
- A structured report is produced with severity-ranked findings
- The report includes next actions and remediation suggestions

### 4. Review Council (4 Reviewers)

In OpenCode, run:

```
/review-council
```

Verify:
- Four agents are dispatched in parallel (not three)
- The summary shows verdicts from all four: The Adversary, The Architect, The Guard, The Tester
- The overall verdict requires all four to APPROVE

For spec review mode:

```
/review-council specs
```

Verify The Tester audits spec artifacts for testability alongside the other three.

### 5. Scaffold Deployment

```bash
# Run tests to verify scaffold changes
go test -race -count=1 ./internal/scaffold/...

# Verify all 7 assets are embedded
go test -race -count=1 -run TestAssetPaths ./internal/scaffold/...

# Verify no drift between .opencode/ and scaffold assets
go test -race -count=1 -run TestEmbeddedAssetsMatchSource ./internal/scaffold/...

# Manual verification: scaffold into a temp directory
cd $(mktemp -d) && mkdir -p . && echo "module test" > go.mod
gaze init
ls -la .opencode/agents/    # Should show gaze-reporter.md, reviewer-testing.md
ls -la .opencode/command/   # Should show gaze.md, speckit.testreview.md, review-council.md
ls -la .opencode/references/ # Should show doc-scoring-model.md, example-report.md
```

### 6. Ownership Behavior

```bash
# First init: all 7 created
gaze init
# Expected: 7 "created:" lines

# Second init (same version): all 7 skipped
gaze init
# Expected: 7 "skipped:" lines, "2 files skipped (use --force to overwrite)" hint
# (only 2 user-owned agent files + 1 user-owned command = 3 user-owned, but
# tool-owned with identical content are also skipped — hint counts only user-owned)

# Modify a tool-owned command
echo "modified" >> .opencode/command/speckit.testreview.md
gaze init
# Expected: speckit.testreview.md "updated: (content changed)"

# Modify a user-owned agent
echo "modified" >> .opencode/agents/reviewer-testing.md
gaze init
# Expected: reviewer-testing.md "skipped: (already exists)"

# Force overwrite everything
gaze init --force
# Expected: 7 "overwritten:" lines
```

## Test Commands

```bash
# Run all scaffold tests
go test -race -count=1 ./internal/scaffold/...

# Run full unit + integration test suite
go test -race -count=1 -short ./...

# Lint
golangci-lint run
```
