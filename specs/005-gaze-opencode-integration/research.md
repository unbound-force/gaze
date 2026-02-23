# Research: Gaze OpenCode Integration & Distribution

**Date**: 2026-02-23 | **Branch**: `005-gaze-opencode-integration`

## R-001: Go Module Path Migration

**Decision**: Use `sed` for bulk replacement + `go mod edit` for
`go.mod` + `go mod tidy` for `go.sum` regeneration.

**Rationale**: The migration is a mechanical string replacement
across 28 `.go` files (83 import lines), 2 testdata fixtures,
4 string literals (JSON Schema `$id` URLs, test assertions), and
7 markdown docs. No specialized tooling needed — `sed` covers all
cases. `gomove` is abandoned and unreliable. `gofmt -r` doesn't
work on import path strings.

**Alternatives considered**:
- `marwan-at-work/mod` — purpose-built for module path migration,
  but overkill for a non-v2 rename where `sed` suffices.
- `gomove` — abandoned/archived, unreliable with modern Go modules.
- `gofmt -r` — only works on Go expressions, not import path
  strings.

**Key findings**:
- `go.sum` does not contain self-references; `go mod tidy`
  regenerates it automatically.
- No `retract` directive needed — zero published version tags
  exist.
- No vanity import comments needed — these are a pre-modules
  mechanism.
- No old-repo redirect needed — no published versions to redirect.
- Testdata fixtures with import paths MUST be updated (2 files).
- String literals in `internal/report/schema.go` (JSON Schema
  `$id` URLs) and `internal/loader/loader_test.go` (hardcoded
  paths) MUST also be updated.

**Migration script**:
```bash
go mod edit -module github.com/unbound-force/gaze
find . -name '*.go' -exec sed -i '' \
  's|github.com/unbound-force/gaze|github.com/unbound-force/gaze|g' {} +
find . -name '*.md' -exec sed -i '' \
  's|github.com/unbound-force/gaze|github.com/unbound-force/gaze|g' {} +
go mod tidy
go build ./...
go test -race -count=1 -short ./...
```

---

## R-002: GoReleaser v2 Configuration

**Decision**: Use GoReleaser v2 with `homebrew_casks` (not the
deprecated `brews`), `goreleaser/goreleaser-action@v7`, and
`CGO_ENABLED=0` for cross-platform builds.

**Rationale**: GoReleaser v2 is the current stable release.
v2.10+ deprecated `brews` in favor of `homebrew_casks` which
generates proper Homebrew Casks. The `@v7` action is the current
recommended version.

**Alternatives considered**:
- GoReleaser v1 — legacy, requires pinning an older action
  version, missing v2 features.
- `brews` key — deprecated since v2.10, generates "hackyish"
  formulas. `homebrew_casks` is the replacement.
- `goreleaser-action@v6` — works but `@v7` is the current
  recommended version per docs.

**Key v1-to-v2 breaking changes**:
| v1 Key | v2 Key |
|--------|--------|
| `build:` (singular) | `builds:` (plural) |
| `brews:` | `homebrew_casks:` |
| `brews.tap:` | `repository:` |
| `archives.format:` | `archives.formats:` (list) |
| `--rm-dist` | `--clean` |
| `changelog.skip:` | `changelog.disable:` |

**Configuration requirements**:
- `version: 2` mandatory at top of config.
- `CGO_ENABLED=0` required for cross-compilation (no C deps in
  Gaze — all pure Go).
- `ldflags`: default `-s -w -X main.version={{.Version}}` plus
  `-X main.commit={{.Commit}}` and `-X main.date={{.CommitDate}}`
  for reproducible builds.
- `fetch-depth: 0` required in checkout step for changelog
  generation.
- `homebrew_casks` directory must be `Casks` (not `Formula`).
- Cross-repo tap publishing requires a PAT (not `GITHUB_TOKEN`)
  stored as `HOMEBREW_TAP_GITHUB_TOKEN`.

