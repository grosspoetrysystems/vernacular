---
type: guide
status: stable
sources:
- resource: internal/cli/cli.go
- resource: internal/workspace/init.go
- resource: internal/workspace/promote.go
- resource: internal/workspace/stash.go
- resource: internal/workspace/burn.go
- resource: skills/vernacular/SKILL.md
- resource: skills/vernacular/references/adopt.md
---
# Task workflows

These recipes assume `vern` is installed and commands are run from the relevant checkout.
They keep the private/canonical boundary explicit; read [the CLI reference](cli.md) for all
flags and [the workspace model](model.md) for the underlying rules.

## Set up a repository before planning

```sh
command -v vern
git config --get vernacular.workspace
vern attach                         # inspect registered candidates, if any
vern init                            # only after the human approves setup
vern status
```

`vern init` creates a new workspace when the checkout is unattached. After setup, offer the
adoption flow described in [Agent workflow](agent-workflow.md) rather than moving files silently.
If an existing workspace belongs to the checkout, `vern init` reports it instead of creating a
second one.

## Start an agent session

```sh
vern status
vern path apparatus
vern path working
```

Read only the apparatus and working material relevant to the task. Put plans, research, and
reports in working unless they are worth keeping; save durable material in apparatus's private
Git history. Keep project code and requested project documentation in canon. The bundled skill
requires this read-before-planning behavior when a workspace is attached.

## Promote a reviewed document

1. Put the draft in apparatus or working.
2. Preview the crossing and its scanner findings:

   ```sh
   vern promote "$(vern path apparatus)/design/overview.md" docs/overview.md
   ```

   This prints the plan and, unless scanner findings block it first, prompts `Promote? [y/N]`.
   It is not read-only: answering `y` applies the promotion immediately. To inspect without
   applying, answer `N` or cancel, then rerun with `--yes` once the plan is correct. If findings
   are intentional, include `--allow-findings` on the preview and rerun. If you answer `y`,
   promotion is already complete; do not rerun because existing destinations are never overwritten.
3. Review the destination with Git. `vern` never stages or commits it.

Apparatus sources remain available after promotion. Working sources are consumed only after the
copy succeeds. Existing destinations are never overwritten.

## Move accidental repository material out

```sh
vern stash draft.log logs
# keep an approved document rather than scratch:
vern stash --apparatus PLAN.md plans
# restore the latest move if it was a mistake:
vern stash --undo
```

A stash prints its ID and an exact undo command. Tracked paths require the explicit `--tracked`
flag because Git will report their deletion; obtain approval before using it. The stash journal
is append-only, so undo records the reversal instead of erasing history.

## Reattach a moved checkout or lost registry

```sh
vern attach                           # find the workspace ID
vern attach vrn_…                     # attach by ID
vern attach /path/to/workspace        # re-register a workspace directory
vern status
```

The directory form repairs the registry when the workspace is still present in its default
location or when its directory moved. It does not switch to an older copy merely because a
registry pointer is stale; inspect any reported error before retrying.

## Run the local skill check

```sh
sh skills/vernacular/scripts/check.sh
```

Use `sh skills/vernacular/scripts/check.sh --durable` only when the extra throwaway-clone
exercise is useful. The durable mode cleans only its temporary copy; it does not alter the
checkout's files, but it does exercise attachment recovery and `git clean -xfd` on that copy.

## Remove a workspace

Do not automate this from an agent. The human may run:

```sh
vern burn
```

Vernacular prints counts for apparatus and working files plus any unsaved apparatus changes, not
individual filenames, then requires the exact workspace ID. Confirmation deletes the workspace
and its apparatus history, registry entry, and repository link. It is intentionally irreversible;
externally configured spaces are left alone.

## Product web boundary

There is no Vernacular web server, HTTP API, or website workflow in this repository. `vern`
operates on local Git metadata and local filesystem paths. Do not invent a browser or network
workflow; the Poolboy-rendered `dist/` site belongs to this documentation experiment, not to the
product.
