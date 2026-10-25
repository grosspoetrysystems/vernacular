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

1. `vern status` shows the workspace and what is in it. No `adoption-*.md` in
   `vern path apparatus` means adoption never ran (`vern init` was run by hand): offer it
   once, steps 2–5 of `references/adopt.md`, then go straight back to the task. If it shows
   changes not in apparatus history yet, save them now (step 3) before starting.
2. Before planning or reporting, read what is relevant:
   - `vern path apparatus`: durable material — product documents, decisions,
     research, experiments.
   - `vern path working`: scratch and earlier agents' reports.
3. Write:
   - scratch, reports, and one-off output to `vern path working`;
   - material worth keeping to `vern path apparatus`, then save it in apparatus's own Git
     history: `git -C "$(vern path apparatus)" add -A && git -C "$(vern path apparatus)"
     commit -m "<what, in a few words>"`. A save point, not a chronology: one commit per
     piece of work. Don't ask first (it is private and local); tell the human in one line,
     e.g. "Saved to apparatus history: GPS-361 plan";
   - into the repository only what the task asks for as part of the project.
4. Material that should become part of the project moves only through
   `vern promote SRC DEST`, with the human's agreement. It consumes working sources after
   a successful copy; apparatus sources stay in place. Don't edit the source during
   promotion. If source cleanup fails, inspect both locations rather than rerunning blindly.
5. Step 3 means stashing is rare: it is for material that is in the repository anyway — a
   write that missed the workspace, or something that became worth keeping but is not
   public. Asked to port one, or catching your own misfire: `vern stash PATH` puts it in
   working, the default for anything new. If the human says to keep it, use
   `vern stash --apparatus PATH [SHELF]` onto the shelf that matches its kind —
   `ls "$(vern path apparatus)"`, or propose a new shelf if none fits. Don't ask which; say
   the move in one line ("`notes/` → working"). It keeps the name inside SHELF, never
   overwrites, and refuses tracked paths; add `--tracked` only after the human approves.
   A wrong stash comes back with `vern stash --undo` (the latest) or `--undo ID`.
6. Asked to tidy working, or it has become a pile: offer once to sort it onto shelves
   ("Tidying working" in `references/adopt.md`).

### Not set

Ask before the first step that produces material that is not part of the project: a plan,
research, a report, scratch — or before bootstrapping a new project (an empty directory, or
a repository with no commits), which produces both. Keeping the plan in chat to avoid the
question skips it; ask anyway. Code-only tasks in an established repository need nothing.

1. `vern attach` lists existing workspaces and the checkout each was last seen
   with. If one matches this repository (a moved or recloned checkout), ask
   whether to attach it: `vern attach ID`.
2. Otherwise ask, as one question the human settles with Enter: "No Vernacular workspace
   here. Run `vern init` so plans and research stay out of the repo? (Y/n)". A new project
   without Git yet: `git init` first; `vern init` works before the first commit.
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
- Never run `vern burn`; only the human does, at a terminal. It deletes apparatus and its
  history beyond recovery.
- Workspace paths are machine-specific: never write them into tracked files.
- What enters the repository is ordinary project work. Its commits, pull requests, and docs
  follow the project's own conventions and style guides, and never mention Vernacular, the
  workspace, apparatus, working, or promotion. Vernacular's vocabulary and trailers belong
  in apparatus history only, which never goes public.
- `vern init` changes only local Git config; it is safe in repositories you do
  not own, and leaves no trace in commits, patches, or pull requests.
