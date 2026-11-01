---
tag: version-pinning
author: jay-flowers
category: gotcha
created_at: 2026-10-31T23:34:53Z
identity: version-pinning-20261031T233453-jay-flowers
tier: draft
---

golang.org/x/mod/semver.IsValid alone is insufficient validation when pinning canonical full-triple release tags: semver.IsValid("v2.3") is true AND semver.Compare("v2.3", "v2.3.0") == 0 (missing minor/patch components default to 0), so a partial form passes an IsValid+floor-comparison gate while naming a tag that does not exist — the generated reference then fails at CI run time with an opaque "workflow not found" error. When a tag format is canonical (always vMAJOR.MINOR.PATCH), layer a strict-form regex (e.g. ^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$) on top of IsValid+Compare, and keep IsValid in the chain to reject malformed pre-release identifiers the character class alone would admit (e.g. "v2.3.0-."). Verified in gaze init --ci's resolveGazeVersion (internal/scaffold/scaffold.go), where the "partial version passes bare semver but not strict form" test case pins the behavior; the same layering principle applies to any grammar-validated value substituted into generated YAML (restricted character classes also guarantee scalar safety — no quotes, colons, or CR can reach the double-quoted go-version scalar).
