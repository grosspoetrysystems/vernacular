# AGENTS.md

## Vernacular workspace

This repository is project canon.

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
python3 tests/test_validate_pr.py
```

Keep the CLI to `init`, `attach`, `path`, `status`, and `promote` unless a new command has proven itself in use. Shell out to `git`; do not add a Git library.
