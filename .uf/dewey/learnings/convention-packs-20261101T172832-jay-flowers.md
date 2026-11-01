---
tag: convention-packs
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:28:32Z
identity: convention-packs-20261101T172832-jay-flowers
tier: draft
---

Unbound Force convention packs pair each canonical scaffolded pack (.opencode/uf/packs/go.md, TC-NNN rule IDs, "scaffolded by uf" marker, Custom Rules section intentionally empty) with a project-specific *-custom.md sibling (go-custom.md, CR-NNN rule IDs) that is the designated extension point — the canonical pack's own footer ("Project-specific custom rules belong in go-custom.md") and the unbound-force website developer docs ("Adding Custom Rules": edit the appropriate *-custom.md, use CR-NNN, tag severity) both document this mechanism. Proposals that add rules by editing the canonical pack directly (gaze issue #266 proposed TC-013 in go.md) conflict with the structure: local canonical edits risk drift from the scaffold source and rule-ID collisions if upstream uf later claims the same number. Correct routing: project-specific rules land as CR-NNN in the *-custom.md sibling (loaded alongside the canonical pack by Cobalt-Crush and all Divisor personas, so the guidance still reaches agents); universal rules are proposed upstream to the canonical pack owner. Verified in gaze main 2026-11-01: go.md defines TC-001–TC-012 with the empty-canonical-section footer; go-custom.md's Custom Rules section is empty.
