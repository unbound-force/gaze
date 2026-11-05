---
tag: test-infrastructure
author: jay-flowers
category: pattern
created_at: 2026-11-02T23:26:46Z
identity: test-infrastructure-20261102T232646-jay-flowers
tier: draft
---

Self-verifying test doubles: encode the expected invocation contract into the fake subprocess itself, so every integration test implicitly guards it. In gaze's spec-040 work, internal/aireport/testdata/fake_opencode/main.go verifies the adapter's full argv shape (run first, --agent gaze-headless exactly, --format json exactly) and the written agent file (frontmatter present, mode: primary, deny-all permission block, 0600 file mode) before producing any output, and exits with a distinct status (2) for invocation-contract violations vs simulated CLI failures (1). This converts otherwise-passing wiring regressions — wrong agent name, relaxed permissions, wrong format flag — into immediate failures in every test that touches the double, with zero per-test assertions. Pair it with: env-var capture modes (FAKE_OPENCODE_CAPTURE_ARGS / FAKE_OPENCODE_CAPTURE_AGENT) for tests that need explicit argv/agent-file assertions, and a default mode emitting a minimal valid event stream so ordinary callers exercise the extraction path end to end. Extends the hermetic-fake-binary pattern (test-infrastructure-20261102T194026): the double is not just hermetic, it is contract-enforcing.
