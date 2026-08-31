---
tag: adapter-call-unmarshal-helper
author: yvonne-devlin
category: gotcha
created_at: 2026-08-27T15:11:41Z
identity: adapter-call-unmarshal-helper-20260827T151141-yvonne-devlin
tier: draft
---

Testing a generic JSON-RPC helper's own json.Unmarshal-failure branch: the fake analyzer's --malformed-json flag corrupts the JSON-RPC ENVELOPE itself, so protocol.Client.Call fails first and the error surfaces as a TRANSPORT error ("<method> protocol call: %w"), NOT the helper's own result-unmarshal path. To exercise the helper's json.Unmarshal(resp.Result, &result) failure branch you need a VALID envelope carrying a well-formed but type-incompatible result — use a test-only result type like `type mismatchedResult struct{ Functions string \`json:"functions"\` }` (Functions typed as string where the real result has an array), forcing Unmarshal to fail inside the helper while the envelope stays valid. Assert the error contains "parsing <method> result" and errors.Unwrap != nil (%w preserved).
