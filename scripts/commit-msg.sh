#!/usr/bin/env sh
set -eu

msg_file=${1:?commit message file required}
header=$(sed -n '1p' "$msg_file" | tr -d '\r')
# Anything after the header must be blank, a comment, or a DCO sign-off trailer.
extra=$(sed '1d' "$msg_file" | grep -Ev '^(#|Signed-off-by: |[[:space:]]*$)' || true)

if [ -z "$header" ]; then
  echo "commit message header is required" >&2
  exit 1
fi

case "$header" in
  Merge\ *|Revert\ *)
    exit 0
    ;;
esac

if [ -n "$extra" ]; then
  echo "commit message must be a single line (Signed-off-by trailers allowed)" >&2
  exit 1
fi

if [ "${#header}" -gt 100 ]; then
  echo "commit message header must be 100 characters or fewer" >&2
  exit 1
fi

pattern='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._/-]+\))?!?: [^ ].+$'

if ! printf '%s\n' "$header" | grep -Eq "$pattern"; then
  echo "commit message must match Conventional Commits: type(scope): subject" >&2
  exit 1
fi
