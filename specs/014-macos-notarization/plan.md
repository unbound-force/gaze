# Implementation Plan: macOS Code Signing and Notarization

**Branch**: `014-macos-notarization` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-macos-notarization/spec.md`

## Summary

Add macOS code signing and notarization to the existing GoReleaser v2 release pipeline so that Homebrew-installed gaze binaries are trusted by macOS Gatekeeper. GoReleaser v2 OSS has built-in cross-platform notarization (importing quill as a Go library), enabling signing and notarization from the existing `ubuntu-latest` CI runner with no additional tooling. The implementation requires adding a `notarize.macos` section to `.goreleaser.yaml` and passing 5 Apple credential secrets through the GitHub Actions workflow.

## Technical Context

**Language/Version**: Go 1.24+ (no Go code changes; YAML/workflow configuration only)
**Primary Dependencies**: GoReleaser v2 (OSS), quill (embedded in GoReleaser as a Go library)
**Storage**: N/A
**Testing**: Manual verification via test release + `spctl --assess` on macOS; CI dry-run validation
**Target Platform**: GitHub Actions `ubuntu-latest` runner (unchanged)
**Project Type**: Single CLI binary — release pipeline configuration change
**Performance Goals**: Signing and notarization complete within 30 minutes
**Constraints**: Must remain on `ubuntu-latest` runner; no macOS runner cost; graceful degradation when secrets absent
**Scale/Scope**: 2 files modified (`.goreleaser.yaml`, `.github/workflows/release.yml`); 5 GitHub secrets to configure

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Research Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | This feature does not alter Gaze's analysis engine, side effect detection, or reporting. It only modifies the release distribution pipeline. No risk of introducing false positives or false negatives. |
| **II. Minimal Assumptions** | PASS | No changes to how Gaze analyzes host projects. No new user-facing annotations or restructuring required. The signing is transparent to end users — they simply stop seeing Gatekeeper warnings. The only new assumption is that the maintainer has Apple Developer credentials, which is documented in the Assumptions section. |
| **III. Actionable Output** | PASS | No changes to Gaze's output, reports, or metrics. The feature exclusively affects binary distribution trust. |

**Gate result**: PASS — All three principles satisfied. This feature operates entirely outside the analysis domain and has no impact on accuracy, assumptions, or output quality.

### Post-Design Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | Design confirms: zero changes to analysis engine, side effect detection, or reporting. Only 2 YAML configuration files modified. No new Go code. No risk to accuracy. |
| **II. Minimal Assumptions** | PASS | Design confirms: no new user-facing requirements. The only new assumption (Apple Developer credentials) is a maintainer prerequisite, not an end-user requirement. Users continue using `brew install --cask gaze` with no workflow changes. The `isEnvSet` guard ensures graceful degradation. |
| **III. Actionable Output** | PASS | Design confirms: no changes to Gaze's output formats, reports, or metrics. The feature is invisible to users — they simply stop seeing Gatekeeper warnings. |

**Post-design gate result**: PASS — No constitution concerns introduced by the design.

## Project Structure

### Documentation (this feature)

```text
specs/014-macos-notarization/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: GoReleaser + quill research findings
├── data-model.md        # Phase 1: Configuration entities and secrets model
├── quickstart.md        # Phase 1: Step-by-step setup guide
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
.goreleaser.yaml                     # Add notarize.macos section
.github/workflows/release.yml       # Add 5 Apple credential env vars
```

**Structure Decision**: No new source files or directories. This feature modifies two existing configuration files at the repository root. No Go code changes required.

## Complexity Tracking

> No Constitution Check violations. No complexity justification needed.
