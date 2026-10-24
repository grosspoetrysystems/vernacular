# Maintaining Vernacular

Vernacular's contribution policy is adapted from Helicopter v0.1.0, pinned to commit `cf83ae51567cd0801484cf6ead72cf599b58423a` and recorded in the [release manifest](https://github.com/grosspoetrysystems/helicopter/releases/download/v0.1.0/helicopter.json). The policy source was adopted into this repository and adapted for Vernacular's owners, checks, dependencies, security, and release paths. These files express policy; GitHub settings enforce it.

## Activate the repository

- [x] Configure the DCO sign-off check: the [DCO app](https://github.com/apps/dco) is installed on the organization, and the configurator requires its `DCO` check.
- [ ] Decide which bot or GitHub App identities, if any, may interact with the repository. List only approved bot logins in the `HELICOPTER_AUTOMATION_LOGINS` repository variable.
- [ ] Review the two-PR limit and issue-first categories in `CONTRIBUTING.md`.
- [ ] Create or replace the `needs-triage` and `automation-review` labels used by the issue forms.
- [ ] Test the pull request template and both acceptance/refusal paths of the intake workflow.
- [ ] Optionally add a `HELICOPTER_REVIEW_KEY` secret with a credit limit to enable the advisory first review.
- [ ] Keep `solo` while Vernacular has one maintainer, then switch to `contributor` or `team` when the maintainer arrangement changes. Run the configurator only after the acceptance and refusal checks pass.
- [ ] Publish reachable private security and conduct contacts.

## Maintainer modes

| Mode | Use when | Metadata exemption | Protected-branch reviews |
| --- | --- | --- | --- |
| `solo` | One person owns or maintains the project | The configured immutable user ID on a same-repository branch | Zero required approvals |
| `contributor` | Maintainers primarily receive outside contributions | None; every pull request follows the contribution contract | One CODEOWNER approval after the latest push |
| `team` | An internal team builds in the open and accepts outside contributions | Same-repository `OWNER`, `MEMBER`, or `COLLABORATOR` authors | One CODEOWNER approval after the latest push; administrators included |

An unset or invalid mode fails closed to `contributor`. In `team` mode, the queue also covers internal maintainer pull requests. In the other modes, it queues outside contributions only.

## GitHub settings

Review the dry run first. Vernacular keeps `check` and `DCO` required by default; `--extra-check` adds checks rather than replacing them.

```sh
python3 scripts/configure_repository.py solo \
  --repo grosspoetrysystems/vernacular \
  --dry-run
```

After the valid and refused draft-pull-request checks pass, apply the same command without `--dry-run`. The configurator protects the default branch, creates the queue labels, applies the repository security baseline where GitHub permits it, and protects `v*` release tags. It never changes GitHub settings until the dry-run flag is removed.

All modes:

- require `test-validator`, `validate-metadata`, `check`, and `DCO`;
- require resolved review conversations;
- block force pushes and branch deletion; and
- retain administrator bypass for emergency recovery in `solo` and `contributor` modes; `team` includes administrators.

`solo` requires zero approvals because no second reviewer exists. `contributor` and `team` require one approval and CODEOWNER review; both require approval after the latest push. Required checks must have reported successfully before making them required. Do not set the mode variables at organization scope: inherited variables can affect every repository.

The configurator also sets workflow permissions read-only by default, prevents workflows from approving pull requests, enables first-time-contributor approval, enables Dependabot and security settings where the plan permits, creates `queue:` labels, and sets `HELICOPTER_AUTOMATION_LOGINS=dependabot[bot]` only when that variable is unset. Review any manual-step output from GitHub rather than assuming every security feature is available on every plan.

For public repositories, set the concurrent non-draft pull-request limit for users without write access. Start at two unless maintainer capacity supports more; add trusted contributors to any bypass list deliberately.

## Protect the policy plane

`.github/CODEOWNERS` covers workflows, the validator, queue/configuration scripts, tests, governance files, module manifests, hooks, agent instructions, and Vernacular's release and security paths.

The intake and queue workflows use `pull_request_target`, `issue_comment`, and `schedule` because those events load workflows and scripts from the protected default branch and receive a write-capable token even for fork pull requests. Their safety depends on these invariants:

1. only event metadata, the GitHub API, and protected default-branch files are read; the diff is read as text, never applied;
2. the contributor head is never checked out, built, or executed;
3. the only secret is the optional model key, held by `first-review`, which runs PR-Agent pinned by image digest; and
4. write permission is job-scoped: `first-review` writes its review comment and labels, while queue jobs write queue labels and notices and hold no secret.

The `pull_request_target` workflow uses the base branch's workflow and trusted-base checkout. A newly changed workflow is not the runtime for a remote pull request until that workflow change is merged into the default branch.

| Job | Token | Secret |
| --- | --- | --- |
| `validate-metadata` | `contents: read` | none |
| `first-review` | `contents: read`; `issues`, `pull-requests: write` | `HELICOPTER_REVIEW_KEY` |
| `queue` (both workflows) | `contents`, `checks: read`; `issues`, `pull-requests: write` | none |

Bump the PR-Agent image deliberately: resolve a new digest with `docker buildx imagetools inspect pragent/pr-agent:VERSION-github_action --format '{{.Manifest.Digest}}'` and review its release notes first.

Keep builds and tests in a separate unprivileged `pull_request` workflow on ephemeral GitHub-hosted runners. Preserve Vernacular's existing `check.yml`, release workflow, DCO path, security route, and release attestations. Pin every external action to a reviewed full commit SHA and set `persist-credentials: false` unless a job genuinely needs to push.

## Intake order

1. Check the template (Change, Verification, Risk), DCO sign-off, bot authorization, issue requirement, and pull-request budget.
2. Reject duplicates, out-of-scope work, and unreviewable size before expensive CI.
3. Route protected paths, dependencies, generated artifacts, release changes, and security-sensitive changes to owners.
4. Run independent tests and security checks without repository secrets.
5. Use automated or AI review only as advisory evidence.
6. Merge through the protected branch after human review.

Use one actionable response: one reason, one policy link, one repair path. Do not deploy several bots that repeat the same refusal.

## Review ramp

Each pull request earns the next, more expensive step by passing the cheaper one. A new push starts again from the bottom.

| Step | Cost | Output | Then |
| --- | --- | --- | --- |
| Admission (`validate-metadata`) | $0 | refusal codes, or pass | refused: contributor's turn |
| First review (`first-review`, optional) | a fraction of a cent with the default model | one PR-Agent comment: key issues, effort, tests, security, merge recommendation | advisory: contributor fixes what it found |
| Queue | $0 | queue label, nudge, announced pause | human review and merge |

Admission and the optional first review are the automated responses on a pull request; the queue adds a notice only when a maintainer is late.

**First review.** Runs only when `HELICOPTER_REVIEW_KEY` exists, never on drafts, on opening and on every push, editing its one comment in place. `.pr_agent.toml` on the default branch holds the model, cost rail, output sections, and input rules. The review is advisory: it does not move the turn or block merging.

**Queue.** Every run recomputes state from the pull-request timeline; labels are outputs. The clock runs only while it is a maintainer's turn and restarts whenever the turn passes to a maintainer. It is the contributor's turn after refused admission or a maintainer's **Request changes**, until the contributor pushes or replies. Any maintainer comment or review restarts the clock.

- After `HELICOPTER_QUEUE_NUDGE_HOURS` (default 168, seven days), one comment mentions the requested reviewers.
- After `HELICOPTER_QUEUE_PAUSE_HOURS` (default 504, 21 days), the pull request is labelled `queue:no-capacity` and the contributor is told publicly. It stays open.
- Drafts are not queued. Internal maintainer pull requests are queued only in `team` mode.
- When another pull request that references the same issue merges while this one waits, the pull request is labelled `queue:possibly-superseded` and gets one comment naming it. Nothing closes automatically and the turn does not change; close them in a batch after review.

`contribution-queue.yml` reconciles every hour and on pull-request comments, so hour thresholds mean something. Maintainer reviews on fork pull requests are picked up by the next sweep. On a private repository, lower the cron frequency if day thresholds are enough.

## Risk and response

- **Routine:** Request a specific repair for missing fields, weak reproduction, or excessive scope.
- **Abuse:** Close duplicate, flooding, or unauthorized automation submissions, then apply PR or interaction limits.
- **Suspicious:** Quarantine dependency substitution, obfuscated files, reviewer prompt injection, or workflow escalation. Do not execute the contribution; involve security owners.
- **Incident:** Cancel runs, revoke identities, and freeze affected merges or releases after credential exposure, malicious workflow execution, a compromised bot or runner, or protected-branch compromise. Rotate possibly exposed credentials, preserve URLs, actor IDs, SHAs, run IDs, and logs, then rebuild from trusted state and use private disclosure.

Automation may triage or temporarily contain; a human must be able to reverse consequential moderation.

## Tune from evidence

Track maintainer minutes per accepted pull request, first-review latency, CI cost before triage, false refusal rate, major rewrite/revert rate, duplicate/spam rate, and repeat-contributor rate. Do not optimize for raw pull-request count, generated lines, or bot comments.

Do not publish response-time promises until someone owns them. If you publish targets, call them service objectives rather than guarantees.
