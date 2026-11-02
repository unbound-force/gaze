<!--
  Tasks for gaze-init-triage-advisor. Mark each task complete
  ([x]) as soon as it is finished. [P] marks tasks that can run
  in parallel with their siblings.
-->

## 1. Gaze repository config (Phase 1)

- [x] 1.1 Swap `divisor-testing` for `divisor-gaze-testing` in the
  `advisor.triage` list of `.uf/review-matrix.yaml`; keep the
  generic capabilities entry

## 2. gaze init registration (Phase 2)

- [x] 2.1 Add `triage` to the advisor-mode loop in
  `syncReviewMatrix` (`internal/scaffold/ufconfig.go`), order
  `code`, `spec`, `triage`, `feedback`; missing/invalid triage
  list skips the file with the existing warning
- [x] 2.2 Update the `ufGenericTestingAgent` comment and the
  `syncReviewMatrix` / `syncUFReviewConfig` GoDoc to describe the
  four-mode swap

## 3. Tests (Phase 3)

- [x] 3.1 Invert the triage assertions in
  `internal/scaffold/ufconfig_test.go` (triage contains the gaze
  persona and not the generic one)
- [x] 3.2 Add a case for a matrix without `advisor.triage`
  (file untouched, warning printed)
- [x] 3.3 Confirm the idempotency test covers the four-mode
  end-state

## 4. Documentation (Phase 4)

- [x] 4.1 [P] Update `docs/guides/opencode-integration.md` `.uf`
  registration section
- [x] 4.2 [P] Update `docs/reference/cli/init.md`
- [x] 4.3 [P] Add an AGENTS.md Recent Changes entry

## 5. Verification (Phase 5)

- [x] 5.1 CI parity: `go build ./...`;
  `go test -race -count=1 -short -timeout 15m ./...`;
  `golangci-lint run`
- [x] 5.2 Constitution alignment check (Accuracy, Minimal
  Assumptions, Actionable Output, Testability)
- [x] 5.3 Run `/uf.review-council` until APPROVE
- [x] 5.4 File a website docs issue in `unbound-force/website`
  (unbound-force/website#366; `gaze init` writes a different
  consumer matrix)
