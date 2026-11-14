---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:39:24Z
identity: test-quality-20261102T213924-jay-flowers
tier: draft
---

When validating test-gap feedback, audit both coverage layers — fixture files and synthetic in-memory cases — before endorsing the stated gap. In gaze PR #314, the reviewer's claim "no fixture exercises \r\n endings" was literally true of the opencode_streams/ fixture files, yet TestExtractFinalText_Synthetic had already pinned CRLF framing for valid events via strings.ReplaceAll; the genuinely untested compositions were junk interleaved with valid events and junk lines carrying \r\n endings. Compose the uncovered dimensions in the fix fixture instead of re-testing a covered one, and include a JSON-but-wrong-shape line (e.g. [1,2], which fails json.Unmarshal into the event struct) alongside bare non-JSON prose, because "lines that are not JSON" in a doc comment usually means "lines that fail to unmarshal into the target type".
