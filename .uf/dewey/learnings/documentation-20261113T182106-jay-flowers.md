---
tag: documentation
author: jay-flowers
category: gotcha
created_at: 2026-11-13T18:21:06Z
identity: documentation-20261113T182106-jay-flowers
tier: draft
---

A documentation note that declares a minimum version for a behavior creates a sweep obligation over every example in the same document. In gaze's docs/guides/ci-integration.md the note was rewritten to say pins below @v2.4.0 install a binary whose adapter output is not validated, while the canonical consumer example 190 lines later still pinned @v2.3.0 and omitted gaze-version — teaching the unvalidated path even though the prose warned against it. When a version-availability claim changes, grep the document for older pins and update the copy-paste examples first, because examples are copied and prose is skimmed.
