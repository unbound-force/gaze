# Quickstart: Output Voice & Style Standardization

**Branch**: `011-output-voice-style` | **Date**: 2026-03-01

## What This Feature Does

Rewrites the gaze-reporter agent prompt to produce fun, emoji-rich output instead of the clinical, emoji-free output from spec 010. The Go CLI formatters are unchanged — only the OpenCode agent's markdown report output changes.

## Files Changed

| File | Change Type |
|------|-------------|
| `.opencode/agents/gaze-reporter.md` | Rewrite Output Format, inline tone directives, Example Output |
| `internal/scaffold/assets/agents/gaze-reporter.md` | Identical rewrite (must stay byte-identical to active prompt) |
| `AGENTS.md` | Update Recent Changes and spec listing |
| `specs/010-report-voice-refinement/*` | Delete entire directory (7 files) |

## How to Verify

After implementation, run the `/gaze` command in OpenCode and check that the output:

1. Has emoji-prefixed section headers (🔍, 📊, 🧪, 🏷️, 🏥)
2. Uses colored circle emojis for severity (🟢, 🟡, 🔴, ⚪)
3. Uses letter grades (A through F) instead of word grades (Poor/Fair/Good/Strong/Excellent)
4. Has numbered recommendations with severity emoji prefixes
5. Contains no exclamation marks (or at most one), no slang, no first-person pronouns
6. Omits empty sections silently (no placeholders)

## Key Design Decisions

1. **Closed emoji vocabulary**: Only 10 specific emojis are allowed. No emoji creep.
2. **Grade boundaries**: B+ and above = 🟢, B through C = 🟡, C- and below = 🔴.
3. **Tone defined by bans**: Anti-patterns (exclamation marks, slang, puns, first-person) are banned. No required word list.
4. **Spec 010 fully deleted**: Not archived, not deprecated — removed entirely.
5. **Dual-file sync**: Both copies of gaze-reporter.md must be identical (active + scaffolded).

## Prerequisites

- Rebase branch onto current `main` before implementing spec 010 deletions (spec 010 files exist on `main` but not on this branch)
- No Go code changes required — this is purely a markdown prompt rewrite
