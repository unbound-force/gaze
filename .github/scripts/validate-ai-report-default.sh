#!/usr/bin/env bash
#
# Requires bash 4+ (mapfile). On macOS, /bin/bash is 3.2; use Homebrew bash.
#
# Release-time check that the reusable AI-report workflow's `gaze-version`
# input default equals the tag being released.
#
# Callers reference the reusable workflow by tag, and GitHub resolves input
# defaults from the workflow file at that tag. A caller pinned to @vX.Y.Z
# that omits `gaze-version` therefore installs whatever release the tagged
# file names. If the default is stale, that caller silently reports with an
# older binary. Gaze's own dogfood job sets `build-from-source: true`, so CI
# never exercises the default; this gate is the only check.
#
# The rule is exact equality, pre-releases included: a v2.5.0-rc.1 tag must
# ship a v2.5.0-rc.1 default, because a later final tag cannot change what
# the pre-release tag already contains.
#
# Usage: validate-ai-report-default.sh -- <release-tag> [workflow-file]
#   <release-tag>    the tag being released (strict vX.Y.Z[-prerelease] form)
#   [workflow-file]  default: .github/workflows/reusable_gaze_ai_report.yml
#
# Exits:
#   0  default equals the release tag
#   1  default differs from the release tag
#   2  usage error, malformed release tag, missing workflow file, or not
#      exactly one parsable gaze-version default

set -euo pipefail
export LC_ALL=C

if [[ "${1:-}" != "--" || $# -lt 2 || $# -gt 3 ]]; then
  printf 'usage: %s -- <release-tag> [workflow-file]\n' "$0" >&2
  exit 2
fi

release_tag="$2"
workflow_file="${3:-.github/workflows/reusable_gaze_ai_report.yml}"

if [[ ! "$release_tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
  printf 'invalid release tag: %s\n' "$release_tag" >&2
  exit 2
fi

if [[ ! -f "$workflow_file" ]]; then
  printf '::error::workflow file not found: %s\n' "$workflow_file"
  exit 2
fi

# Print the `default:` value of every `gaze-version:` mapping key, one per
# line. A block ends at the first non-blank, non-comment line indented no
# deeper than its key. Surrounding single or double quotes are stripped.
defaults=()
mapfile -t defaults < <(awk '
  BEGIN { squote = sprintf("%c", 39) }
  function indent(s) { match(s, /^ */); return RLENGTH }
  /^[[:space:]]*gaze-version:[[:space:]]*(#.*)?$/ { in_block = 1; key_indent = indent($0); next }
  in_block && /^[[:space:]]*(#.*)?$/ { next }
  in_block && indent($0) <= key_indent { in_block = 0 }
  in_block && /^[[:space:]]*default:/ {
    value = $0
    sub(/^[[:space:]]*default:[[:space:]]*/, "", value)
    sub(/[[:space:]]+#.*$/, "", value)
    sub(/[[:space:]]+$/, "", value)
    first = substr(value, 1, 1); last = substr(value, length(value), 1)
    if (length(value) >= 2 && first == last && (first == "\"" || first == squote)) value = substr(value, 2, length(value) - 2)
    print value
  }
' "$workflow_file")

if [[ ${#defaults[@]} -ne 1 ]]; then
  printf '::error::expected exactly one gaze-version default in %s, found %d\n' "$workflow_file" "${#defaults[@]}"
  exit 2
fi

default_value="${defaults[0]}"
if [[ "$default_value" != "$release_tag" ]]; then
  printf '::error::%s gaze-version default is %s but the release tag is %s: callers pinned to @%s that omit gaze-version would install %s. Set the default to %s before releasing.\n' \
    "$workflow_file" "$default_value" "$release_tag" "$release_tag" "$default_value" "$release_tag"
  exit 1
fi

printf 'gaze-version default %s matches release tag %s\n' "$default_value" "$release_tag"
