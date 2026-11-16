# CI Workflow Requirements

## ADDED Requirements

### Requirement: Published releases MUST be verified installable from the public Go module proxy

The repository MUST run a post-release verification workflow
(`.github/workflows/post-release-install-check.yml`) that installs the exact
published release tag of gaze from the public Go module proxy and fails
unless the installed binary's build info reports module
`github.com/unbound-force/gaze/v2` at exactly that tag. The workflow MUST
run automatically when the `Release` workflow completes successfully, MUST
also be runnable on demand via `workflow_dispatch` with a tag input, and
MUST NOT run as part of pre-merge CI (no `push` or `pull_request`
triggers). The install MUST use temporary `GOBIN` and module-cache
directories so no pre-populated cache can satisfy the check. Network- and
API-dependent steps MUST carry bounded retries so transient proxy
propagation lag does not produce a false failure; the final attempt MUST
stay fatal. The workflow MUST NOT modify any release artifact and MUST NOT
require permissions beyond read-only `contents: read`.

#### Scenario: Successful release is verified automatically

- **GIVEN** the `Release` workflow for tag `vX.Y.Z` completes successfully
- **WHEN** the completion event reaches the post-release check
- **THEN** it runs
  `go install github.com/unbound-force/gaze/v2/cmd/gaze@vX.Y.Z` against
  proxy.golang.org, with temporary `GOBIN` and module-cache directories
- **AND** `go version -m` on the installed binary reports module
  `github.com/unbound-force/gaze/v2` at `vX.Y.Z`
- **AND** the binary executes (`gaze --version`) and the job succeeds

#### Scenario: Manual dispatch verifies an exact tag

- **GIVEN** a maintainer dispatches the workflow with tag input `vX.Y.Z`
  for an already-published release
- **WHEN** the run starts
- **THEN** it performs the same install-and-verify procedure for exactly
  `vX.Y.Z`
- **AND** the job outcome reflects whether that tag is installable from the
  public proxy at run time

#### Scenario: Proxy propagation lag is retried within bounds

- **GIVEN** tag `vX.Y.Z` was just published and proxy.golang.org has not
  propagated it yet
- **WHEN** a `go install` attempt fails
- **THEN** the workflow retries a bounded number of times with a delay
  between attempts
- **AND** if the final attempt still fails, the job fails with an
  `::error::` naming the tag and the attempt count

#### Scenario: Module or version mismatch fails the check

- **GIVEN** `go install` succeeds but the binary's build info reports a
  module other than `github.com/unbound-force/gaze/v2`, or a version other
  than the tag under verification
- **WHEN** the verification step evaluates the build info
- **THEN** the job fails with an `::error::` naming the expected tag and
  the installed module and version

#### Scenario: Failed release runs do not trigger verification

- **GIVEN** the `Release` workflow completes with a failure or cancellation
- **WHEN** its completion event reaches the post-release check
- **THEN** the check does not run, producing no verification signal for a
  release that was not published

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
