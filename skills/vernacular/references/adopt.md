# Adopting Vernacular

Two flows: a new project, and an existing project that already has private material in
its tree. Both end with the same checks.

## When to use

- The human asks to set up, adopt, or start using Vernacular in a repository.
- A new project is being bootstrapped (`git init`, rack-mount-go, Helicopter, or similar)
  and its plans, reports, and research should stay out of the first commit.
- An existing repository holds working material in its tree: self-ignored directories
  (a `.gitignore` containing `*`), ignored scratch directories, untracked plans or reports.

## When not to use

- The repository is already attached and the human did not ask for an inventory: use the
  main skill instead.
- Governed material: secrets, credentials, customer or regulated data. It belongs in a
  proper system, not apparatus. Say so and leave it in place.

## What this never does

- No commits, no staging, no deletion.
- No edits to tracked files, and no moving tracked files, without the human's approval.
- No classifying or moving anything the human has not decided on.
- No changes to the repository beyond `vern init`'s one line of local Git config.

## From scratch

1. `git init` if there is no repository yet.
2. `vern init` before any bootstrap tool or agent generates material, so the first commit
   never sees it. `vern init` works before the first commit.
3. Run the bootstrap. Plans, research, and reports it produces go to `vern path working`
   (or `vern path apparatus` if worth keeping), not the tree.
4. Ask whether to add the optional snippet that `vern init` printed to `AGENTS.md`. Default
   no: it is a committed trace, and the skill already does its job on this machine.
5. Before the first commit, `git status --short` must list only project files.
6. Run the checks below.

## Existing project

### 1. Inventory (read-only)

From the repository root:

    git status --short --ignored        # ?? untracked, !! ignored
    git ls-files '*.md' | grep -v /     # tracked root-level Markdown

For each untracked or ignored entry, and each tracked root-level file that is not a standard
project file (README, LICENSE, CHANGELOG, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT,
MAINTAINERS, AGENTS, CLAUDE), collect:

- **Tracked or not**: the `git status` marker, or `git ls-files --error-unmatch PATH`.
- **Ignore rule**: `git check-ignore -v PATH` names the file and line of the rule. For a
  directory, ask about a file inside it: `git check-ignore -v "$(find PATH -type f | head -1)"`.
  A directory that ignores itself (its own `.gitignore` holds `*`) only shows up this way.
- **Size**: `du -sh PATH`.
- **Last modified**: `find PATH -type f -exec ls -lt {} + | head -1`.
- **Referenced by**: `git grep -n -F 'PATH'`, the tracked files that break if it moves.

Skip descending into dependency and build directories (`node_modules`, `dist`, `bin`,
`target`, `coverage`, `.venv`); report their size only.

Show one table: path, tracked, ignore rule, size, last modified, referenced by, and a
proposal: **leave**, **apparatus**, **working**, or **canon**. Propose **leave** for
dependencies, build output, caches, and tool or harness state (`.omp/`, `.claude/`,
`.wrangler/`, …). The proposal is a suggestion; the human decides every row.

### 2. Decide

Ask the human to assign each row. Point out rows that tracked files reference: moving them
breaks those references until the human fixes them. Proceed only with rows the human
assigned; everything else stays where it is.

### 3. Attach

If `git config --get vernacular.workspace` is empty, `vern init`. If `vern attach` lists a
workspace last seen with another checkout of this repository, offer `vern attach ID`
instead.

### 4. Move

- **apparatus / working**: `mv PATH "$(vern path apparatus)/"` (or `working`). Keep the
  name. If the destination exists, stop and ask; never overwrite.
- **Tracked file to apparatus or working**: only with explicit approval for that file. Same
  `mv`; the deletion then shows in `git status` for the human to commit.
- **canon**: untracked material that belongs in the project stays where it is for the human
  to stage. Material already in the workspace enters canon only through
  `vern promote SRC DEST`.
- Leave references in tracked files alone; list them for the human.

### 5. Record

Write `$(vern path apparatus)/adoption-YYYY-MM-DD.md`: each row with its decision, where it
went, and the tracked references left to fix. This is how the workspace stays legible about
what came from the tree.

### 6. Verify the tree

`git status --short --ignored` again. Moved rows are gone; nothing else changed except
approved deletions of tracked files.

## Checks

Run from inside the attached checkout:

    sh "<this skill's directory>/scripts/check.sh"

It reads the checkout and runs every destructive step (`git clean -xfd`, moving the
repository, a worktree, a lost `~/.vern`) on a throwaway clone, then restores the workspace's
last-seen path. Report its output verbatim. Exit status is the number of failures; on any
failure, stop and report rather than working around it. Friction that needs a change to
`vern` is a ticket, not a workaround.
