#!/usr/bin/env python3

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from configure_repository import baseline, protection, tag_ruleset
from validate_pr import validate_body

SCRIPT = ROOT / "scripts" / "validate_pr.py"
TEMPLATE = ROOT / ".github" / "pull_request_template.md"


VALID = """## Change

Small fix.

## Verification

Ran the check.

## Risk

- [x] None of the above
"""


def run_cli(event: dict[str, object], *arguments: str) -> subprocess.CompletedProcess[str]:
    with tempfile.TemporaryDirectory() as directory:
        event_path = Path(directory) / "event.json"
        event_path.write_text(json.dumps(event), encoding="utf-8")
        return subprocess.run(
            [sys.executable, str(SCRIPT), str(event_path), *arguments],
            check=False,
            capture_output=True,
            text=True,
        )


def owner_event(
    *,
    author_id: int = 1,
    author_type: str = "User",
    owner_type: str = "User",
    owner_id: int = 1,
    sender_id: int = 1,
    sender_type: str = "User",
    event_repository_id: int = 100,
    head_repository_id: int = 100,
    author_association: str = "OWNER",
) -> dict[str, object]:
    return {
        "repository": {"id": event_repository_id},
        "sender": {"id": sender_id, "type": sender_type},
        "pull_request": {
            "author_association": author_association,
            "body": "",
            "user": {"id": author_id, "type": author_type},
            "base": {"repo": {"id": 100, "owner": {"id": owner_id, "type": owner_type}}},
            "head": {"repo": {"id": head_repository_id}},
        },
    }


def codes(body: str) -> set[str]:
    return {refusal.code for refusal in validate_body(body)}


class ValidateBodyTests(unittest.TestCase):
    def test_minimal_submission_passes(self) -> None:
        for newline in ("\n", "\r\n"):
            with self.subTest(newline=newline):
                self.assertEqual(codes(VALID.replace("\n", newline)), set())

    def test_each_missing_section_is_named(self) -> None:
        for section in ("Change", "Verification", "Risk"):
            with self.subTest(section=section):
                body = VALID.replace(f"## {section}", "## Other")
                refusals = validate_body(body)
                self.assertIn("REFUSE_TEMPLATE_INCOMPLETE", {r.code for r in refusals})
                self.assertIn(f"## {section}", " ".join(r.message for r in refusals))

    def test_unfilled_template_is_refused_with_specific_fixes(self) -> None:
        self.assertEqual(
            codes(TEMPLATE.read_text(encoding="utf-8")),
            {"REFUSE_CHANGE_EMPTY", "REFUSE_VERIFICATION_EMPTY", "REFUSE_RISK_UNCHECKED"},
        )

    def test_filled_template_passes(self) -> None:
        body = (
            TEMPLATE.read_text(encoding="utf-8")
            .replace("Closes #", "Fixes the queue label.\n\nCloses #12")
            .replace("<commands and concise result; project CI verifies independently>", "unittest: 40 passed")
            .replace("- [ ] None of the above", "- [x] None of the above")
        )
        self.assertEqual(codes(body), set())


    def test_cli_accepts_a_valid_event(self) -> None:
        result = run_cli({"pull_request": {"body": VALID}})

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_solo_mode_requires_configured_maintainer(self) -> None:
        owner = owner_event()
        enabled = ("--mode", "solo", "--solo-maintainer-id", "1")
        cases = {
            "personal owner": (owner, 0),
            "organization maintainer": (
                owner_event(owner_id=200, owner_type="Organization"),
                0,
            ),
            "wrong account": (owner_event(author_id=2), 1),
            "bot": (owner_event(author_type="Bot"), 1),
            "wrong repository": (owner_event(event_repository_id=101), 1),
            "different sender": (owner_event(sender_id=2), 1),
            "bot sender": (owner_event(sender_type="Bot"), 1),
            "fork": (owner_event(head_repository_id=101), 1),
        }
        for name, (event, expected_code) in cases.items():
            with self.subTest(name=name):
                result = run_cli(event, *enabled)
                self.assertEqual(result.returncode, expected_code, result.stdout + result.stderr)

        for invalid_id in ("", "0", "-1", "+1", "01", " 1 ", "owner"):
            with self.subTest(invalid_id=invalid_id):
                rejected = run_cli(owner, "--mode", "solo", "--solo-maintainer-id", invalid_id)
                self.assertEqual(rejected.returncode, 1, rejected.stdout + rejected.stderr)

    def test_contributor_mode_exempts_nobody(self) -> None:
        for mode in ("contributor", "invalid"):
            with self.subTest(mode=mode):
                result = run_cli(owner_event(), "--mode", mode, "--solo-maintainer-id", "1")
                self.assertEqual(result.returncode, 1, result.stdout + result.stderr)

    def test_team_mode_exempts_trusted_same_repository_authors(self) -> None:
        for association in ("OWNER", "MEMBER", "COLLABORATOR"):
            with self.subTest(association=association):
                result = run_cli(
                    owner_event(
                        author_association=association,
                        owner_id=200,
                        owner_type="Organization",
                    ),
                    "--mode",
                    "team",
                )
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

        refused = (
            owner_event(author_association="NONE", owner_id=200, owner_type="Organization"),
            owner_event(author_association="MEMBER", head_repository_id=101),
            owner_event(author_association="MEMBER", sender_id=2),
            owner_event(author_association="MEMBER", author_type="Bot"),
        )
        for event in refused:
            result = run_cli(event, "--mode", "team")
            self.assertEqual(result.returncode, 1, result.stdout + result.stderr)


