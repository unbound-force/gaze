# Implementation Plan: Native macOS Code Signing and Notarization

**Branch**: `015-native-macos-signing` | **Date**: 2026-03-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-native-macos-signing/spec.md`

## Summary

Replace the broken quill-based cross-platform signing (spec 014) with Apple's native `codesign` and `notarytool` running on a `macos-latest` CI runner. The build stays on `ubuntu-latest`. A new `sign-macos` job downloads unsigned darwin archives from the release, imports the .p12 certificate into a temporary Keychain, signs each binary with `codesign`, submits for notarization via `notarytool --wait`, re-archives, and replaces the release assets. Checksums are updated to reflect the signed archives. The existing quill-based `notarize.macos` section is removed from `.goreleaser.yaml`.

## Technical Context

**Language/Version**: Go 1.24+ (no Go code changes; YAML/workflow configuration only)
**Primary Dependencies**: GitHub Actions, `codesign` (macOS native), `xcrun notarytool` (macOS native), `security` (macOS Keychain), `gh` CLI (GitHub)
**Storage**: N/A
**Testing**: Manual verification via test release + `codesign --verify` + `spctl --assess` on macOS
**Target Platform**: GitHub Actions `ubuntu-latest` (build) + `macos-latest` (sign)
**Project Type**: Single CLI binary — release pipeline configuration change
**Performance Goals**: Signing + notarization complete within 30 minutes
**Constraints**: Public repo (free macOS minutes); same 5 secrets from spec 014; brief unsigned window acceptable
**Scale/Scope**: 2 files modified (`.goreleaser.yaml`, `.github/workflows/release.yml`); no new files outside specs/

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Research Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | This feature does not alter Gaze's analysis engine, side effect detection, or reporting. It only modifies the release distribution pipeline. No risk of introducing false positives or false negatives. |
| **II. Minimal Assumptions** | PASS | No changes to how Gaze analyzes host projects. No new user-facing annotations or restructuring required. The signing is transparent to end users. The only assumption is that the maintainer has Apple Developer credentials (already documented and configured from spec 014). |
| **III. Actionable Output** | PASS | No changes to Gaze's output, reports, or metrics. The feature exclusively affects binary distribution trust. |

**Gate result**: PASS — All three principles satisfied. This feature operates entirely outside the analysis domain.

### Post-Design Check

| Principle | Status | Assessment |
|-----------|--------|------------|
| **I. Accuracy** | PASS | Design confirms: zero changes to analysis engine. Only workflow YAML modified. No new Go code. |
| **II. Minimal Assumptions** | PASS | Design confirms: no new user-facing requirements. Same 5 secrets from spec 014 reused. Users continue using `brew install --cask gaze` unchanged. Signing job conditional on secret presence — skips cleanly when absent. |
| **III. Actionable Output** | PASS | Design confirms: no changes to output formats, reports, or metrics. Feature is invisible to users — they simply stop seeing Gatekeeper warnings. |

**Post-design gate result**: PASS — No constitution concerns.

## Project Structure

### Documentation (this feature)

```text
specs/015-native-macos-signing/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: Native signing research
├── data-model.md        # Phase 1: Workflow entities and secrets model
├── quickstart.md        # Phase 1: Verification guide
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
.goreleaser.yaml                     # Remove notarize.macos section
.github/workflows/release.yml       # Remove MACOS_* env vars from GoReleaser step;
                                     # add sign-macos job with codesign + notarytool
```

**Structure Decision**: No new source files or directories. This feature modifies two existing configuration files. The GoReleaser config gets simpler (removal). The workflow file gains a new job. No Go code changes.

## Complexity Tracking

> No Constitution Check violations. No complexity justification needed.
