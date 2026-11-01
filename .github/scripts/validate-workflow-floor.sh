#!/usr/bin/env bash
#
# Release-time consistency check between the tag being released and the
# workflow floor embedded in `gaze init --ci`.
#
# `gaze init --ci` pins the scaffolded CI workflow to the gaze release tag
# and rejects tags below the minWorkflowTag floor declared in
# internal/scaffold/scaffold.go. A release below that floor would ship a
# binary that fails its own --ci validation; a release above it presumes the
# floor tag exists and contains the reusable workflow. This script fails the
# release workflow when either invariant breaks.
#
# Usage: validate-workflow-floor.sh -- <release-tag> [repo-root]
#   <release-tag>  the tag being released (strict vX.Y.Z[-prerelease] form)
#   [repo-root]    repository root to validate against (default: .)
#
# Exits:
#   0  tag is consistent with the floor, or the tree predates floor governance
#   1  validation failure (tag below floor; floor tag missing or incomplete)
#   2  usage error, malformed release tag, or malformed repository state
#      (duplicate, malformed, or pre-release floor constant; unreadable
#      scaffold.go; floor-bearing tree missing the reusable workflow)

set -euo pipefail
export LC_ALL=C

if [[ "${1:-}" != "--" || $# -lt 2 || $# -gt 3 ]]; then
  printf 'usage: %s -- <release-tag> [repo-root]\n' "$0" >&2
  exit 2
fi

release_tag="$2"
repo_root="${3:-.}"
scaffold_go="$repo_root/internal/scaffold/scaffold.go"
workflow_path=".github/workflows/reusable_gaze_ai_report.yml"

# parse_release <tag> sets parsed_major, parsed_minor, parsed_patch, and
# parsed_pre globals; returns 1 for any tag that is not strict canonical
# semver (no leading zeros, pre-release suffix of non-empty dot-separated
# identifiers, mirroring x/mod/semver's IsValid rejection of malformed
# pre-release identifiers such as empty dot-separated segments).
parse_release() {
  if [[ ! "$1" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
    return 1
  fi
  parsed_major="${BASH_REMATCH[1]}"
  parsed_minor="${BASH_REMATCH[2]}"
  parsed_patch="${BASH_REMATCH[3]}"
  parsed_pre="${BASH_REMATCH[4]-}"
}

# compare_number <a> <b> prints -1, 0, or 1 for the numeric ordering of a
# relative to b.
compare_number() {
  if (( $1 < $2 )); then
    printf '%s\n' -1
  elif (( $1 > $2 )); then
    printf '%s\n' 1
  else
    printf '%s\n' 0
  fi
}

# compare_release <tag-a> <tag-b> prints -1, 0, or 1: the semver precedence
# of tag-a relative to tag-b. A pre-release of a version sorts below the
# canonical release; two pre-releases of the same version compare lexically,
# which is correct here because the floor is rejected below unless canonical,
# so at most one operand (the release tag) can carry a pre-release suffix.
compare_release() {
  parse_release "$1"
  local a_major="$parsed_major" a_minor="$parsed_minor" a_patch="$parsed_patch" a_pre="$parsed_pre"
  parse_release "$2"
  local b_major="$parsed_major" b_minor="$parsed_minor" b_patch="$parsed_patch" b_pre="$parsed_pre"
  local result
  result="$(compare_number "$a_major" "$b_major")"
  if [[ "$result" != 0 ]]; then
    printf '%s\n' "$result"
    return 0
  fi
  result="$(compare_number "$a_minor" "$b_minor")"
  if [[ "$result" != 0 ]]; then
    printf '%s\n' "$result"
    return 0
  fi
  result="$(compare_number "$a_patch" "$b_patch")"
  if [[ "$result" != 0 ]]; then
    printf '%s\n' "$result"
    return 0
  fi
  if [[ -n "$a_pre" || -n "$b_pre" ]]; then
    if [[ "$a_pre" == "$b_pre" ]]; then
      printf '%s\n' 0
    elif [[ -z "$a_pre" ]]; then
      printf '%s\n' 1
    elif [[ -z "$b_pre" ]]; then
      printf '%s\n' -1
    elif [[ "$a_pre" < "$b_pre" ]]; then
      printf '%s\n' -1
    else
      printf '%s\n' 1
    fi
    return 0
  fi
  printf '%s\n' 0
}

if ! parse_release "$release_tag"; then
  printf 'invalid release tag: %s\n' "$release_tag" >&2
  exit 2
fi

# Extract the floor from the source of truth. The value is never duplicated
# here: this script greps internal/scaffold/scaffold.go so the gate and the
# scaffold cannot drift apart.
floor_lines=()
if [[ -f "$scaffold_go" ]]; then
  matches_file="$(mktemp)"
  trap 'rm -f "$matches_file"' EXIT
  grep_status=0
  grep -E '^[[:space:]]*const minWorkflowTag = "[^"]*"' "$scaffold_go" >"$matches_file" || grep_status=$?
  if (( grep_status > 1 )); then
    printf '::error::could not read %s to extract the workflow floor (grep status %d)\n' "$scaffold_go" "$grep_status"
    exit 2
  fi
  mapfile -t floor_lines <"$matches_file"
fi
if [[ ${#floor_lines[@]} -eq 0 ]]; then
  printf '::notice::no minWorkflowTag constant in internal/scaffold/scaffold.go; tree predates workflow-floor governance, skipping\n'
  exit 0
fi
if [[ ${#floor_lines[@]} -gt 1 ]]; then
  printf '::error::expected exactly one minWorkflowTag declaration in internal/scaffold/scaffold.go, found %d\n' "${#floor_lines[@]}"
  exit 2
fi
floor="$(printf '%s\n' "${floor_lines[0]}" | sed -E 's/^[[:space:]]*const minWorkflowTag = "([^"]*)".*$/\1/')"
if ! parse_release "$floor"; then
  printf '::error::malformed minWorkflowTag value in internal/scaffold/scaffold.go: %s\n' "$floor"
  exit 2
fi
if [[ -n "$parsed_pre" ]]; then
  printf '::error::workflow floor must be a canonical release tag, found pre-release floor: %s\n' "$floor"
  exit 2
fi

# A floor-bearing tree must ship the workflow the floor points at.
if [[ ! -f "$repo_root/$workflow_path" ]]; then
  printf '::error::floor-bearing tree must contain %s\n' "$workflow_path"
  exit 2
fi

precedence="$(compare_release "$release_tag" "$floor")"
if [[ "$precedence" == "-1" ]]; then
  printf '::error::release tag %s is below the workflow floor %s: the released binary would fail its own --ci validation\n' "$release_tag" "$floor"
  exit 1
fi
if [[ "$precedence" == "0" ]]; then
  printf 'release tag %s ships the reusable AI-report workflow at the workflow floor\n' "$release_tag"
  exit 0
fi

# Above the floor: the floor tag must exist and contain the workflow, proving
# the scaffold's pin has something real to point at.
if ! git -C "$repo_root" rev-parse -q --verify "refs/tags/$floor" >/dev/null; then
  printf '::error::release tag %s exceeds the workflow floor %s, but tag %s does not exist in the repository\n' "$release_tag" "$floor" "$floor"
  exit 1
fi
if ! git -C "$repo_root" show "$floor:$workflow_path" >/dev/null 2>&1; then
  printf '::error::release tag %s exceeds the workflow floor %s, but tag %s does not contain %s\n' "$release_tag" "$floor" "$floor" "$workflow_path"
  exit 1
fi
printf 'release tag %s is above workflow floor %s; floor tag exists and contains %s\n' "$release_tag" "$floor" "$workflow_path"
