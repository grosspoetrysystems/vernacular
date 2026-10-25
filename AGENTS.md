# AGENTS.md

## Vernacular workspace

This repository is project canon. Its working context lives outside it:
product documents, decisions, research, experiments, and earlier agents'
reports. Before planning work or reporting on status or direction, list
and read what is relevant under `vern path apparatus` and `vern path working`.

Do not create exploratory reports, plans, research, benchmark output,
or temporary Markdown in this repository unless explicitly requested.

For temporary research, generated reports, and scratch files, write under
the path printed by `vern path working`.
For durable non-canonical work, use `vern path apparatus`.

Only move material into this repository through an intentional
project change, such as `vern promote`.

## Checks

```sh
golangci-lint fmt ./...
golangci-lint run ./...
go test -cover ./...
python3 -m unittest discover -s tests -v
```

Keep the CLI to `init`, `attach`, `path`, `status`, `promote`, `stash`, and `burn` unless a new command has proven itself in use. Shell out to `git`; do not add a Git library.
