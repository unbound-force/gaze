# Quickstart: Deterministic AI Report Rendering

Validation guide proving the feature end to end. Commands run from
the repository root. Contract details are in [contracts/](contracts/).
Types are in [data-model.md](data-model.md).

## Prerequisites

- Go 1.25+ (`go.mod` directive)
- `go build -o "${TMPDIR:-/tmp}/gaze" ./cmd/gaze`
- Optional: `opencode` 1.19.0 on PATH for scenario 7. Scenarios 1–6
  need no AI provider and no network.

## 1. Automated suites (CI parity)

```bash
go build ./cmd/gaze
go test -race -count=1 -short ./...
golangci-lint run
bash .github/scripts/test-validate-workflow-floor.sh   # unchanged; CI runs it on the Go 1.25 leg
```

Expected: all pass. The key new tests:

| Test (pattern) | Proves |
|----------------|--------|
| `go test ./internal/aireport -run 'Golden'` | SC-003: renderer matches golden files |
| `go test ./internal/aireport -run 'ByteIdentity'` | SC-002: 0 differing bytes outside the AI slots |
| `go test ./internal/aireport -run 'Grade'` | SC-004: every rubric boundary pinned |
| `go test ./internal/aireport -run 'ParseAIAnswer'` | FR-018: every validation rule |
| `go test ./internal/aireport -run 'Incident'` | SC-001 / FR-036: incident replay exits non-zero |
| `go test ./internal/aireport -run 'ExtractFinalText\|OpenCode'` | FR-016 / FR-037: structured extraction |
| `go test ./cmd/gaze -run 'Report'` | FR-038 / SC-005: CLI wiring, CLI golden, JSON↔markdown round trip |
| `go test ./internal/scaffold -run 'ToolOwned\|EmbeddedAssets\|ExampleReport'` | FR-034 / SC-010 |

## 2. Deterministic report with no AI (US3, SC-009)

```bash
unset OPENCODE_CONFIG_CONTENT
"${TMPDIR:-/tmp}/gaze" report --format=markdown ./... > /tmp/r1.md; echo "exit=$?"
"${TMPDIR:-/tmp}/gaze" report --format=markdown ./... > /tmp/r2.md
diff /tmp/r1.md /tmp/r2.md && echo IDENTICAL
```

Expected: exit 0. The report contains `# 🔍 Gaze Full Quality
Report`, `## 📊 CRAP Summary`, `## 🧪 Quality Summary`,
`## 🏷️ Classification Summary`, and `## 🏥 Overall Health Assessment`,
with a 5-row scorecard and no `gaze:ai-` markers. Runs on the same
day produce identical files.

## 3. Section selection (US3 AS-4, FR-026–FR-028)

```bash
gaze report --format=markdown --sections=crap ./internal/crap      # title "🔍 Gaze CRAP Report", CRAP section only
gaze report --format=markdown --sections=quality ./internal/quality # title "🔍 Gaze Quality Report"
gaze report --format=markdown --sections=bogus ./...                # exit≠0: valid sections are crap, quality, classification, health
gaze report --format=markdown --sections=crap --min-contract-coverage=50 ./...  # exit≠0 before analysis
```

## 4. Thresholds and step summary in markdown mode (US3 AS-2/3, FR-029)

```bash
export GITHUB_STEP_SUMMARY=/tmp/summary.md; : > "$GITHUB_STEP_SUMMARY"
gaze report --format=markdown --max-crapload=0 ./internal/crap; echo "exit=$?"
grep -c "Gaze Full Quality Report" "$GITHUB_STEP_SUMMARY"
unset GITHUB_STEP_SUMMARY
```

Expected: the report is printed, `CRAPload: N/0 (FAIL)` appears on
stderr, exit is non-zero, and the step summary contains the report.

## 5. Grades in JSON match the markdown (FR-010, SC-005)

```bash
gaze report --format=json ./internal/crap | jq '.assessment.scorecard[] | {label, grade, marker}'
gaze report --format=markdown ./internal/crap | sed -n '/Summary Scorecard/,/^$/p'
```

Expected: each label shows the same `marker grade` pair in both
outputs.

## 6. Local reporter prompt is ignored (US5)

```bash
ls .opencode/agents/gaze-reporter.md       # exists in this repo
gaze report --ai=opencode --model=none/none ./internal/crap 2>&1 >/dev/null | grep '^notice: ignoring .opencode/agents/gaze-reporter.md'
gaze report --format=markdown ./internal/crap 2>&1 >/dev/null | grep -c 'notice: ignoring'   # 0
```

Expected: exactly one notice line in text mode, none in markdown
mode. (The first command may then fail on the bogus model; that is
fine, since only the notice matters here.)

## 7. Real AI path, optional (US1)

```bash
OPENCODE_CONFIG_CONTENT="$(cat ~/my-opencode-config.json)" \
  gaze report --ai=opencode ./internal/crap; echo "exit=$?"
```

Expected, one of:

- exit 0 and a report containing both `<!-- gaze:ai-summary -->` and
  `<!-- gaze:ai-recommendations -->` blocks, or
- exit ≠ 0 with `AI answer rejected (rule …): …` on stderr, and the
  deterministic report plus `> ⚠️ AI narrative rejected: …` on stdout.

Narration never appears in the output.

## 8. Interactive `/gaze` (US4, SC-007)

In OpenCode on this repository, run `/gaze`, `/gaze crap ./internal/crap`,
and `/gaze quality ./internal/quality`. Expected: each shows the
Gaze-rendered report verbatim, followed by an `## Assessment` section.
Record the agent prompt deltas:

```bash
wc -l -w -c .opencode/agents/gaze-reporter.md
python3 -c "print(len(open('.opencode/agents/gaze-reporter.md',encoding='utf-8').read())//4)"
```

Expected: 160 lines or fewer and about 1,600 tokens or fewer, compared
with 528 lines and about 5,118 tokens before.

## 9. Scaffold propagation (FR-034)

```bash
d=$(mktemp -d); cd "$d" && go mod init example.com/x >/dev/null
gaze init && echo "# custom" >> .opencode/agents/gaze-reporter.md
gaze init | grep 'updated: .opencode/agents/gaze-reporter.md'
```

Expected: the customized agent file is reported as `updated` and its
content matches the embedded copy again.

## 10. Post-release check (SC-008, research R13)

After a release containing this feature, and after the follow-up that
bumps the reusable workflow's `gaze-version`, confirm the dogfood
`Gaze AI Report` job either publishes the full deterministic report
with validated AI slots or fails. It must never pass with narration.
