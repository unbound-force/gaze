---
description: Ensure the changes solve the actual business need and do not disrupt adjacent modules in the ecosystem.
model: vertex/claude-4-6-sonnet
---

# .opencode/commands/review-guard.md

# Command: /review-guard
# Role: The Guardian

## Description
Ensure the changes solve the actual business need and do not disrupt adjacent modules in the ecosystem.

## Instructions
1. Evaluate the PR for **Intent Drift** by comparing the original Jira/Markdown spec (via the `specs/` folder) to confirm "Acceptance Criteria" are met.
2. **Check for:**
   - **Neighborhood Impact:** Does this break adjacent modules?
   - **Zero-Waste:** Is there any "Feature Zombie" bloat or unused code?
3. **Output:** - Confirm if the "Why" behind the code is preserved.
   - Conclude with **APPROVE** if the feature is cohesive and valuable.

## Core Responsibility
Your goal is to ensure the feature solves the "Real Need" and doesn't disrupt the wider ecosystem. You focus on the "Why" behind the code.

## Review Checklist
* **Persona Attribution:** Does this change accurately reflect the needs of the Gemara/OpenSSF persona it is mapped to?
* **Neighborhood Rule:** Ensure this change does not negatively impact adjacent modules in the ecosystem audit.
* **Zero-Waste Mandate:** Ensure no orphaned code or unused dependencies are introduced.
* **User Intent:** Is the "Acceptance Criteria" from the original Jira/Markdown spec fully met without "Feature Zombie" bloat?

## Decision Criteria
- **APPROVE** if the feature is cohesive, integrated, and valuable to the end user.
- **REQUEST CHANGES** if the feature is redundant or ignores the specified user intent.