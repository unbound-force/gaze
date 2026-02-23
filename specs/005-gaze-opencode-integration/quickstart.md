# Quickstart: Gaze OpenCode Integration

## For Users (Installing in Your Go Project)

### Option 1: Homebrew (recommended)

```bash
brew install unbound-force/tap/gaze
```

### Option 2: Go Install

```bash
go install github.com/unbound-force/gaze/cmd/gaze@latest
```

### Setup OpenCode Integration

```bash
cd /path/to/your-go-project
gaze init
```

This creates 4 files in `.opencode/`:
- `.opencode/agents/gaze-reporter.md` — Quality report agent
- `.opencode/agents/doc-classifier.md` — Doc-enhanced classifier
- `.opencode/command/gaze.md` — `/gaze` command
- `.opencode/command/classify-docs.md` — `/classify-docs` command

### Usage in OpenCode

```
/gaze ./...                     # Full quality report
/gaze crap ./internal/store     # CRAP scores only
/gaze quality ./pkg/api         # Test quality metrics only
/classify-docs ./internal/db    # Doc-enhanced classification
```

## For Developers (Working on Gaze Itself)

### Prerequisites

- Go 1.24+
- GoReleaser v2 (for release testing)

### Build

```bash
go build -o gaze ./cmd/gaze
```

### Test

```bash
go test -race -count=1 -short ./...
```

### Test Release (Local)

```bash
goreleaser release --snapshot --clean
```

This produces archives in `dist/` without publishing.

### Homebrew Tap Prerequisites (One-Time Setup)

Before the first release, the following manual steps are required:

1. **Create the tap repository**: Create `unbound-force/homebrew-tap`
   on GitHub. This is where GoReleaser publishes the Homebrew cask
   formula on each release.

2. **Create a Personal Access Token (PAT)**: Generate a GitHub PAT
   with `repo` scope that has push access to
   `unbound-force/homebrew-tap`.

3. **Add the secret**: In the `unbound-force/gaze` repository
   settings, add the PAT as a repository secret named
   `HOMEBREW_TAP_GITHUB_TOKEN`.

Without these prerequisites, the release workflow will build and
publish binaries to the GitHub Release page, but the Homebrew cask
update step will fail.

### Create a Release

```bash
git tag v0.1.0
git push origin v0.1.0
```

The GitHub Actions release workflow handles the rest:
1. Builds binaries for macOS + Linux (amd64 + arm64)
2. Creates a GitHub Release with archives and checksums
3. Updates the Homebrew cask in `unbound-force/homebrew-tap`

### Dogfooding

The `.opencode/` files in the Gaze repo are the source of truth.
They're used during development AND embedded into the binary for
distribution. To verify they're in sync:

```bash
go test ./internal/scaffold/...
```

### Module Path

The canonical module path is `github.com/unbound-force/gaze`.
All imports use this path. The repo at `jflowers/gaze` is a
fork — PRs go to `unbound-force/gaze`.
