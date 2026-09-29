# Installation

Gaze is a single binary with no runtime dependencies. Choose the installation method that fits your workflow.

## Prerequisites

- **Go 1.25.0 or later** is required for `go install` and building from source. Check your version with `go version`.
- **macOS (darwin)** or **Linux** on **amd64** or **arm64** architectures. Windows is not currently supported.

## Homebrew (recommended)

The fastest way to install on macOS or Linux:

```bash
brew install unbound-force/tap/gaze
```

Homebrew binaries for macOS are code-signed with an Apple Developer ID certificate and notarized by Apple's notary service. macOS Gatekeeper trusts the binary on first run -- no security overrides needed.

## Go Install

If you have Go 1.25.0+ installed:

```bash
go install github.com/unbound-force/gaze/v2/cmd/gaze@latest
```

This places the `gaze` binary in your `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is on your `$PATH`.

## Build from Source

Clone the repository and build:

```bash
git clone https://github.com/unbound-force/gaze.git
cd gaze
go build -o gaze ./cmd/gaze
```

The resulting `gaze` binary is in the current directory. Move it somewhere on your `$PATH` (e.g., `/usr/local/bin/`) or run it directly with `./gaze`.

## Supported Platforms

Pre-built binaries are published for every release:

| OS    | Architecture | Archive                          |
|-------|-------------|----------------------------------|
| macOS | amd64       | `gaze_<version>_darwin_amd64.tar.gz` |
| macOS | arm64       | `gaze_<version>_darwin_arm64.tar.gz` |
| Linux | amd64       | `gaze_<version>_linux_amd64.tar.gz`  |
| Linux | arm64       | `gaze_<version>_linux_arm64.tar.gz`  |

Download archives from the [GitHub Releases](https://github.com/unbound-force/gaze/releases) page.

## Verify Your Installation

After installing, confirm the binary is available:

```bash
gaze --version
```

You should see output like:

```text
gaze version v0.x.x
```

If the command is not found, verify that the binary's location is on your `$PATH`.

## Next Steps

- [Concepts: Why Line Coverage Isn't Enough](concepts.md) -- understand the problem Gaze solves
- [Quickstart](quickstart.md) -- produce your first analysis in under 10 minutes
