# Adopting Vernacular

Everyone keeps working material differently: a folder with any name, files at the root, a
notes folder elsewhere, an app, or nowhere. Adoption meets it where it is, takes a few
questions, and ends with the human back at work: the repository clean and ready to share,
their material findable by agents, nothing lost.

## Tone

- Short. One question at a time; a "no" ends adoption immediately.
- Describe what you see in the human's terms ("benchmarks, ADRs, planning docs"), never
  Vernacular's.
- Show before you move. Nothing moves, regroups, or changes without a yes.
- No lecture about spaces, canon, or the PRD. The result is the explanation.

## What this never does

- Commit, stage, or delete anything.
- Rename or edit the human's files. Shelving changes folders, never file names or contents.
- Move tracked files, or edit tracked files, without explicit approval for each one.
- Touch dependencies, build output, caches, or tool and harness state (`node_modules`,
  `dist`, `bin`, `target`, `coverage`, `.venv`, `.omp/`, `.claude/`, `.wrangler/`, …).
- Handle governed material (secrets, credentials, customer or regulated data): say it
  belongs in a proper system and leave it in place.

## 1. Attach

If `git config --get vernacular.workspace` is empty: `vern init` (after the human's yes from
the main skill), or `vern attach ID` if `vern attach` lists a workspace last seen with
another checkout of this repository.

New project: run `vern init` before any bootstrap tool or agent generates material, so the
first commit never sees it. It works before the first commit. Then skip to step 5.

## 2. Look (read-only, seconds)

    git status --short --ignored        # ?? untracked, !! ignored

Peek inside each untracked or ignored entry that is not on the never-touch list: file
names, extensions, a few headings, sizes (`du -sh`). Enough to describe it, not to audit it.
Tracked files that read as working material (a committed `PLAN.md`, `REPORT.md`, notes
folder) can be offered too, marked as tracked: moving them is a project change.

Cross-reference the ignore rules. They are where people already said "this is mine, keep it
out", whatever they named it:

    cat .git/info/exclude                               # personal, never committed
    git config --get core.excludesFile                  # global; default ~/.config/git/ignore
    git ls-files '.gitignore' '*/.gitignore'            # the project's own rules
    git check-ignore -v PATH                            # which rule hides PATH

- A rule in `.git/info/exclude` or the global file (`notes/`, `scratch/`, `*.local.md`) is
  a strong sign of personal material.
- A rule in the tracked `.gitignore` (`/bench-output`) means the project already
  acknowledges the path; tools or docs may depend on it.
- A directory that hides itself (its own `.gitignore` holds `*`) shows as `!!` in
  `git status --ignored`, but `git check-ignore` on the directory says not ignored; ask about
  a file inside it instead.
- A rule that matches nothing today is history, not material; mention it only at the end.

Note anything tracked files refer to: `git grep -n -F 'NAME'`. Moving it would break those
references.

## 3. Offer

Describe what you found by kind, with rough counts and sizes, and let the human pick which
kinds to move and shelve. Whatever they leave unpicked stays exactly where it is.

> I see benchmarks (40 runs, 1.2 GB), ADRs (6), planning docs (3), and competitor analysis
> notes (2). Which should I move into your Vernacular workspace and organize into shelves?
>
> - [ ] Benchmarks — `.substrate/bench/`
> - [ ] ADRs — `.substrate/adr/`
> - [ ] Planning docs — `PLAN.md`, `ROADMAP-draft.md`
> - [ ] Competitor analysis — `notes/competitor-*.md`

If your harness has a multi-select question tool (a checkbox list toggled with the space
bar), use it: one option per kind, labelled with the kind and its count, the paths in the
description, plus "None, not now". Otherwise print the numbered list and ask which numbers
to move. One kind per option, never one option per file; split a kind only if the human
asks.

If nothing turned up, ask once: "Do you already keep notes for this project anywhere — a
folder somewhere else, an app?"

