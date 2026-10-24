#!/usr/bin/env python3
"""Validate the metadata contract in a GitHub pull request event."""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

REQUIRED_SECTIONS = ("Change", "Verification", "Risk")
PLACEHOLDERS = (
    "Closes #",
    "Commands and results:",
    "<commands and concise result; project CI verifies independently>",
    "Check every area this pull request changes:",
    "Details and mitigations:",
)


@dataclass(frozen=True)
class Refusal:
    code: str
    message: str


def _metadata_exempt(event: dict[str, Any], mode: str, solo_maintainer_id: int) -> bool:
    pull_request = event.get("pull_request") or {}
    author = pull_request.get("user") or {}
    base_repository = (pull_request.get("base") or {}).get("repo") or {}
    head_repository = (pull_request.get("head") or {}).get("repo") or {}
    repository = event.get("repository") or {}
    owner = base_repository.get("owner") or {}
    sender = event.get("sender") or {}
    repository_id = repository.get("id")

    trusted_same_repository_author = (
        type(repository_id) is int
        and repository_id > 0
        and owner.get("type") in {"User", "Organization"}
        and author.get("type") == "User"
        and sender.get("type") == "User"
        and type(author.get("id")) is int
        and author.get("id") > 0
        and sender.get("id") == author.get("id")
        and base_repository.get("id") == repository_id
        and head_repository.get("id") == repository_id
    )
    if not trusted_same_repository_author:
        return False

    if mode == "team":
        return pull_request.get("author_association") in {"OWNER", "MEMBER", "COLLABORATOR"}

    if mode != "solo" or author.get("id") != solo_maintainer_id:
        return False

    return owner.get("type") == "Organization" or owner.get("id") == solo_maintainer_id


def _authorized_automation(event: dict[str, Any], logins: frozenset[str]) -> bool:
    """A maintainer-listed bot (HELICOPTER_AUTOMATION_LOGINS) on a branch of this repository."""
    pull_request = event.get("pull_request") or {}
    author = pull_request.get("user") or {}
    sender = event.get("sender") or {}
    repository_id = (event.get("repository") or {}).get("id")
    return (
        author.get("type") == "Bot"
        and author.get("login") in logins
        and sender.get("login") == author.get("login")
        and type(repository_id) is int
        and ((pull_request.get("base") or {}).get("repo") or {}).get("id") == repository_id
        and ((pull_request.get("head") or {}).get("repo") or {}).get("id") == repository_id
    )


def _section(body: str, name: str) -> str | None:
    """Return a section's text without comments and template placeholders, or None if absent."""
    match = re.search(rf"^## {re.escape(name)}[ \t\r]*$(?P<text>.*?)(?=^## |\Z)", body, re.MULTILINE | re.DOTALL)
    if not match:
        return None
    text = re.sub(r"<!--.*?-->", "", match.group("text"), flags=re.DOTALL)
    lines = [line.strip() for line in text.splitlines()]
    return "\n".join(line for line in lines if line and line not in PLACEHOLDERS and not line.startswith("```"))


def validate_body(body: str) -> list[Refusal]:
    refusals: list[Refusal] = []
    sections = {name: _section(body, name) for name in REQUIRED_SECTIONS}

    missing = [f"## {name}" for name, text in sections.items() if text is None]
    if missing:
        refusals.append(
            Refusal(
                "REFUSE_TEMPLATE_INCOMPLETE",
                "Missing required sections: " + ", ".join(missing) + ". Copy them from the pull request template.",
            )
        )

    if sections["Change"] == "":
        refusals.append(
            Refusal("REFUSE_CHANGE_EMPTY", "Under ## Change, describe the problem, the fix, and its visible effect.")
        )
    if sections["Verification"] == "":
        refusals.append(
            Refusal(
                "REFUSE_VERIFICATION_EMPTY",
                "Under ## Verification, list the commands or steps you ran and what they showed.",
            )
        )
    if sections["Risk"] is not None and not re.search(r"^- \[[xX]\] ", sections["Risk"], re.MULTILINE):
        refusals.append(
            Refusal(
                "REFUSE_RISK_UNCHECKED",
                "Under ## Risk, check every area this pull request changes, or check `None of the above`.",
            )
        )

    return refusals


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("event", type=Path, help="GitHub pull request event JSON")
    parser.add_argument(
        "--mode",
        default="contributor",
        help="repository mode: solo, contributor, or team",
    )
    parser.add_argument(
        "--solo-maintainer-id",
        default="",
        help="immutable numeric GitHub user ID authorized in solo mode",
    )
    parser.add_argument(
        "--automation-logins",
        default="",
        help="maintainer-authorized bot logins (HELICOPTER_AUTOMATION_LOGINS), comma or space separated",
    )
    args = parser.parse_args()

    try:
        event = json.loads(args.event.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        print(f"REFUSE_EVENT_INVALID: {error}")
        return 2

    if re.fullmatch(r"[1-9][0-9]*", args.solo_maintainer_id):
        solo_maintainer_id = int(args.solo_maintainer_id)
    else:
        solo_maintainer_id = 0

    mode = args.mode if args.mode in {"solo", "contributor", "team"} else "contributor"
    if _metadata_exempt(event, mode, solo_maintainer_id):
        print(f"{mode.capitalize()}-mode metadata exemption accepted for the trusted maintainer.")
        return 0

    if _authorized_automation(event, frozenset(filter(None, re.split(r"[\s,]+", args.automation_logins)))):
        print("Authorized automation exemption accepted. Human review is still required.")
        return 0

    body = (event.get("pull_request") or {}).get("body") or ""
    refusals = validate_body(body)
    if refusals:
        print("Contribution intake refused:")
        for refusal in refusals:
            print(f"{refusal.code}: {refusal.message}")
        print("\nUpdate the pull request description from the repository template.")
        return 1

    print("Contribution metadata accepted. Human review is still required.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
