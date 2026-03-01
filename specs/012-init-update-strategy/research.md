# Research: Init Update Strategy

**Branch**: `012-init-update-strategy` | **Date**: 2026-03-01

## R1: Version Marker Extraction Strategy

### Decision

Extract the version string from the first line of an existing scaffolded file using prefix/suffix stripping on the known marker format `<!-- scaffolded by gaze VERSION -->`.

### Rationale

The version marker format is a project-internal convention, fully controlled by the `versionMarker()` function in `scaffold.go`. It has a fixed prefix (`<!-- scaffolded by gaze `) and suffix (` -->`). Simple string operations (TrimPrefix/TrimSuffix) are sufficient — no regex, no HTML parsing, no multi-line scanning. This keeps the implementation minimal and avoids new dependencies.

### Alternatives Considered

1. **Regex parsing** (`regexp.MustCompile`): More flexible but overkill for a fixed format. Adds compilation cost (trivial) and cognitive overhead for readers. Rejected for unnecessary complexity.
2. **Read entire file and scan all lines**: Allows the marker to appear anywhere. Rejected because the marker is always the first line (guaranteed by `Run()`), so reading beyond line 1 would be wasteful and introduce ambiguity about which line "wins."
3. **Structured metadata file** (e.g., `.opencode/.gaze-manifest.json`): Would store version per file in a separate manifest. Rejected because it adds a new file to manage, introduces sync issues between manifest and actual files, and is a larger scope change than warranted.

## R2: Result Struct Extension

### Decision

Replace the `Skipped []string` field with two new fields: `Updated []string` and `UpToDate []string`. The `Updated` field captures files whose version marker differs from the current version and were replaced. The `UpToDate` field captures files whose version marker matches the current version and were left untouched.

Add an `UpdatedFrom map[string]string` field to record the previous version for each updated file (keyed by relative path), enabling the "v1.0.0 -> v2.0.0" output required by FR-007.

### Rationale

The existing `Skipped` field conflates two distinct states: "file is current" and "file is outdated but I didn't touch it." The new design makes the distinction explicit in the data model, which simplifies both the summary output and test assertions. The `UpdatedFrom` map avoids passing version information through side channels.

### Alternatives Considered

1. **Keep `Skipped` and add `Updated` alongside**: Would mean `Skipped` now only contains "up to date" files, which is a misleading name. Rejected for naming confusion.
2. **Use a single `FileResult` struct per file** (disposition enum + metadata): Cleaner object model but a larger refactor of all callers and tests. Deferred — the current slice-based approach is consistent with the existing code style.
3. **Store old version in a separate return value**: More fragile coupling. Rejected in favor of the self-contained `UpdatedFrom` map.

## R3: Dev Version Behavior

### Decision

When `opts.Version` is `"dev"`, treat all existing files as outdated regardless of their on-disk version marker. This means dev builds always rewrite all files (equivalent to implicit `--force` for staleness, but still tracked as "updated" rather than "overwritten").

### Rationale

Developers working on gaze itself frequently change the embedded asset files. Requiring `--force` every time they want to test updated prompts creates friction. Since dev builds have no meaningful version identity, always refreshing is the safe and simple choice. The user explicitly chose this behavior during specification clarification.

### Alternatives Considered

1. **Dev never updates**: Requires `--force` for every change during development. Rejected for developer friction.
2. **Dev uses content comparison**: Reads on-disk file and compares byte-for-byte with embedded content. More precise but adds I/O cost and complexity for a development-only scenario. Rejected for over-engineering.

## R4: printSummary Output Format

### Decision

Extend `printSummary` to handle the four file states with these output formats:

- `  created:     .opencode/agents/gaze-reporter.md`
- `  updated:     .opencode/agents/gaze-reporter.md (v1.0.0 -> v2.0.0)`
- `  up to date:  .opencode/agents/gaze-reporter.md`
- `  overwritten: .opencode/agents/gaze-reporter.md`

The summary footer shows counts: `"1 created, 2 updated, 1 up to date."` Up-to-date files are listed individually (not suppressed) so the user sees a complete manifest of all scaffolded files. The existing `"use --force to overwrite"` hint is removed since files are now updated automatically; it is no longer needed.

### Rationale

Individual listing of all files (including up-to-date) provides full transparency at minimal cost (4 files total). The version transition annotation on "updated" lines satisfies FR-007. Removing the `--force` hint for the non-force path avoids confusing users into thinking they need to force something that already happened.

### Alternatives Considered

1. **Suppress up-to-date files from individual listing**: Shorter output but users lose visibility into which files are managed. Rejected for reduced transparency.
2. **Use color/emoji for status**: Adds visual clarity but increases implementation scope (terminal color detection, Lipgloss dependency in scaffold package). Deferred — can be added later without changing the data model.
3. **Machine-readable JSON output mode**: Useful for CI but out of scope for this feature. The `Result` struct is already returned programmatically; JSON formatting can be added independently.

## R5: Backward Compatibility of Existing Tests

### Decision

Existing tests that assert on `result.Skipped` must be updated to use `result.UpToDate` (for same-version re-runs) or `result.Updated` (for cross-version re-runs). The test names and comments reference spec success criteria (SC-xxx) and should be updated to reflect the new behavior semantics.

### Rationale

The `Skipped` field is being removed. Tests that use it will fail at compile time, forcing explicit review. This is preferable to silently leaving stale test logic. The number of affected tests is small (3 in scaffold_test.go, 1 in main_test.go).

### Alternatives Considered

1. **Keep `Skipped` as a deprecated alias**: Avoids test churn but leaves confusing dead code. Rejected for zero-waste mandate.
2. **Add `Updated` without removing `Skipped`**: Both fields would need to be populated, doubling bookkeeping. Rejected for unnecessary complexity.
