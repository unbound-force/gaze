---
tag: ci-parity
author: jay-flowers
category: gotcha
created_at: 2026-11-25T03:27:59Z
identity: ci-parity-20261125T032759-jay-flowers
tier: draft
as_of: "gaze@73bb94085f82edd0182836a17ed2e0669856cc02"
evidence:
  - "session:7e4660d1-156f-482e-84ff-dfce158e8a5b"
  - ".gitignore:67"
  - ".mega-linter.yml"
---

During the /uf.unleash Step 6 pre-flight soft gate on opsx/report-language-dispatch (2026-11-25), a working-tree `yamllint .` run exited 1 with nine errors — every one traced to .uf/review-matrix.override.yaml, a gitignored (.gitignore:67), untracked local tooling file that CI's checkout never contains; the three remaining findings were pre-existing warnings on unchanged workflow files that green main already carries. The causality classification that worked: verify the failing file's git-tracking status first (git check-ignore -v, git ls-files — empty output means untracked), lint the branch's in-diff files directly, and re-run the tool over the tracked-file CI surface (git ls-files '*.yml' '*.yaml' | xargs yamllint); that run exited 0, proving CI parity without building a baseline worktree. A mechanical branch-vs-baseline worktree comparison would have misclassified the failure as branch-caused (the untracked file exists only in the main checkout, so a fresh baseline worktree passes), sending an approving branch into a pointless fix loop. The general rule: local lint runs walk the working directory, CI lints the tracked tree — when they disagree, the tracked-file set is the CI-parity surface, and git-tracking evidence beats a worktree diff for classifying out-of-scope failures.
