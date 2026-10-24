# Contributing to Vernacular

Vernacular welcomes useful, reviewable work however it was produced.

## Set up

Requires Go (version in `go.mod`), Git, Python 3.10+ for the contribution policy tests, [golangci-lint v2](https://golangci-lint.run/), and [Lefthook](https://lefthook.dev/).

```sh
lefthook install
golangci-lint fmt ./...
golangci-lint run ./...
go test -cover ./...
python3 -m unittest discover -s tests -v
```

Commit messages are one-line [Conventional Commits](https://www.conventionalcommits.org/) headers plus a DCO sign-off (`git commit -s`).

## Your agreement

You agree once, not on every pull request: sign off your commits under the [Developer Certificate of Origin](#legal), which certifies that you have the right to submit the change. Pull requests carry no production or authorship declarations; reviewers judge the change, its verification, and its risk.

What we expect of every contributor, however the change was produced: read the whole diff, be able to explain it, verify the evidence you report, and revise it in response to review. Using AI does not transfer those responsibilities to a tool.

## AI and agent use

You do not need to declare how a change was produced. If a tool helped, a commit trailer it adds is welcome, for example the Linux kernel's `Assisted-by: AGENT_NAME:MODEL_VERSION`. An AI tool never adds `Signed-off-by`; only a person certifies the DCO. Never publish raw transcripts that may contain secrets, personal data, or restricted source material.

An agent must not:

- impersonate a person or conceal material automation;
- approve or merge its own contribution;
- bypass required checks or reviews;
- change policy or security controls merely to make its own work pass;
- use repository or deployment secrets in untrusted contribution CI;
- post substantive answers as though they were a contributor's judgment; or
- flood Vernacular with speculative, duplicate, or unrelated submissions.

Maintainers moderate observable behavior and policy violations, not guesses that writing “looks AI-generated.”

## Before implementation

Open or reference an accepted issue before starting:

- a feature or public API change;
- a cross-cutting refactor;
- a dependency addition or replacement;
- authentication, authorization, or security-sensitive work; or
- a significant build, workflow, release, or policy change.

Small bug fixes, documentation corrections, and focused maintenance may skip issue-first review unless a maintainer says otherwise.

Unapproved bots and autonomous agents must not open issues, pull requests, reviews, or comments. Their operator must request authorization with the **Automation authorization** issue form before they interact with Vernacular.

## Pull request contract

The pull request template has three required sections:

- **Change:** the problem, the smallest solution, and its user-visible effect; link an accepted issue when required.
- **Verification:** the commands or steps you ran and what they showed, or why verification does not apply.
- **Risk:** check every area the change touches (dependencies, security, workflows, generated content), or `None of the above`.

Each pull request should solve one coherent problem. If a section is missing or empty, the admission check names it and says how to fix it.

Use a draft pull request while work is incomplete. Maintainers may ask you to split unrelated, generated, mechanical, or otherwise unreviewable changes. Passing CI does not guarantee acceptance.

When you have addressed review, push your changes and reply, or press **Re-request review**. Either one hands the pull request back to a maintainer. Queue labels show whose turn it is: `queue:awaiting-author` is yours and `queue:awaiting-reviewer` is ours.

Contributors without write access may have at most **two non-draft pull requests** open at once unless a maintainer grants an exception.

## Review

Vernacular independently reruns relevant checks. Contributor reports are useful context, not authoritative evidence. AI review is advisory and cannot replace required human approval.

Changes to workflows, security, authentication, dependencies, release machinery, contribution policy, or agent instructions require the owners named in `.github/CODEOWNERS`. Changes that delete or consume source material after promotion are security-sensitive and require the same careful review.

## Legal

Vernacular uses [Developer Certificate of Origin 1.1](https://developercertificate.org/). Sign off every commit:

```text
Signed-off-by: Your Name <your.email@example.com>
```

A person or authorized organization, not an AI system, makes that certification. The [DCO app](https://github.com/apps/dco) supplies the required `DCO` check.

## Security and conduct

Report vulnerabilities privately as described in `SECURITY.md`. Follow `CODE_OF_CONDUCT.md` in all project spaces.
