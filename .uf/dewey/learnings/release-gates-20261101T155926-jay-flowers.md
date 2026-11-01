---
tag: release-gates
author: jay-flowers
category: gotcha
created_at: 2026-11-01T15:59:26Z
identity: release-gates-20261101T155926-jay-flowers
tier: draft
---

In bash release-gate scripts, (( )) arithmetic evaluates at 64-bit signed precision: a version grammar like ^v[0-9]+\.[0-9]+\.[0-9]+$ admits components that wrap (or raise an expression error under set -e) at 19+ digits (>= 2^63), so the comparison is not total over the grammar the regex admits. When a gate compares tags with bash arithmetic, bound component length in the grammar itself (e.g. (0|[1-9][0-9]{0,8})) so the comparison is total by construction, or document the downstream containment. Observed in gaze's validate-workflow-floor.sh: both overflow outcomes happen to fail closed (non-zero exits), but only by accident of the failure modes — the at-floor/above-floor branches would still pass a wrapped tag if the floor tag checks succeed.
