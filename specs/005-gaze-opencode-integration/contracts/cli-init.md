# Contract: `gaze init` CLI Subcommand

**Date**: 2026-02-23 | **Branch**: `005-gaze-opencode-integration`

## Command Signature

```
gaze init [--force]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | false | Overwrite existing files |

## Behavior

### Success Output

```
Gaze OpenCode integration initialized:
  created: .opencode/agents/gaze-reporter.md
  created: .opencode/agents/doc-classifier.md
  created: .opencode/command/gaze.md
  created: .opencode/command/classify-docs.md

Run /gaze in OpenCode to generate quality reports.
```

### Partial Output (some files exist)

```
Gaze OpenCode integration initialized:
  created: .opencode/agents/gaze-reporter.md
  skipped: .opencode/agents/doc-classifier.md (already exists)
  created: .opencode/command/gaze.md
  created: .opencode/command/classify-docs.md

Run /gaze in OpenCode to generate quality reports.
1 file skipped (use --force to overwrite).
```

### Force Output

```
Gaze OpenCode integration initialized:
  overwritten: .opencode/agents/gaze-reporter.md
  overwritten: .opencode/agents/doc-classifier.md
  overwritten: .opencode/command/gaze.md
  overwritten: .opencode/command/classify-docs.md

Run /gaze in OpenCode to generate quality reports.
```

### Warning (no go.mod)

```
Warning: no go.mod found in current directory.
Gaze works best in a Go module root.

Gaze OpenCode integration initialized:
  created: ...
```

### Error (write failure)

```
Error: creating .opencode/agents/gaze-reporter.md: permission denied
```

Exit code 1.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (files created/skipped/overwritten) |
| 1 | Error (filesystem failure) |

## Version Marker

Every scaffolded file has the following prepended as the first
line:

```
<!-- scaffolded by gaze v0.1.0 -->
```

For development builds:

```
<!-- scaffolded by gaze dev -->
```
