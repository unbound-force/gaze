# Quickstart: Consolidate /classify-docs into /gaze

**Date**: 2026-03-01
**Branch**: `012-consolidate-classify-docs`

## What This Changes

- The `/classify-docs` command and `doc-classifier` agent are removed
- The `/gaze` command now handles everything: CRAP, quality, classification (with document-enhanced scoring), and health assessment
- The gaze-reporter agent prompt is restructured for reliable emoji/formatting output
- `gaze init` distributes 2 files instead of 4

## Implementation Order

### Step 1: Delete the command and agent files (4 deletions)

```bash
# Live copies
rm .opencode/command/classify-docs.md
rm .opencode/agents/doc-classifier.md

# Embedded scaffold copies
rm internal/scaffold/assets/command/classify-docs.md
rm internal/scaffold/assets/agents/doc-classifier.md
```

### Step 2: Rewrite the gaze-reporter agent prompt

Edit `.opencode/agents/gaze-reporter.md`:

1. Add formatting override block after opening paragraph (new section 3)
2. Add compact example after override block (new section 4 — sandwich top)
3. Rewrite Full Mode section:
   - Remove lines 134-136 (`/classify-docs` delegation)
   - Add `### Document-Enhanced Classification` subsection with inlined scoring model
4. Update canonical example (remove `/classify-docs` reference at line 341)
5. Keep full example at end (sandwich bottom)

### Step 3: Sync the embedded scaffold copy

```bash
cp .opencode/agents/gaze-reporter.md internal/scaffold/assets/agents/gaze-reporter.md
```

### Step 4: Update scaffold tests

Edit `internal/scaffold/scaffold_test.go`:
- Change all `4` → `2` for file counts in 6 test functions
- Remove `doc-classifier.md` and `classify-docs.md` from expected path lists

### Step 5: Verify

```bash
# Run scaffold tests
go test -race -count=1 ./internal/scaffold/...

# Verify no stale references
grep -r "classify-docs\|doc-classifier" .opencode/ internal/scaffold/assets/
# Should return no results

# Verify scaffold creates 2 files
# (covered by TestRun_CreatesFiles)
```

## Testing Checklist

- [ ] `go test -race -count=1 ./internal/scaffold/...` passes
- [ ] `grep -r "classify-docs" .opencode/ internal/scaffold/assets/` returns nothing
- [ ] `grep -r "doc-classifier" .opencode/ internal/scaffold/assets/` returns nothing
- [ ] Gaze-reporter prompt contains formatting override block
- [ ] Gaze-reporter prompt contains compact example early (sandwich top)
- [ ] Gaze-reporter prompt contains full example at end (sandwich bottom)
- [ ] Gaze-reporter prompt contains Document-Enhanced Classification section
- [ ] Gaze-reporter prompt contains scoring model tables (signal sources + AI inference)
- [ ] Manual: run `/gaze` in OpenCode and verify emojis appear in output
