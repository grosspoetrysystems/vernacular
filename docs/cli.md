---
type: reference
status: stable
sources:
- resource: cmd/vern/main.go
- resource: internal/cli/cli.go
- resource: internal/workspace/init.go
- resource: internal/workspace/promote.go
- resource: internal/workspace/stash.go
- resource: internal/workspace/burn.go
---
# `vern` CLI reference

The `vern` executable dispatches to the local workspace package. Run it inside the repository
for repository-attached operations. `vern --version` prints the release or build version, and
`vern help` prints the same command usage embedded in the CLI.

Flags may appear before, between, or after positional arguments. `--` ends flag parsing and
makes the remaining arguments positional. A successful command returns exit code `0`; unknown
commands and usage errors return `2`; operational failures return `1`. Help and version are
successful read-only commands.

## `vern init`

```sh
vern init
```

Run with no arguments inside a Git checkout. A newly created workspace is attached, the default
apparatus history is created, and the three space paths are printed. An already-attached
checkout instead reports the existing attachment and any saved-history notice and refreshes
canon's last-seen path, without reprinting the space paths. Tracked files are not changed in
either case. It rejects positional arguments.

## `vern attach [ID | DIR]`

```sh
vern attach                 # list registered workspaces
vern attach vrn_…           # attach by workspace ID
vern attach /path/to/ws     # attach and re-register by workspace directory
```

With no argument, lists registered workspaces and marks broken entries. With one ID or directory,
attaches the current checkout to that workspace. A previous different attachment is detached in
local Git config; its files are not touched. More than one argument is a usage error.

## `vern path canon|apparatus|working`

```sh
vern path canon
vern path apparatus
vern path working
```

Prints one absolute space path for scripts. `canon` is meaningful inside an attached checkout;
`apparatus` and `working` resolve through the workspace. An unknown space or missing attachment
is an error.

## `vern status`

```sh
vern status
```

Shows the workspace ID and root, canon when known, and a bounded recent listing for apparatus
and working. It reports missing spaces and, for a versioned apparatus, changes not yet in
apparatus history. It takes no arguments. It does not modify project checkout files, but
resolving the workspace may repair a missing or stale registry pointer in `~/.vern/workspaces/`
and print a notice to stderr.

## `vern promote [--yes] [--allow-findings] SRC DEST`

```sh
vern promote [--yes] SRC DEST
vern promote --allow-findings SRC DEST
```

Promotes a source inside the workspace's apparatus or working space into a destination inside a
checkout attached to that workspace. A destination must not already exist; an existing directory
or a destination ending in a separator receives the source basename. Apparatus sources are
copied and retained. Working sources are moved and consumed only after the planned copy succeeds.

The command prints the plan and scans files for possible secrets or local paths. Findings block
before confirmation unless `--allow-findings` is explicit; without blocking findings, it asks for
`y`/`yes` unless `--yes` (or `-y`) is supplied. The scan is deliberately small and is not a
substitute for a secret-management system. A successful promotion prints Git's status for the
destination; it does not stage or commit it.

## `vern stash [--apparatus] [--tracked] SRC [SHELF]`

```sh
vern stash report.txt reports
vern stash --apparatus PLAN.md plans
vern stash --tracked tracked-note.md notes
```

Moves a source inside the attached checkout to a shelf in working, or to apparatus when
`--apparatus` is supplied. The source name is preserved and existing destinations are refused.
Tracked paths are refused unless `--tracked` is supplied, because that makes Git report the
tracked deletion. The result includes a stash ID and an undo command. Apparatus stashes are
saved to apparatus history when possible.

To restore the latest recorded move, use `vern stash --undo`. To restore a specific move, use
`vern stash --undo ID`. Undo accepts only the optional ID; it never overwrites an existing
canon destination.

## `vern burn`

```sh
vern burn
```

Shows the workspace and unsaved apparatus changes, then requires the exact workspace ID. On
confirmation it deletes the workspace directory, apparatus history, registry entry, and local
repository link. There is no recovery for deleted workspace material. Paths configured outside
the workspace are left alone. It takes no arguments.

See [the workspace model](model.md) for the invariants behind these commands and [task
workflows](workflows.md) for copyable sequences.
