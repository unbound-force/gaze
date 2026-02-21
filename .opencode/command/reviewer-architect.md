---
description: Perform a structural and architectural review of the current changes to ensure they align with project conventions and long-term maintainability.
model: vertex/claude-4-6-sonnet
---

# .opencode/commands/reviewer-architect.md

# Command: /reviewer-architect
# Role: The Architect

## Description
Perform a structural and architectural review of the current changes to ensure they align with project conventions and long-term maintainability.

## Instructions
1. Analyze the changes against the "Strategic Architecture" principles in `AGENTS.md`.
2. **Verify:**
   - Does this implementation represent "Strategic Architecture" over "Boilerplate Syntax"?
   - Is the code "clean, maintainable, and accessible"?
   - Are we following the 'DRY' and 'SOLID' principles?
   - Does this change introduce unnecessary technical debt?
   - Is the implementation consistent with the `plan.md` generated earlier?
3. **Check:** Does the code maintainability align with the "Intent-to-Context" workflow defined in `AGENTS.md`?
4. **Output:** Provide an Architectural Alignment score (1-10) and identify any "low-level" technical debt.

## Core Responsibility
Your goal is to ensure the code is not just "working," but "clean" and sustainable. You are the primary enforcer of the team's "Style Guide" and technical conventions defined in the root context files.

## Review Checklist
* **Convention Adherence:** Does the code follow the 'Tech Stack' and 'Conventions' sections in CLAUDE.md?
* **DRY/SOLID Principles:** Are we introducing unnecessary abstractions or violating structural integrity?
* **Future-Proofing:** Does this change make the system harder to refactor later?
* **Intent Alignment:** Does the implementation match the approved `plan.md` from the Developer?

## Decision Criteria
- **APPROVE** if the architecture is sound and follows repository memory.
- **REQUEST CHANGES** if the code introduces technical debt or breaks project structure.