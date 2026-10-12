# Contributing to Vernacular

Vernacular welcomes human-authored, AI-assisted, and agent-authored work when it is useful, reviewable, and owned by an accountable person.

## Set up

Requires Go (version in `go.mod`), Git, Python 3.10+ for the contribution validator tests, [golangci-lint v2](https://golangci-lint.run/), and [Lefthook](https://lefthook.dev/).

```sh
lefthook install
golangci-lint fmt ./...
golangci-lint run ./...
go test -cover ./...
python3 tests/test_validate_pr.py
```

Commit messages are one-line [Conventional Commits](https://www.conventionalcommits.org/) headers plus a DCO sign-off (`git commit -s`).

## Human accountability

Every pull request needs an accountable human contributor. By submitting one, you confirm that you:

- reviewed the complete diff;
- understand and can explain the change;
- verified the evidence reported in the pull request;
- can revise the contribution in response to review; and
- have the right to submit it under Vernacular's license and contribution terms.

AI assistance does not transfer those responsibilities to a model, provider, agent, bot, or tool.

## AI and agent use

Trivial autocomplete and private explanatory use need no disclosure. Disclose AI assistance when it materially shaped the implementation, tests, issue, or pull request.

Choose exactly one origin in the pull request template:

- **Human-authored:** AI was not materially involved.
- **AI-assisted:** a human directed the work and materially authored, checked, and revised the result.
- **Agent-authored:** an agent performed substantial repository work; an accountable human still reviewed and owns the result.
- **Authorized automation:** a maintainer-approved bot or GitHub App acted within its registered purpose and permissions.

Agent-authored and authorized-automation pull requests must include the agent identity, immutable run ID, base commit SHA, capabilities, network mode, verification, and a sanitized provenance or run-log reference when available. Do not publish raw transcripts that may contain secrets, personal data, or restricted source material.

An agent must not:

- impersonate a person or conceal material automation;
- approve or merge its own contribution;
- bypass required checks or reviews;
- change policy or security controls merely to make its own work pass;
- use repository or deployment secrets in untrusted contribution CI;
- post substantive answers as though they were the accountable human's judgment; or
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

Unapproved bots and autonomous agents must not open issues, pull requests, reviews, or comments. Their accountable operator must request authorization with the **Automation authorization** issue form before they interact with Vernacular.

## Pull request contract

Each pull request must:

- solve one coherent problem;
- explain motivation and user-visible behavior;
- link an accepted issue when required;
- report commands and results, or explain why verification is not applicable;
- declare dependency, security, and workflow impact; and
- disclose material AI or agent involvement.

Use a draft pull request while work is incomplete. Maintainers may ask you to split unrelated, generated, mechanical, or otherwise unreviewable changes. Passing CI does not guarantee acceptance.

Contributors without write access may have at most **two non-draft pull requests** open at once unless a maintainer grants an exception.

## Review

Vernacular independently reruns relevant checks. Contributor and agent reports are useful context, not authoritative evidence. AI review is advisory and cannot replace required human approval.

Changes to workflows, security, authentication, dependencies, release machinery, contribution policy, or agent instructions require the owners named in `.github/CODEOWNERS`.

## Legal

Vernacular uses [Developer Certificate of Origin 1.1](https://developercertificate.org/). Sign off every commit:

```text
Signed-off-by: Your Name <your.email@example.com>
```

The accountable human or authorized organization, not an AI system, makes that certification.

## Security and conduct

Report vulnerabilities privately as described in `SECURITY.md`. Follow `CODE_OF_CONDUCT.md` in all project spaces.
