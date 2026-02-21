# .opencode/agent/review-council.md

# Command: /review-council

## Description
Review the current codebase for compliance with the "Behavioral Constraints" in `AGENTS.md` using the review council.

## Instructions
1. Deligate the review to the review council, i.e. use the commands `/reviewer-adversary`, `/reviewer-architect` and `/reviewer-guard` commands collecting all **REQUEST CHANGES**.
2. Provide the feedback to agents to address the **REQUEST CHANGES** and repeat the process until all **REQUEST CHANGES** are addressed. If the process has exceeded 3 iterations, request the user to give direction to continue the process or to stop the process.
3. Provide a report to the user on what was found and what was fixed. If the process was stopped then additionally report on the currect set of outstanding **REQUEST CHANGES**. If there were persistent circular **REQUEST CHANGES** then report on those with additional detail so that the user is armed with pertenant information to make a decision.