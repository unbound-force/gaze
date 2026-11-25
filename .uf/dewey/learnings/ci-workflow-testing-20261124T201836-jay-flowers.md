---
tag: ci-workflow-testing
author: jay-flowers
category: pattern
created_at: 2026-11-24T20:18:36Z
identity: ci-workflow-testing-20261124T201836-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

Contract tests for GitHub Actions workflows (cmd/gaze/*_workflow_test.go) tend to pin `run:` scripts with substring checks. That catches drift between copies but not logic mutations applied to both copies at once, such as a widened path prefix, a dropped empty-diff guard, or swapped outputs. When a run script is a deterministic classifier or gate, test it by running it: extract the script from the parsed YAML, replace the `${{ ... }}` expressions, and run it with bash in a t.TempDir() git repo for each spec scenario. Skip the test when bash, git or jq is missing. .github/scripts/test-validate-workflow-floor.sh already uses the synthetic-repo approach.
