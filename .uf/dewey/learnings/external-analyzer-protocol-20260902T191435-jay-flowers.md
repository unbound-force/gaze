---
tag: external-analyzer-protocol
author: jay-flowers
category: pattern
created_at: 2026-09-02T19:14:35Z
identity: external-analyzer-protocol-20260902T191435-jay-flowers
tier: draft
---

The comma-ok type assertion pattern (ecp, ok := providers.ContractCoverage.(*adapter.ExternalContractCoverageProvider)) is the correct way to access provider-specific methods from the crap.ContractCoverageProvider interface in the gaze codebase. The interface is intentionally narrow (universal scoring engine), and provider-specific metadata like detection confidence lives on the concrete type. The comma-ok pattern degrades gracefully to zero-value defaults when mock providers are used in tests. This was a consensus finding across all spec reviewers that the design must explicitly specify this pattern rather than assuming bare type assertions.
