---
tag: markdown-escaping
author: jay-flowers
category: gotcha
created_at: 2026-11-15T01:12:10Z
identity: markdown-escaping-20261115T011210-jay-flowers
tier: draft
---

A backtick-toggle scanner over rendered markdown misaligns inside double-backtick code spans whose content embeds backticks (e.g. the ``parse`raw`Literal`` cell emitted from gaze's golden fixture payload-full.json): the toggle treats the second delimiter backtick as a span close and the embedded backticks as new delimiters, so scanned span boundaries are not the true code-span boundaries. Soundness then rests on parity (even embedded-backtick counts resynchronize the toggle after the cell) and on the target byte being absent from fixture data — premises that must be asserted in-test (e.g. grep the fixture bytes for the target character, assert even backtick counts), not stated in comments, because golden fixtures are routinely regenerated and edited.