class AuthorizedAutomationTests(unittest.TestCase):
    def bot_event(self, login: str = "dependabot[bot]", head_repository_id: int = 100) -> dict[str, object]:
        event = owner_event(author_type="Bot", sender_type="Bot", head_repository_id=head_repository_id)
        event["pull_request"]["user"]["login"] = login
        event["sender"]["login"] = login
        return event

    def test_listed_bot_on_a_repository_branch_is_admitted(self) -> None:
        result = run_cli(self.bot_event(), "--automation-logins", "dependabot[bot], renovate[bot]")
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_unlisted_bot_or_fork_follows_the_contract(self) -> None:
        for event, logins in (
            (self.bot_event(login="other[bot]"), "dependabot[bot]"),
            (self.bot_event(head_repository_id=101), "dependabot[bot]"),
            (self.bot_event(), ""),
        ):
            result = run_cli(event, "--automation-logins", logins)
            self.assertEqual(result.returncode, 1, result.stdout)

    def test_listed_login_must_be_a_bot(self) -> None:
        event = owner_event(author_association="NONE")
        event["pull_request"]["user"]["login"] = event["sender"]["login"] = "dependabot[bot]"
        self.assertEqual(run_cli(event, "--automation-logins", "dependabot[bot]").returncode, 1)


class ConfigureRepositoryTests(unittest.TestCase):
    def test_workflows_get_read_only_tokens_and_cannot_approve(self) -> None:
        calls = {endpoint: payload for _, endpoint, payload in baseline("o/r")}
        self.assertEqual(
            calls["repos/o/r/actions/permissions/workflow"],
            {"default_workflow_permissions": "read", "can_approve_pull_request_reviews": False},
        )
        self.assertEqual(
            calls["repos/o/r/actions/permissions/fork-pr-contributor-approval"],
            {"approval_policy": "first_time_contributors"},
        )

    def test_extra_checks_are_required_alongside_the_defaults(self) -> None:
        contexts = protection("solo", ("checks", "check"))["required_status_checks"]["contexts"]
        self.assertEqual(contexts, ["test-validator", "validate-metadata", "check", "DCO", "checks"])

    def test_release_tags_cannot_move_or_be_deleted(self) -> None:
        ruleset = tag_ruleset()
        self.assertEqual((ruleset["target"], ruleset["enforcement"]), ("tag", "active"))
        self.assertEqual(ruleset["conditions"]["ref_name"]["include"], ["refs/tags/v*"])
        self.assertEqual({rule["type"] for rule in ruleset["rules"]}, {"update", "deletion"})

    def test_mode_protection_presets(self) -> None:
        expected_reviews = {
            "solo": (0, False, False, False),
            "contributor": (1, True, True, True),
            "team": (1, True, True, True),
        }
        for mode, reviews in expected_reviews.items():
            with self.subTest(mode=mode):
                settings = protection(mode)
                review_settings = settings["required_pull_request_reviews"]
                self.assertEqual(
                    (
                        review_settings["required_approving_review_count"],
                        review_settings["require_code_owner_reviews"],
                        review_settings["dismiss_stale_reviews"],
                        review_settings["require_last_push_approval"],
                    ),
                    reviews,
                )
                self.assertEqual(
                    settings["required_status_checks"],
                    {"strict": True, "contexts": ["test-validator", "validate-metadata", "check", "DCO"]},
                )
                self.assertEqual(settings["enforce_admins"], mode == "team")
                self.assertFalse(settings["allow_force_pushes"])
                self.assertFalse(settings["allow_deletions"])
                self.assertTrue(settings["required_conversation_resolution"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
