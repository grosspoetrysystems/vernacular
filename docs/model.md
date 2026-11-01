---
type: reference
status: stable
sources:
- resource: README.md
- resource: internal/workspace/workspace.go
- resource: internal/workspace/init.go
- resource: internal/workspace/history.go
- resource: internal/workspace/status.go
- resource: internal/workspace/promote.go
- resource: internal/workspace/stash.go
---
# Workspace model

Vernacular separates project canon from material that supports work but is not part of the
project. The spaces are ordinary filesystem locations, resolved from a self-describing
workspace manifest.

| Space | Meaning | Persistence |
|---|---|---|
| **canon** | The attached Git checkout: source, committed documentation, and other project files. | The project's own Git history. |
| **apparatus** | Durable private product documents, research, decisions, experiments, and reports. | The default apparatus is its own local Git repository. |
| **working** | Scratch, temporary output, and disposable agent reports. | No history by design. |

## Attachment and discovery

`vern init` creates a workspace under `~/.vern/data/<id>/` with `apparatus/` and `working/`,
registers the ID under `~/.vern/workspaces/`, and stores the opaque ID in the attached
checkout's local Git config as `vernacular.workspace`. The workspace manifest records the
canon path and the two private spaces. It does not add a tracked file, ignore rule, hook, or
commit to canon.

`vern attach ID` attaches another checkout to an existing workspace. `vern attach DIR` also
re-registers a workspace directory, which is the recovery path when the registry is stale or
`~/.vern` was lost. Workspace lookup gives an attached checkout precedence over a nearby
manifest, so a moved repository and its worktrees can resolve the same workspace. Ordinary
lookup resolves the current attachment in memory; it does not refresh the persisted canon
last-seen path. `vern init` and `vern attach` refresh that path.

The default private spaces are required to stay outside canon. Path resolution follows real
paths for existing prefixes, and nested repositories or paths inside `.git` are not accepted as
canon destinations. These checks keep private material from becoming an accidental project
file.

## History and movement

- A default apparatus gets its own Git repository. Vernacular records stashes and other kept
  material there without requiring the project's signing setup; a human or agent can inspect
  that history for recovery. An apparatus path deliberately configured elsewhere belongs to its
  owner and is not automatically versioned.
- `vern promote SRC DEST` copies from apparatus or consumes from working into canon. It refuses
  an existing destination, scans files for a small set of secret/local-path shapes, and, unless
  findings block it, asks for confirmation unless `--yes` is used. `--allow-findings` is explicit
  when a scan finding is intentional; staging and committing remain with Git and the human. The
  scanner is a hygiene aid, not a security boundary.
- `vern stash SRC [SHELF]` moves material from canon to a shelf in working, or to apparatus with
  `--apparatus`. It keeps the source name, refuses to overwrite, and refuses tracked files
  unless `--tracked` is explicitly supplied. Each move is appended to `stash.jsonl`;
  `vern stash --undo [ID]` moves a recorded stash back without deleting its journal entry.

## Boundaries and irreversible actions

Vernacular does not fetch, push, publish, or modify the project's commit history. `promote` is
the intentional crossing from private space into canon; it never stages or commits. `stash` is
the opposite move and records an undo path. `vern burn` is different: after the exact workspace
ID is typed, it deletes the workspace, apparatus history, registry entry, and repository link.
It cannot recover that material and leaves spaces the human pointed outside the workspace alone.

For the agent-facing decision rules, see [Agent workflow](agent-workflow.md). For command syntax,
see [CLI reference](cli.md).
