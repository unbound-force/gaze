---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:28:39Z
identity: issue-triage-20261101T172839-jay-flowers
tier: draft
---

Convention-pack rule proposals (retro-generated issues requesting a new TC-/CS-/SC- rule) are mechanically verifiable at triage time: (1) grep the target pack section to confirm the proposed rule ID is the next free number; (2) grep the codebase to confirm the claimed usage sites of both the legacy and preferred patterns; (3) check proposed placement against the pack architecture — canonical packs (e.g., go.md) are scaffolded assets whose Custom Rules sections state that project-specific rules belong in the matching *-custom.md pack with CR-NNN IDs (also documented in the unbound-force/website developer docs), so appending a new numbered rule to the canonical file (e.g., TC-013 into go.md) is a placement decision the spec phase must make explicitly: either a local CR-NNN rule in go-custom.md (safe against scaffold updates) or an upstream change to the canonical pack source in the unbound-force repo. A proposal that cites a concrete rework cost, an in-repo precedent for the preferred pattern, and a measurable validation criterion is a strong VALID candidate even at self-assessed moderate confidence.
