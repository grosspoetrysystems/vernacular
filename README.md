# Vernacular

Vernacular gives a Git repository two private companion spaces on your machine, so the
material a project generates while it is being built does not have to live in the repository
or be thrown away.

- **canon** — the Git repository. What the project promises.
- **apparatus** — durable work that is not canon: product documents, research, decisions,
  experiments, personal method.
- **working** — scratch, agent reports, temporary output. Disposable.

Apparatus and working live under `~/.vern/data/<id>/`, outside the repository. `git clean -xfd`
cannot reach them, no `.gitignore` entry is needed, and nothing in the repository changes except
one line of local Git config. Nothing reaches commits, patches, or pull requests, so Vernacular
is as usable in a repository you clone to fix one thing or review as in your own.

This matters most with coding agents, which generate plans, reports, and research that otherwise
land in the repository and get committed, and which work better when they can read that material
back.

## Install

```sh
npm install --global @grosspoetrysystems/vernacular   # from v0.1.0; prebuilt binary and the agent skill
go install github.com/grosspoetrysystems/vernacular/cmd/vern@latest
```

`vern` must be on your `PATH` (`go install` puts it in `$(go env GOPATH)/bin`). Release
archives on the [releases page](https://github.com/grosspoetrysystems/vernacular/releases) carry a
cosign-signed checksum file and GitHub build provenance (`gh attestation verify FILE --repo
grosspoetrysystems/vernacular`); npm packages are published with provenance.

## Agents

Agents learn about Vernacular from the `vernacular` [agent skill](skills/vernacular/SKILL.md),
installed once per machine into each harness's user skill directory. It tells an agent to check
`git config vernacular.workspace`, read the workspace before planning, write scratch and research
there instead of into the repository, and ask before running `vern init` where there is no
workspace yet. Nothing is added to any repository or to your instruction files.

Right after `vern init`, or when asked to set up Vernacular, the skill follows
[`references/adopt.md`](skills/vernacular/references/adopt.md). It looks at what the repository
already hides from Git (ignored and untracked material, cross-referenced with your ignore rules),
describes it by kind ("benchmarks, ADRs, planning docs"), and lets you pick which kinds to move.
Picked material is organized into shelves in the workspace with file names unchanged, and an undo
block is written alongside. Anything you keep elsewhere (a notes folder, an app) stays where it is.
[`scripts/check.sh`](skills/vernacular/scripts/check.sh) confirms the attachment;
`--durable` also proves it survives `git clean -xfd`, a moved repository, a worktree, a reclone,
and a lost `~/.vern`, on a throwaway clone.

```sh
npx skills add grosspoetrysystems/vernacular -g   # detects installed agents, links the skill
```

Or link it by hand from a checkout of this repository:

| Harness | User skill directory |
|---|---|
| Claude Code | `~/.claude/skills/` |
| Codex, pi, OMP | `~/.agents/skills/` |
| Antigravity CLI | `~/.gemini/antigravity-cli/skills/` |
| Cursor | `~/.cursor/skills/` |
| GitHub Copilot | `~/.copilot/skills/` |
| Gemini CLI | `~/.gemini/skills/` |

```sh
mkdir -p ~/.claude/skills && ln -s "$PWD/skills/vernacular" ~/.claude/skills/vernacular
```

Sandboxed harnesses also need access to `~/.vern`, once per machine:

- Claude Code: `"permissions": {"additionalDirectories": ["/Users/you/.vern"]}` in
  `~/.claude/settings.json`.
- Codex: `writable_roots = ["/Users/you/.vern"]` under `[sandbox_workspace_write]` in
  `~/.codex/config.toml`.
- Antigravity CLI: `--add-dir ~/.vern`.

Removing the skill is deleting the link.

## Use

```sh
cd my-project
vern init                 # attach this repository to a new workspace
vern path working         # absolute path for scratch
vern path apparatus       # absolute path for durable non-canonical work
vern status               # workspace, spaces, and what exists outside canon
vern stash run.log logs   # move run.log out of the repository onto the "logs" shelf in working
vern stash --apparatus PLAN.md plans   # keep it: onto the "plans" shelf in apparatus
vern promote "$(vern path apparatus)/notes/design.md" docs/design.md
```

`vern status` lists the most recently changed files in apparatus and working, so one command
shows an agent (or you) what is there.

`vern init` also prints an optional `AGENTS.md` section for repositories you own and want to
describe themselves to agents without the skill.

`vern promote` copies from apparatus or working into canon. It never overwrites, stages, or
commits; it refuses destinations the repository ignores, and refuses files that look like
secrets or local paths unless `--allow-findings` is given.

`vern stash` is the opposite: it moves a path out of the repository onto a shelf (a top-level
folder) in working, or in apparatus with `--apparatus` for material worth keeping, keeping its
name. It never overwrites, refuses tracked paths unless `--tracked` is given, and prints the
`mv` that undoes it.

## Attachment

`vern init` stores an opaque ID in the repository's local Git config (`vernacular.workspace`)
and registers the workspace in `~/.vern/workspaces/<id>`. The attachment follows the repository
when it moves and is shared by its worktrees.

```sh
vern attach                             # list registered workspaces
vern attach vrn_1a2b3c4d                # attach this repository to an existing workspace
vern attach ~/.vern/data/vrn_1a2b3c4d   # re-register a moved workspace, or a lost ~/.vern
git config --unset vernacular.workspace # detach
```

Run `vern help` for the full command surface.

## Contributing

See `CONTRIBUTING.md`. Vernacular is [MIT](LICENSE) licensed.