| Answer | Do |
|---|---|
| None / not now | Stop. Back to the task. |
| Some kinds picked | Step 4, with only those kinds. |
| A folder outside the repository | Use it in place: set `spaces.apparatus.path` in the workspace's `vernacular.yml` (`vern status` prints its directory) to that folder's absolute path. Nothing moves, nothing is regrouped. |
| An app or a notepad | Add an *Elsewhere* section to `$(vern path apparatus)/README.md`: one line per place, what lives there and how to reach it. Offer to paste anything they want kept as files. |

## 4. Shelve and move

Propose the result as a tree before touching anything:

    apparatus/
      bench/           ← .substrate/bench/       (already a shelf: name kept)
      adr/             ← .substrate/adr/         (already a shelf: name kept)
      plans/           ← PLAN.md, ROADMAP-draft.md         (loose files: new shelf)
      research/
        competitors/   ← notes/competitor-*.md             (loose files: new shelf)

- Shelves come from the material, not a fixed taxonomy; use the human's words where they
  have them. Add a shelf when material needs one, not to fill a set. File names never change.
- Existing organization outranks a tidier layout. A folder that is already shelved keeps its
  shelf names and moves shelf by shelf as it is; if it has its own README or index, keep it
  and its relative links working. Cohesive trees (a benchmark run, an artifact set, a spike)
  move intact. Propose new shelves only for loose files.
- Scratch, logs, and one-off output go to `vern path working` instead, said in the preview.
- Flag anything tracked files reference; suggest leaving it unless the human wants to fix
  the references.

On yes (or after their edits), say "Porting now." and, for each row, `mkdir -p` the shelf and `mv` the source into
it. Never overwrite: if a destination exists, stop and ask. Tracked files move only with
explicit approval, and the deletion then shows in `git status` for the human to commit.

Then write or update the shelf index, `$(vern path apparatus)/README.md`, so a cold agent
knows what each shelf holds before opening it. Keep an existing index's wording; add rows
for new shelves only:

    | Shelf | Contents | How to use it |
    |---|---|---|
    | benchmarks/ | Benchmark runs and the harness to replay them | Rerun before citing; old numbers are not current |

Write `$(vern path apparatus)/adoption-YYYY-MM-DD.md`: what moved where, references left to
fix, and an undo block that reverses every move, with literal absolute paths so it can be
pasted as is:

```sh
mkdir -p "/Users/me/code/foo" && mv "/Users/me/.vern/data/vrn_7f29c1a0/apparatus/plans/PLAN.md" "/Users/me/code/foo/PLAN.md"
```

Leave ignore rules alone, even ones that matched only what moved: a rule like `/notes` may
be there for future material, and editing the tracked `.gitignore` is a project change. List
them in the record; edit one only if the human asks.

## 5. Finish

    sh "<this skill's directory>/scripts/check.sh"     # fast; reads only

Then tell the human, in a few short lines, and go back to the original task. For example:

> Ported benchmarks, ADRs, and planning docs into your workspace. `git status` is clean;
> `vern status` shows where they went, and agents read them before planning.
> To port anything else later, just ask, or from a terminal:
> `mv PATH "$(vern path apparatus)/"`. Undo is in `adoption-YYYY-MM-DD.md`.

If tracked deletions were approved, say they are waiting in `git status` instead of "clean".

Report `check.sh` output only if something failed. Friction that needs a change to `vern`
is a ticket, not a workaround.

`check.sh --durable` additionally proves the attachment survives a reclone,
`git clean -xfd`, a moved repository, a worktree, and a lost `~/.vern`, on a throwaway
clone. It copies the repository, so offer it rather than running it by default.

## Tidying working

Working is scratch and makes no promise of order. When the human asks to tidy it, or
`vern status` shows a pile of loose files at its top level, offer once:

> Working has 30 loose files: agent reports, probe logs, a few drafts. Want me to sort
> them? Drafts worth keeping go to apparatus shelves; the rest onto shelves in working.

Same rules as adoption: a pick list by kind, a preview tree, `mv` only, no renames, no
deletion, an undo block in the record. Nothing is thrown away; clearing working out stays
the human's call.
