# Contract: internal/scaffold Package API

**Date**: 2026-02-23 | **Branch**: `005-gaze-opencode-integration`

## Package Declaration

```go
// Package scaffold embeds distributable OpenCode agent and command
// files and writes them to a target project directory.
package scaffold
```

## Public API

### Run

```go
// Run scaffolds OpenCode agent and command files into the target
// directory. It creates .opencode/agents/ and .opencode/command/
// subdirectories and writes the embedded quality-reporting files.
//
// Each file is prepended with a version marker comment:
//   <!-- scaffolded by gaze vX.Y.Z -->
//
// If a file already exists and opts.Force is false, the file is
// skipped. If opts.Force is true, the file is overwritten.
//
// Run returns a Result summarizing what was created, skipped, or
// overwritten.
func Run(opts Options) (*Result, error)
```

**Preconditions**:
- `opts.TargetDir` must be a writable directory (or creatable).
- `opts.Version` should be set; defaults to `"dev"` if empty.
- `opts.Stdout` should be set; defaults to `os.Stdout` if nil.

**Postconditions**:
- On success, exactly 4 files exist under
  `opts.TargetDir/.opencode/` (created or pre-existing).
- `Result.Created + Result.Skipped + Result.Overwritten` accounts
  for all 4 embedded assets.
- No files outside `.opencode/` are modified.

**Error conditions**:
- Directory creation fails (permissions, disk full).
- File write fails (permissions, disk full).
- Embedded FS read fails (should never happen at runtime).

### Options

```go
// Options configures the scaffold operation.
type Options struct {
    TargetDir string    // Root directory (default: CWD)
    Force     bool      // Overwrite existing files
    Version   string    // Version string for marker (default: "dev")
    Stdout    io.Writer // Output writer (default: os.Stdout)
}
```

### Result

```go
// Result reports what the scaffold operation did.
type Result struct {
    Created     []string // Files written for the first time
    Skipped     []string // Files that existed (Force=false)
    Overwritten []string // Files replaced (Force=true)
}
```

## Embedded Assets

```go
//go:embed assets/*
var assets embed.FS
```

The `assets/` directory mirrors the `.opencode/` structure:

| Embedded Path | Output Path |
|---------------|-------------|
| `assets/agents/gaze-reporter.md` | `.opencode/agents/gaze-reporter.md` |
| `assets/agents/doc-classifier.md` | `.opencode/agents/doc-classifier.md` |
| `assets/command/gaze.md` | `.opencode/command/gaze.md` |
| `assets/command/classify-docs.md` | `.opencode/command/classify-docs.md` |

## CLI Integration

```go
// In cmd/gaze/main.go:

initCmd := &cobra.Command{
    Use:   "init",
    Short: "Scaffold OpenCode agents and commands for Gaze",
    Long: `Initialize OpenCode integration in the current directory.

Creates .opencode/agents/ and .opencode/command/ directories with
Gaze's quality reporting agent and commands. After running this,
you can use /gaze in OpenCode to generate quality reports.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        force, _ := cmd.Flags().GetBool("force")
        cwd, err := os.Getwd()
        if err != nil {
            return fmt.Errorf("getting working directory: %w", err)
        }
        result, err := scaffold.Run(scaffold.Options{
            TargetDir: cwd,
            Force:     force,
            Version:   version,
            Stdout:    cmd.OutOrStdout(),
        })
        if err != nil {
            return err
        }
        // Print summary (handled by Run via Stdout)
        _ = result
        return nil
    },
}
initCmd.Flags().Bool("force", false, "Overwrite existing files")
root.AddCommand(initCmd)
```

## Test Contract

```go
// TestRun_CreatesFiles verifies SC-001.
func TestRun_CreatesFiles(t *testing.T)

// TestRun_SkipsExisting verifies SC-002.
func TestRun_SkipsExisting(t *testing.T)

// TestRun_ForceOverwrites verifies SC-003.
func TestRun_ForceOverwrites(t *testing.T)

// TestRun_VersionMarker verifies SC-004.
func TestRun_VersionMarker(t *testing.T)

// TestEmbeddedAssetsMatchSource verifies SC-005 / FR-017.
func TestEmbeddedAssetsMatchSource(t *testing.T)

// TestRun_NoGoMod_PrintsWarning verifies US4-AS6.
func TestRun_NoGoMod_PrintsWarning(t *testing.T)
```
