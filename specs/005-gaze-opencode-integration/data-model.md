# Data Model: Gaze OpenCode Integration & Distribution

**Date**: 2026-02-23 | **Branch**: `005-gaze-opencode-integration`

## Entities

### ScaffoldOptions

Configuration for the `gaze init` subcommand.

```go
// Options configures the scaffold operation.
type Options struct {
    // TargetDir is the root directory to scaffold into.
    // Defaults to the current working directory.
    TargetDir string

    // Force overwrites existing files when true.
    // When false, existing files are skipped.
    Force bool

    // Version is the gaze version string to embed in the
    // version marker comment. Set by ldflags at build time.
    // Defaults to "dev" for development builds.
    Version string

    // Stdout is the writer for summary output.
    // Defaults to os.Stdout.
    Stdout io.Writer
}
```

**Validation rules**:
- `TargetDir` must be a valid directory path (created if absent).
- `Version` is never empty; falls back to `"dev"`.
- `Force` has no validation (boolean).

### ScaffoldResult

Outcome of the scaffold operation. Returned by `Run()`.

```go
// Result reports what the scaffold operation did.
type Result struct {
    // Created lists files that were written for the first time.
    Created []string

    // Skipped lists files that already existed and were not
    // overwritten (Force was false).
    Skipped []string

    // Overwritten lists files that existed and were replaced
    // (Force was true).
    Overwritten []string
}
```

**Invariants**:
- `len(Created) + len(Skipped) + len(Overwritten)` equals the
  total number of embedded assets (currently 4).
- `Overwritten` is always empty when `Force` is false.
- `Skipped` is always empty when `Force` is true.

### AssetFile (internal)

Represents a single distributable file within the embedded
filesystem. Not exported — used internally by the scaffold
package.

```go
// assetFile represents one embedded template file.
type assetFile struct {
    // RelPath is the path relative to .opencode/, e.g.,
    // "agents/gaze-reporter.md" or "command/gaze.md".
    RelPath string

    // Content is the raw file content from embed.FS.
    Content []byte
}
```

**Relationships**:
- `assetFile.RelPath` determines the output path under
  `Options.TargetDir/.opencode/`.
- `assetFile.Content` is compared against the `.opencode/`
  source file in the drift detection test.

## Embedded Asset Manifest

The embedded filesystem contains exactly 4 files:

```text
internal/scaffold/assets/
├── agents/
│   ├── gaze-reporter.md     # Quality report agent
│   └── doc-classifier.md    # Document-enhanced classifier
└── command/
    ├── gaze.md              # /gaze command
    └── classify-docs.md     # /classify-docs command
```

**Constraint**: This manifest is verified by the drift detection
test. Adding or removing files requires updating both the
`.opencode/` directory and the `assets/` directory.

## State Transitions

The scaffold operation is stateless — it reads the embedded
filesystem and writes files to disk. There are no lifecycle
transitions, database records, or persistent state.

```text
gaze init
  │
  ├─ For each embedded asset:
  │    ├─ File exists? ──┐
  │    │                  ├─ Force=true → Overwrite → Overwritten[]
  │    │                  └─ Force=false → Skip → Skipped[]
  │    └─ File absent? → Create dirs + Write → Created[]
  │
  └─ Return Result
```

## Relationships to Existing Entities

```text
┌──────────────────┐
│ cmd/gaze/main.go │
│ (init subcommand)│
└────────┬─────────┘
         │ calls
         ▼
┌──────────────────┐     ┌──────────────────┐
│ scaffold.Run()   │────>│ scaffold.Options  │
│ scaffold.Result  │     │ scaffold.assetFile│
└────────┬─────────┘     └──────────────────┘
         │ reads
         ▼
┌──────────────────┐
│ embed.FS         │
│ (assets/)        │
└──────────────────┘
```

The scaffold package has no dependency on any other internal
package. It only uses the standard library (`embed`, `io/fs`,
`os`, `path/filepath`, `fmt`, `io`).
