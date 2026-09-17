---
tag: external-analyzer-protocol
author: jay-flowers
category: pattern
created_at: 2026-09-01T21:31:12Z
identity: external-analyzer-protocol-20260901T213112-jay-flowers
tier: draft
---

When wiring a new JSON-RPC protocol method into the ExternalSideEffectAnalyzer adapter pattern, the established approach is: (1) create a new file in internal/adapter/ following the file-per-concern pattern (e.g., classify.go for classify_signals), (2) implement a fetch function that gracefully degrades by returning nil on all error types (transport, protocol, unmarshal) with stderr warnings, (3) implement a pure data transformation merge function that keeps the logic testable without I/O, (4) wire the composed method into the existing loadBatch() and loadStreaming() flow via a single classifyAndMerge() call. The classify_signals method established that single-return-value signatures (returning nil instead of error) are preferable when the caller treats all errors identically as "continue without data" — this simplifies the call site and follows design decision D3 (graceful degradation).
