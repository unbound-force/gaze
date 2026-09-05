---
tag: external-analyzer-protocol
author: jay-flowers
category: pattern
created_at: 2026-09-01T21:31:16Z
identity: external-analyzer-protocol-20260901T213116-jay-flowers
tier: draft
---

When adding a new capability to the external analyzer protocol adapter, the Session struct's Providers field serves as the integration point for capability gating. The pattern is: (1) add a new provider field to the adapter.Providers struct (e.g., SideEffects), (2) gate it on the analyzer's capability response (e.g., classify_signals: true), (3) thread the config through NewSession → NewExternalSideEffectAnalyzer so the adapter has access to classification thresholds. The fake_analyzer test binary in internal/protocol/testdata/fake_analyzer/main.go must be updated to advertise the new capability and return realistic test data for integration tests. When removing a CLI guard that previously blocked a feature (like --analyzer on gaze quality), ensure both the rejection test is replaced with an acceptance test AND the binary-not-found error path is covered.
