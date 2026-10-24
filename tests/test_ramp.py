#!/usr/bin/env python3

import sys
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from configure_repository import QUEUE_LABELS as PROVISIONED
from pr_queue import LABELS as QUEUE_LABELS, SUPERSEDED, linked_issues, merged_rival, notice, plan, queued, turn

NOW = datetime(2026, 10, 30, tzinfo=timezone.utc)


def at(day: int) -> str:
    return (NOW - timedelta(days=30 - day)).isoformat().replace("+00:00", "Z")


PR = {"user": {"login": "alice"}, "created_at": at(0), "draft": False}


def comment(day, login, association="NONE", body="", kind="User"):
    return {"event": "commented", "created_at": at(day), "actor": {"login": login, "type": kind},
            "author_association": association, "body": body}


def review(day, login, state, association="COLLABORATOR"):
    return {"event": "reviewed", "submitted_at": at(day), "user": {"login": login, "type": "User"},
            "author_association": association, "state": state}


def commit(day):
    return {"event": "committed", "committer": {"date": at(day)}}


class TurnTests(unittest.TestCase):
    def assertTurn(self, events, whose, day, **kwargs):
        current = turn(PR, events, NOW, **kwargs)
        self.assertEqual((current.whose, current.since), (whose, datetime.fromisoformat(at(day).replace("Z", "+00:00"))))

    def test_new_pull_request_waits_on_reviewer_from_opening(self):
        self.assertTurn([], "reviewer", 0)

    def test_changes_requested_hands_turn_to_author_until_push(self):
        self.assertTurn([review(3, "max", "changes_requested")], "author", 3)
        self.assertTurn([review(3, "max", "changes_requested"), commit(5)], "reviewer", 5)

    def test_author_reply_returns_turn_but_author_ping_does_not_reset_clock(self):
        self.assertTurn([review(3, "max", "changes_requested"), comment(6, "alice")], "reviewer", 6)
        self.assertTurn([comment(4, "alice"), commit(9)], "reviewer", 0)

    def test_author_re_request_hands_back_but_maintainer_request_does_not(self):
        def requested(day, login):
            return {"event": "review_requested", "created_at": at(day), "actor": {"login": login, "type": "User"}}
        asked = review(3, "max", "changes_requested")
        self.assertTurn([asked, requested(6, "alice")], "reviewer", 6)
        self.assertTurn([asked, requested(6, "max")], "author", 3)

    def test_maintainer_response_resets_clock_but_does_not_end_author_turn(self):
        self.assertTurn([comment(4, "max", "MEMBER")], "reviewer", 4)
        self.assertTurn([review(3, "max", "changes_requested"), comment(4, "max", "OWNER")], "author", 3)
        self.assertTurn([review(3, "max", "changes_requested"), review(4, "max", "approved")], "reviewer", 4)

    def test_review_bot_comments_do_not_move_the_turn(self):
        events = [review(3, "max", "changes_requested"),
                  comment(4, "github-actions[bot]", "NONE", body="PR Reviewer Guide", kind="Bot")]
        self.assertTurn(events, "author", 3)

    def test_bots_others_and_listed_automation_do_not_count_as_maintainers(self):
        events = [comment(2, "dependabot[bot]", "COLLABORATOR", kind="Bot"), comment(3, "bob"),
                  comment(4, "roboomp", "COLLABORATOR")]
        self.assertTurn(events, "reviewer", 0, ignored=frozenset({"roboomp"}))

    def test_refused_admission_is_the_authors_turn(self):
        self.assertEqual(turn(PR, [], NOW, admitted=False).whose, "author")

    def test_drafts_are_not_queued_and_ready_restarts_clock(self):
        self.assertIsNone(turn({**PR, "draft": True}, [], NOW))
        self.assertTurn([{"event": "ready_for_review", "created_at": at(8)}], "reviewer", 8)


