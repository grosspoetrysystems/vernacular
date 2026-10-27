---
okf_version: "0.2"
---
# Vernacular

Vernacular gives a Git repository two private companion spaces on the local machine. The
repository remains the **canon**; durable project-support material goes in **apparatus** and
scratch goes in **working**.

## Start here

- [The workspace model](model.md) explains the three spaces, attachment, history, and the
  boundaries around moving files.
- [CLI reference](cli.md) documents `vern` invocation, commands, flags, outputs, and failure
  behavior.
- [Agent workflow](agent-workflow.md) describes how the bundled Agent Skill decides where to
  read and write material.
- [Task workflows](workflows.md) gives short recipes for setup, saving, promotion, recovery,
  and reattachment.

## Product surface and scope

This repository contains a local Go CLI (`cmd/vern`) and an Agent Skill
(`skills/vernacular`). It does not contain a Vernacular HTTP service or web application, so
there is no product web workflow to document. A `dist/` directory produced by Poolboy is the
publication output for this documentation corpus, not a Vernacular runtime surface.

Vernacular is local-only: it does not fetch or push, and it does not commit changes in the
attached project. Read the [workspace model](model.md) before moving anything across the
canon boundary.
