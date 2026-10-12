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
one line of local Git config.

This matters most with coding agents, which generate plans, reports, and research that otherwise
land in the repository and get committed.

## Install

```sh
go install github.com/grosspoetrysystems/vernacular/cmd/vern@latest
```

## Use

```sh
cd my-project
vern init                 # attach this repository to a new workspace
vern path working         # absolute path for scratch
vern path apparatus       # absolute path for durable non-canonical work
vern status               # workspace, spaces, and what exists outside canon
vern promote "$(vern path apparatus)/notes/design.md" docs/design.md
```

`vern init` prints a snippet to add to `AGENTS.md` so agents write to the right space.

`vern promote` copies from apparatus or working into canon. It never overwrites, stages, or
commits; it refuses destinations the repository ignores, and refuses files that look like
secrets or local paths unless `--allow-findings` is given.

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