**Changelog**: Use `use: github` for GitHub username attribution.
Group by conventional commit prefix (`feat:`, `fix:`, `docs:`).
Exclude `chore:` commits.

---

## R-003: Go `embed.FS` for Scaffold Assets

**Decision**: Copy `.opencode/` files into
`internal/scaffold/assets/` and embed from there. Verify
consistency with a Go test that compares embedded content against
the `.opencode/` source files.

**Rationale**: Go's `embed.FS` cannot reference files outside the
package directory — `..` path elements are forbidden, and symlinks
are also explicitly disallowed. Files must physically exist under
the embedding package's directory tree.

**Alternatives considered**:
- Embed directly from `../../.opencode/` — not possible, Go
  compiler forbids `..` in embed directives.
- Symlinks from assets/ to .opencode/ — explicitly forbidden by
  the embed package.
- `go generate` with CI diff check — viable but adds workflow
  complexity. A Go test is simpler and runs with `go test ./...`.

**Key findings**:
- `embed.FS` does NOT preserve file permissions (always 0444/0555).
  Must set explicitly on write (0644 for markdown files).
- `embed.FS` preserves file content byte-for-byte (no newline
  conversion). Use `.gitattributes eol=lf` for consistency.
- `embed.FS` does NOT preserve modification times (always zero).
- Use `fs.WalkDir` + `embed.FS.ReadFile` + `os.WriteFile` to
  traverse and write embedded files.
- Version marker (`<!-- scaffolded by gaze vX.Y.Z -->`) must be
  prepended at scaffold time, not stored in embedded source
  (to preserve drift detection accuracy).
- `ReadFile` returns a copy of bytes, safe to mutate for
  prepending.

**Drift detection test pattern**:
```go
func TestEmbeddedAssetsMatchSource(t *testing.T) {
    // Walk internal/scaffold/assets/
    // Compare each file against ../../.opencode/ counterpart
    // Fail if content differs
}
```

---

## R-004: Homebrew Tap Repository

**Decision**: Use `unbound-force/homebrew-tap` with a `Casks/`
directory. GoReleaser auto-publishes the cask on each release.

**Rationale**: Standard Homebrew convention is
`org/homebrew-tap` for a multi-tool tap. The `Casks/` directory
is required by GoReleaser v2's `homebrew_casks` feature. Users
install via `brew install unbound-force/tap/gaze`.

**Prerequisites** (manual, one-time):
1. Create `unbound-force/homebrew-tap` repository on GitHub.
2. Create a PAT with `repo` scope that can push to the tap repo.
3. Add the PAT as `HOMEBREW_TAP_GITHUB_TOKEN` secret in the
   `unbound-force/gaze` repository settings.

**Key findings**:
- If migrating from `brews` to `homebrew_casks` later, create a
  `tap_migrations.json` in the tap repo root.
- `skip_upload: auto` skips pre-release uploads automatically.
- For unsigned binaries on macOS, consider adding a post-install
  hook to remove quarantine xattr. (Evaluate after first release.)

---

## R-005: Version Injection

**Decision**: Add `commit` and `date` variables alongside the
existing `version` variable in `cmd/gaze/main.go`. GoReleaser
injects all three via ldflags.

**Rationale**: The `version` variable already exists (`var version
= "dev"` at line 34). Adding `commit` and `date` provides build
provenance at zero cost — GoReleaser injects them by default.
Use `{{.CommitDate}}` (not `{{.Date}}`) for reproducible builds.

**Go source changes needed**:
```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

**Current state**: Only `version` exists. It flows to
`analysis.Options.Version`, `report.WriteJSON()`, and
`quality.Options.Version`. The `commit` and `date` variables
are new and only used by `gaze --version` output.

---

## R-006: First Release Version

**Decision**: `v0.1.0` — pre-1.0 signals API instability.

**Rationale**: The project has zero published tags. `v0.1.0`
follows semantic versioning convention for initial releases where
the API may still change. The constitution requires semantic
versioning (Development Workflow section).
