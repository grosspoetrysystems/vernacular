---
type: guide
status: stable
sources:
- resource: skills/vernacular/SKILL.md
- resource: skills/vernacular/references/adopt.md
- resource: skills/vernacular/scripts/check.sh
- resource: AGENTS.md
- resource: README.md
---
# Agent workflow

The `skills/vernacular/SKILL.md` file is the agent integration. It keeps plans, research,
reports, experiments, benchmarks, and scratch outside the repository while leaving project code
and requested canonical docs in the checkout.

## At the start of work

1. Check availability without installing it from the agent:

   ```sh
   command -v vern
   ```

   If it is missing, say so once and do not write non-project material into the repository.
2. Check the local Git attachment:

   ```sh
   git config --get vernacular.workspace
   ```
3. If attached, run `vern status`, then read relevant material from `vern path apparatus` and
   `vern path working` before planning or reporting. Write scratch and reports to working. Put
   material worth keeping in apparatus and save it in apparatus's own Git history.
4. If not attached, ask before the first plan, research, report, scratch output, or bootstrap
   action that would create non-project material. `vern attach` can list a matching existing
   workspace; otherwise the approved setup is `vern init`. Code-only work in an established
   repository does not need a workspace.

The skill never puts machine-specific workspace paths into tracked project docs. It also never
runs `vern burn`; only the human performs that destructive action.

## Keep, scratch, and canon

Use the workspace spaces by intent:

- working is for disposable scratch, one-off output, and agent reports;
- apparatus is for durable product documents, decisions, research, experiments, and handoffs;
- canon is for ordinary project changes the task explicitly asks to keep.

To bring approved material into canon, use `vern promote SRC DEST`; do not copy it directly.
Promotion prints the file plan and, unless scanner findings block it, prompts before applying
unless `--yes`; it refuses overwrites, reports scan findings, and leaves staging and committing
to the human. To move material that landed in canon by mistake, use `vern stash`;
`vern stash --undo` restores the latest recorded move.

## Adoption of existing material

After a new `vern init`, or when setup is explicitly requested, follow
`skills/vernacular/references/adopt.md`:

- inspect `git status --short --ignored` and the project's ignore rules read-only;
- describe existing untracked or ignored material by kind before moving anything;
- offer move-all, choose, or not-now, with Enter accepting the recommended answer;
- keep durable material in apparatus shelves and put scratch in working;
- never move tracked files without explicit approval, and leave secrets, credentials, customer
  data, and regulated material in its proper system;
- write an adoption record with stash IDs and an undo block when anything moves.

Adoption does not stage or commit the project repository. It can move accepted material out of
the checkout (tracked paths only with explicit approval), remove a qualifying emptied source
folder, and commit an apparatus move to private apparatus history when that apparatus is
versioned. It does not touch build output, dependencies, caches, or harness state.

## Legibility check

From the checkout, the bundled read-only check is:

```sh
sh skills/vernacular/scripts/check.sh
```

It checks the command is available, the repository is attached, `vern status` names the ID,
`vern path canon` is this checkout, `vern attach` reports the last-seen checkout, and the
manifest names its own ID. `--durable` additionally exercises a throwaway clone, move, worktree,
`git clean -xfd`, and a missing `~/.vern`; it is not the default fast check.

For the space invariants and irreversible operations, read [the workspace model](model.md).
