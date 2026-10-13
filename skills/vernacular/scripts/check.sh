#!/bin/sh
# Checks for a repository attached to a Vernacular workspace. Run from inside the checkout.
#   check.sh             legibility only: reads, takes about a second
#   check.sh --durable   also proves the attachment survives a reclone, git clean -xfd, a moved
#                        repository, a worktree, and a lost ~/.vern, on a throwaway clone (copies
#                        the repository; the checkout itself is only read)
# Exit status is the number of failed checks.
# ponytail: POSIX sh for macOS and Linux; Windows needs WSL or Git Bash.

set -u
fails=0
ok() { printf 'ok    %s\n' "$1"; }
fail() { printf 'FAIL  %s\n' "$1"; fails=$((fails + 1)); }
skip() { printf 'skip  %s\n' "$1"; }
# expect LABEL WANT DIR [ENV...]: `vern path apparatus` in DIR prints WANT.
expect() {
  label=$1 want=$2 dir=$3
  shift 3
  got=$(cd "$dir" && env "$@" vern path apparatus 2>/dev/null) || got=
  if [ "$got" = "$want" ]; then ok "$label"; else fail "$label (got '${got:-error}', want '$want')"; fi
}
# seen: `vern attach` lists the workspace as last seen with this checkout (paths under $HOME print as ~).
seen() {
  want=$canon
  case $canon in "$HOME"/*) want="~${canon#"$HOME"}" ;; esac
  vern attach | awk -v id="$id" -v want="last seen with $want" \
    'index($0, id "  ") == 1 && substr($0, length($0) - length(want) + 1) == want { f = 1 } END { exit !f }'
}

command -v vern >/dev/null || { echo "vern is not on PATH" >&2; exit 1; }
canon=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "not inside a Git repository" >&2; exit 1; }
canon=$(cd "$canon" && pwd -P)
id=$(git -C "$canon" config --get vernacular.workspace) || { echo "not attached: run vern init or vern attach" >&2; exit 1; }
apparatus=$(cd "$canon" && vern path apparatus) || exit 1
root=$(cat "$HOME/.vern/workspaces/$id")

echo "Legible"
if (cd "$canon" && vern status) | grep -q "$id"; then ok "vern status names $id"; else fail "vern status names $id"; fi
if [ "$(cd "$canon" && vern path canon)" = "$canon" ]; then ok "vern path canon is this checkout"; else fail "vern path canon is this checkout"; fi
if seen; then ok "vern attach lists $id with this checkout"; else fail "vern attach lists $id with this checkout"; fi
if grep -q "^id: $id\$" "$root/vernacular.yml"; then ok "manifest names its own id"; else fail "manifest names its own id"; fi

if [ "${1:-}" != --durable ]; then
  echo "$fails failed"
  exit "$fails"
fi

echo "Durable (on a throwaway clone)"
tmp=$(mktemp -d)
tmp=$(cd "$tmp" && pwd -P)
trap '(cd "$canon" && vern init >/dev/null); rm -rf "$tmp"' EXIT
git clone -q "$canon" "$tmp/clone" 2>/dev/null
if (cd "$tmp/clone" && vern path apparatus >/dev/null 2>&1); then fail "fresh clone starts unattached"; else ok "fresh clone starts unattached"; fi
(cd "$tmp/clone" && vern attach "$id" >/dev/null)
expect "reclone: vern attach $id" "$apparatus" "$tmp/clone"
git -C "$tmp/clone" clean -qxfd
expect "git clean -xfd" "$apparatus" "$tmp/clone"
mv "$tmp/clone" "$tmp/moved"
expect "repository moved" "$apparatus" "$tmp/moved"
if git -C "$tmp/moved" rev-parse -q --verify HEAD >/dev/null; then
  git -C "$tmp/moved" worktree add -q --detach "$tmp/worktree" 2>/dev/null
  expect "worktree" "$apparatus" "$tmp/worktree"
else
  skip "worktree (no commits yet)"
fi
mkdir "$tmp/home"
if (cd "$tmp/moved" && HOME="$tmp/home" vern path apparatus >/dev/null 2>&1); then fail "lost ~/.vern is detected"; else ok "lost ~/.vern is detected"; fi
(cd "$tmp/moved" && HOME="$tmp/home" vern attach "$root" >/dev/null)
expect "lost ~/.vern: vern attach DIR" "$apparatus" "$tmp/moved" HOME="$tmp/home"

# The clone's attach recorded it as canon's last-seen path; point it back at this checkout.
(cd "$canon" && vern init >/dev/null)
if seen; then ok "last-seen path restored"; else fail "last-seen path restored"; fi

echo "$fails failed"
exit "$fails"
