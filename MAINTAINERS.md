# Maintaining Vernacular

Vernacular's contribution policy is adapted from [Helicopter](https://github.com/grosspoetrysystems/helicopter). Its files express policy; GitHub settings enforce it.

## Activate the repository

- [ ] Configure the DCO sign-off check.
- [ ] Decide which bot or GitHub App identities, if any, may interact with the repository.
- [ ] Review the two-PR limit and issue-first categories in `CONTRIBUTING.md`.
- [ ] Create or replace the `needs-triage` and `automation-review` labels used by the issue forms.
- [ ] Test the pull request template and both acceptance/refusal paths of the intake workflow.
- [ ] Choose `solo` while Vernacular has one maintainer ([Helicopter maintainer operating modes](https://github.com/grosspoetrysystems/helicopter/blob/main/docs/maintainer-modes.md)): set `HELICOPTER_SOLO_MAINTAINER_ID` to the maintainer's immutable account ID (`gh api user -q .id`), then run the repository configurator after its acceptance and refusal checks pass. Switch to `team` or `contributor` when a second maintainer joins.
- [ ] Publish reachable private security and conduct contacts.

## GitHub settings

Run `python3 scripts/configure_repository.py MODE --repo grosspoetrysystems/vernacular` after the required checks have run successfully. The configurator protects the default branch and applies the selected mode.

All modes:

- require `test-validator`, `validate-metadata`, and `check`;
- require resolved review conversations;
- block force pushes and branch deletion; and
- retain administrator bypass for emergency recovery.

`solo` requires zero approvals because no second reviewer exists. `contributor` and `team` require one approval and CODEOWNER review; `team` also requires approval after the latest push.

For public repositories, set the concurrent non-draft PR limit for users without write access. Start at two unless maintainer capacity supports more; add trusted contributors to the bypass list deliberately.

Enable the security features appropriate to the repository and plan: private vulnerability reporting, dependency graph/review, secret scanning and push protection, code scanning, and release attestations/SBOMs.

## Protect the policy plane

`.github/CODEOWNERS` covers workflows, the validator, governance files, module manifests, hooks, and agent instructions. Add release code when it exists.

The metadata workflow uses `pull_request_target` because that event loads its workflow from the protected default branch. Its safety depends on four invariants:

1. only event metadata and protected default-branch files are read;
2. the contributor head is never checked out or executed;
3. no secret is referenced; and
4. token permissions remain read-only.

Put builds and tests in a separate `pull_request` workflow on ephemeral GitHub-hosted runners. Pin every external action to a reviewed full commit SHA and set `persist-credentials: false` unless a job genuinely needs to push.

## Intake order

1. Check template, accountable human, origin, bot authorization, issue requirement, and PR budget.
2. Reject duplicates, out-of-scope work, and unreviewable size before expensive CI.
3. Route protected paths, dependencies, generated artifacts, and security-sensitive changes to owners.
4. Run independent tests and security checks without repository secrets.
5. Use automated or AI review only as advisory evidence.
6. Merge through the protected ruleset after human review.

Use one actionable response: one reason, one policy link, one repair path. Do not deploy several bots that repeat the same refusal.

## Risk and response

- **Routine:** Request a specific repair for missing fields, weak reproduction, or excessive scope.
- **Abuse:** Close duplicate, flooding, or unauthorized automation submissions, then apply PR or interaction limits.
- **Suspicious:** Quarantine dependency substitution, obfuscated files, reviewer prompt injection, or workflow escalation. Do not execute the contribution; involve security owners.
- **Incident:** Cancel runs, revoke identities, and freeze affected merges or releases after credential exposure, malicious workflow execution, a compromised bot or runner, or protected-branch compromise. Rotate possibly exposed credentials, preserve URLs, actor IDs, SHAs, run IDs, and logs, then rebuild from trusted state and use private disclosure.

Automation may triage or temporarily contain; a human must be able to reverse consequential moderation.

## Tune from evidence

Track maintainer minutes per accepted pull request, first-review latency, CI cost before triage, false refusal rate, major rewrite/revert rate, duplicate/spam rate, and repeat-contributor rate. Do not optimize for raw pull request count, generated lines, or bot comments.

Do not publish response-time promises until someone owns them. If you publish targets, call them service objectives rather than guarantees.
