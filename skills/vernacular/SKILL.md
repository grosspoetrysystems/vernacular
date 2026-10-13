---
name: vernacular
description: Use whenever work produces or needs material that supports building a project but is not part of it — research, competitor analysis, plans, design notes, decisions, experiments, benchmarks, evals, scratch pads, side quests, agent reports, handoffs, build or meta infrastructure — at the start of work in a Git repository, before planning or reporting on project status or direction, and when setting up or adopting Vernacular in a new or existing repository. Finds and uses the project's private Vernacular workspace outside the repository so this material is kept, findable, and never committed.
---

# Vernacular workspace

Vernacular keeps a project's working material in a private workspace outside the
repository. The repository holds one line of local Git config and nothing else.
That material is often the best context for the work: read it before planning,
and write new material there instead of into the repository.

## 1. Is vern available?

    command -v vern

Missing: say so once, and write nothing non-project into the repository. Do not
install it yourself.

## 2. Does this repository have a workspace?

    git config --get vernacular.workspace

### Set

1. `vern status` shows the workspace and what is in it.
2. Before planning or reporting, read what is relevant:
   - `vern path apparatus`: durable material — product documents, decisions,
     research, experiments.
   - `vern path working`: scratch and earlier agents' reports.
3. Write:
   - scratch, reports, and one-off output to `vern path working`;
   - material worth keeping to `vern path apparatus`;
   - into the repository only what the task asks for as part of the project.
4. Material that should become part of the project moves only through
   `vern promote SRC DEST`, with the human's agreement.
5. Asked to port something from the repository into the workspace: put it on the shelf
   (top-level folder of apparatus) that matches its kind — `ls "$(vern path apparatus)"` —
   or propose a new shelf if none fits. Say the move in one line ("`notes/` → `research/`"),
   then `mv`; never overwrite, never rename files, and ask first if the path is tracked.
6. Asked to tidy working, or it has become a pile: offer once to sort it onto shelves
   ("Tidying working" in `references/adopt.md`).

### Not set

Only act when you are about to write material that is not part of the project.

1. `vern attach` lists existing workspaces and the checkout each was last seen
   with. If one matches this repository (a moved or recloned checkout), ask
   whether to attach it: `vern attach ID`.
2. Otherwise ask: "No Vernacular workspace here. Run `vern init`?"
3. Proceed only on yes. Until then, keep the material in the conversation; do not
   write it into the repository.
4. After `vern init`, offer to bring in what the human already keeps (steps 2–5 of
   `references/adopt.md`), then go straight back to the task.

## Setting up

When the human asks to set up or adopt Vernacular, is bootstrapping a new project, or right
after `vern init`, follow `references/adopt.md` in this skill's directory: look, offer the
kinds of material found as a pick list, shelve only what was picked, finish in three lines.

## Rules

- Never commit, stage, or copy workspace material into the repository yourself.
- Workspace paths are machine-specific: never write them into tracked files.
- `vern init` changes only local Git config; it is safe in repositories you do
  not own, and leaves no trace in commits, patches, or pull requests.