class PlanTests(unittest.TestCase):
    def since(self, hours):
        return turn(PR, [{"event": "ready_for_review", "created_at": (NOW - timedelta(hours=hours)).isoformat()}], NOW)

    def test_thresholds_in_hours(self):
        self.assertEqual(plan(self.since(3), NOW, 4, 72), (QUEUE_LABELS["reviewer"], None))
        self.assertEqual(plan(self.since(4), NOW, 4, 72), (QUEUE_LABELS["reviewer"], "nudge"))
        self.assertEqual(plan(self.since(72), NOW, 4, 72), (QUEUE_LABELS["paused"], "pause"))

    def test_default_thresholds_keep_seven_and_twenty_one_days(self):
        self.assertEqual(plan(self.since(7 * 24 - 1), NOW, 168, 504), (QUEUE_LABELS["reviewer"], None))
        self.assertEqual(plan(self.since(7 * 24), NOW, 168, 504), (QUEUE_LABELS["reviewer"], "nudge"))
        self.assertEqual(plan(self.since(21 * 24), NOW, 168, 504), (QUEUE_LABELS["paused"], "pause"))

    def test_author_turn_never_nudges_or_pauses(self):
        current = turn(PR, [review(0, "max", "changes_requested")], NOW)
        self.assertEqual(plan(current, NOW, 4, 72), (QUEUE_LABELS["author"], None))

    def test_notice_reports_hours_then_days(self):
        self.assertIn("waited 5 hours", notice("nudge", self.since(5), NOW, "alice", ["max"]))
        self.assertIn("waited 3 days", notice("nudge", self.since(72), NOW, "alice", ["max"]))


class QueueScopeTests(unittest.TestCase):
    def test_team_mode_queues_members(self):
        for association in ("OWNER", "MEMBER", "COLLABORATOR"):
            pr = {**PR, "state": "open", "author_association": association}
            self.assertTrue(queued(pr, team=True), association)
            self.assertFalse(queued(pr, team=False), association)

    def test_outside_contributions_are_queued_in_every_mode(self):
        pr = {**PR, "state": "open", "author_association": "CONTRIBUTOR"}
        self.assertTrue(queued(pr, team=False))
        self.assertTrue(queued(pr, team=True))

    def test_closed_pull_requests_are_never_queued(self):
        self.assertFalse(queued({**PR, "state": "closed", "author_association": "MEMBER"}, team=True))


def cross_reference(number, merged_day=None, is_pr=True):
    source = {"number": number}
    if is_pr:
        source["pull_request"] = {"merged_at": at(merged_day) if merged_day is not None else None}
    return {"event": "cross-referenced", "source": {"issue": source}}


class SupersededTests(unittest.TestCase):
    pr = {**PR, "number": 50, "title": "fix: thing", "body": "Closes #7. Fixes #12. Step #1 of the plan, see #3."}

    def test_linked_issues_from_cross_references_and_linking_keywords(self):
        known, mentioned = linked_issues(self.pr, [cross_reference(12, is_pr=False), cross_reference(60)])
        self.assertEqual(known, {12})
        self.assertEqual(mentioned, {7})  # #12 already known; bare #1 and #3 are prose

    def test_rival_merged_while_this_pr_waited(self):
        found = merged_rival(self.pr, {12: [cross_reference(61, merged_day=3), cross_reference(62, merged_day=1)]})
        self.assertEqual(found[:2], (62, 12))

    def test_rival_merged_before_this_pr_opened_is_ignored(self):
        opened_later = {**self.pr, "created_at": at(5)}
        self.assertIsNone(merged_rival(opened_later, {12: [cross_reference(61, merged_day=3)]}))

    def test_unmerged_rivals_and_this_pr_are_ignored(self):
        self.assertIsNone(merged_rival(self.pr, {12: [cross_reference(61), cross_reference(50, merged_day=2)]}))


class LabelTests(unittest.TestCase):
    def test_configurator_provisions_every_runtime_label(self):
        self.assertLessEqual(set(QUEUE_LABELS.values()) | {SUPERSEDED}, set(PROVISIONED))


if __name__ == "__main__":
    unittest.main(verbosity=2)
