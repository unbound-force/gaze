# Contract: OpenCode adapter (structured events, no tools)

Covers FR-014, FR-016, and FR-037. The schema was verified against
opencode 1.19.0; see research R1 and R2.

## Invocation

```text
opencode run --dir <tmpDir> --agent gaze-headless --format json [-m <model>] ""
```

The payload goes on stdin. Arguments are separate Go strings with no
shell, unchanged from today.

## Agent file `<tmpDir>/.opencode/agents/gaze-headless.md` (0600; dir 0700)

```text
---
description: Gaze headless judgment call (no tools)
mode: primary
permission:
  "*": deny
---
<system prompt passed to Format>
```

The adapter owns this frontmatter. The system prompt is always
frontmatter-free (the report-mode prompt, or the `--ai-mapper` prompt).

## Extraction (`extractFinalText(stdout []byte) (string, error)`)

1. Split on `\n`, trim `\r`, and skip blank lines. Unmarshal each line
   into `openCodeEvent`. Lines that fail to unmarshal are skipped and
   counted.
2. Event handling:
   - `error` → fail: `opencode reported an error: <name>: <message>`.
     Redacted with `redactEnvValues`, then truncated to 512 bytes.
   - `tool_use` with `part.state.status == "completed"` → fail:
     `opencode executed tool "<tool>" although the gaze-headless agent denies all tools`.
   - `tool_use` with any other status → ignore.
   - `text` → record `{id, messageID, text}` unless `synthetic` or
     `ignored` is set. A duplicate `id` overwrites the earlier text
     and keeps the first-seen position.
   - `step_finish` → remember `part.messageID` as `lastStepMessage`.
   - `step_start`, `reasoning`, and unknown types → ignore.
3. If zero events were parsed: fail with
   `opencode produced no JSON events (expected --format json event stream; pinned opencode-ai 1.19.0)`.
4. `target = lastStepMessage`. If it is empty, use the `messageID` of
   the last recorded text part.
5. Answer = the texts with `messageID == target`, joined with `\n` in
   first-seen order. If the answer is empty after `TrimSpace`, fail
   with `opencode produced no final assistant text (FR-009)` plus
   `formatStderrSuffix(stderr)`.

The adapter returns the answer text. Validation of its shape happens
in the runner (`ParseAIAnswer`), the same as for every other adapter.

## Fake binary (`internal/aireport/testdata/fake_opencode`)

| Mode | Trigger | Behavior |
|------|---------|----------|
| Replay | `FAKE_OPENCODE_STREAM=<file>` | Asserts `run`, `--format json`, `--agent gaze-headless`, and that the agent file exists with `mode: primary` and `"*": deny`. Drains stdin, then writes the file to stdout verbatim. |
| Error | `--exit-error` (via wrapper) | unchanged |
| Empty | `--empty-output` (via wrapper) | unchanged (zero events → "no JSON events") |

Fixtures are in `internal/aireport/testdata/opencode_streams/`:
`answer_only.jsonl`, `narration_then_answer.jsonl`,
`incident.jsonl`, `chinese_answer.jsonl`, `tool_refused.jsonl`,
`tool_completed.jsonl`, `error_event.jsonl`, and `not_json.txt`.

## Test obligations

- Unit-test `extractFinalText` against each fixture's bytes.
- Integration (fake binary on PATH): `Format` returns only the final
  answer for `narration_then_answer` and fails for `tool_completed`
  and `error_event`.
- Incident regression (FR-036, SC-001): the runner in text format
  with `--ai=opencode` and `incident.jsonl` exits non-zero with rule
  `not_json`. `chinese_answer.jsonl` exits non-zero with rule
  `non_english`.
