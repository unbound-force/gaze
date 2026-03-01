# Quickstart: Init Update Strategy

**Branch**: `012-init-update-strategy` | **Date**: 2026-03-01

## What This Feature Does

After this change, `gaze init` automatically detects and updates scaffolded files that are out of date. Previously, running `gaze init` after upgrading gaze would silently skip all existing files. Now it compares the version marker in each file against the running binary's version and replaces outdated files automatically.

## Before (current behavior)

```
$ gaze init                         # First install — creates files
Gaze OpenCode integration initialized:
  created: .opencode/agents/gaze-reporter.md
  created: .opencode/agents/doc-classifier.md
  created: .opencode/command/gaze.md
  created: .opencode/command/classify-docs.md

$ # ... upgrade gaze from v1.0.0 to v2.0.0 ...

$ gaze init                         # Files are silently skipped
Gaze OpenCode integration already up to date:
  skipped: .opencode/agents/gaze-reporter.md (already exists)
  skipped: .opencode/agents/doc-classifier.md (already exists)
  skipped: .opencode/command/gaze.md (already exists)
  skipped: .opencode/command/classify-docs.md (already exists)

4 files skipped (use --force to overwrite).
```

## After (new behavior)

```
$ gaze init                         # First install — same as before
Gaze OpenCode integration initialized:
  created:    .opencode/agents/gaze-reporter.md
  created:    .opencode/agents/doc-classifier.md
  created:    .opencode/command/gaze.md
  created:    .opencode/command/classify-docs.md

$ # ... upgrade gaze from v1.0.0 to v2.0.0 ...

$ gaze init                         # Outdated files are updated automatically
Gaze OpenCode integration initialized:
  updated:    .opencode/agents/gaze-reporter.md (v1.0.0 -> v2.0.0)
  updated:    .opencode/agents/doc-classifier.md (v1.0.0 -> v2.0.0)
  updated:    .opencode/command/gaze.md (v1.0.0 -> v2.0.0)
  updated:    .opencode/command/classify-docs.md (v1.0.0 -> v2.0.0)

4 updated.

$ gaze init                         # Already current — no changes
Gaze OpenCode integration already up to date:
  up to date: .opencode/agents/gaze-reporter.md
  up to date: .opencode/agents/doc-classifier.md
  up to date: .opencode/command/gaze.md
  up to date: .opencode/command/classify-docs.md

$ gaze init --force                 # Force still works as before
Gaze OpenCode integration initialized:
  overwritten: .opencode/agents/gaze-reporter.md
  overwritten: .opencode/agents/doc-classifier.md
  overwritten: .opencode/command/gaze.md
  overwritten: .opencode/command/classify-docs.md
```

## Key Changes

1. **No more silent skipping**: Files with outdated version markers are replaced automatically.
2. **Version transition visible**: Updated files show the old and new version in the output.
3. **`--force` unchanged**: Still overwrites everything unconditionally.
4. **Dev builds always update**: Running a `dev` build always refreshes all files.

## Testing

```bash
# Run unit tests for scaffold package
go test -race -count=1 ./internal/scaffold/...

# Run integration tests for init command
go test -race -count=1 -run TestRunInit ./cmd/gaze/...

# Run all tests
go test -race -count=1 -short ./...
```

## Files Modified

| File | Change |
|------|--------|
| `internal/scaffold/scaffold.go` | Add `extractVersion()`, modify `Run()` logic, update `Result` struct, update `printSummary()` |
| `internal/scaffold/scaffold_test.go` | Update existing tests, add new tests for update/up-to-date/dev/edge-case scenarios |
| `cmd/gaze/main_test.go` | Update `TestRunInit_ForceFlag` to cover new states |
