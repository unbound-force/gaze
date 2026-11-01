---
tag: release-gates
author: jay-flowers
category: gotcha
created_at: 2026-11-01T16:43:50Z
identity: release-gates-20261101T164350-jay-flowers
tier: draft
---

Release/CI shell scripts using bash-4+ builtins (mapfile) run green on GitHub's ubuntu runners (bash 5.x) but fail under macOS local CI-parity runs, where /bin/bash is 3.2 and mapfile exits 127 with 'command not found' — a confusing failure that reads as a script bug rather than an environment gap. When a repo mandates local CI parity (running locally the same scripts CI runs, as gaze's AGENTS.md Technical Guardrails require), record the minimum bash version in the script header or contributing docs. Observed in gaze's validate-workflow-floor.sh and test-validate-workflow-floor.sh, which mirror validate-module-major.sh — all three share the undocumented bash >= 4 requirement; the fix is a one-line header note per script (macOS contributors should use Homebrew bash).
