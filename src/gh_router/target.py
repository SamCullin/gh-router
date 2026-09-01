from __future__ import annotations

import os
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Optional, Sequence, Tuple
from urllib.parse import urlsplit


class TargetError(Exception):
    pass


_OWNER_RE = re.compile(r"^[A-Za-z0-9_.-]+$")
_REPO_RE = re.compile(r"^[A-Za-z0-9_.-]+$")


@dataclass(frozen=True)
class Target:
    repository: Optional[str] = None
    organisation: Optional[str] = None
    source: str = "default"


def _parse_repo_reference(value: str) -> Optional[str]:
    candidate = value.strip()
    if candidate.startswith("repos/"):
        candidate = candidate[6:]
    elif candidate.startswith("/repos/"):
        candidate = candidate[7:]
    elif candidate.startswith("git@github.com:"):
        candidate = candidate[len("git@github.com:") :]
    elif candidate.startswith("ssh://git@github.com/"):
        candidate = candidate[len("ssh://git@github.com/") :]
    elif candidate.startswith("https://") or candidate.startswith("http://"):
        parsed = urlsplit(candidate)
        if parsed.hostname != "github.com":
            return None
        candidate = parsed.path.lstrip("/")
    elif candidate.startswith("github.com/"):
        candidate = candidate[len("github.com/") :]

    parts = candidate.strip("/").split("/")
    if len(parts) < 2:
        return None
    owner, repository = parts[0], parts[1]
    repository = repository.removesuffix(".git")
    if not _OWNER_RE.fullmatch(owner) or not _REPO_RE.fullmatch(repository):
        return None
    return "{}/{}".format(owner, repository)


def _option_value(arguments: Sequence[str], options: Tuple[str, ...]) -> Optional[str]:
    for index, argument in enumerate(arguments):
        for option in options:
            prefix = option + "="
            if argument.startswith(prefix):
                return argument[len(prefix) :]
            if argument == option and index + 1 < len(arguments):
                return arguments[index + 1]
    return None


def _explicit_repository(arguments: Sequence[str]) -> Optional[str]:
    direct = _option_value(arguments, ("-R", "--repo"))
    if direct:
        return _parse_repo_reference(direct)

    for argument in arguments:
        if argument.startswith("-R") and argument != "-R":
            parsed = _parse_repo_reference(argument[2:])
            if parsed:
                return parsed

    for index, argument in enumerate(arguments):
        if argument in ("-R", "--repo"):
            continue
        if argument.startswith("-"):
            continue
        if index and arguments[index - 1] in ("-R", "--repo"):
            continue
        if argument.startswith("repos/"):
            parsed = _parse_repo_reference(argument)
            if parsed:
                return parsed
        if argument.count("/") == 1:
            parsed = _parse_repo_reference(argument)
            if parsed:
                return parsed
    return None


def _api_repository(arguments: Sequence[str]) -> Optional[str]:
    for argument in arguments:
        if argument.startswith("repos/") or argument.startswith("/repos/"):
            parsed = _parse_repo_reference(argument)
            if parsed:
                return parsed
    return None


def _remote_repository(cwd: Path) -> Optional[str]:
    result = subprocess.run(
        ["git", "-C", str(cwd), "remote", "get-url", "origin"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        return None
    return _parse_repo_reference(result.stdout.strip())


def resolve_target(
    arguments: Sequence[str],
    environ: Optional[Mapping[str, str]] = None,
    cwd: Optional[Path] = None,
) -> Target:
    environment = environ or os.environ
    command_repository = _explicit_repository(arguments)
    if command_repository:
        return Target(repository=command_repository, source="command")

    api_repository = _api_repository(arguments)
    if api_repository:
        return Target(repository=api_repository, source="command")

    environment_repository = environment.get("GH_REPO")
    if environment_repository:
        parsed = _parse_repo_reference(environment_repository)
        if not parsed:
            raise TargetError("GH_REPO must use OWNER/REPO form")
        return Target(repository=parsed, source="environment")

    repository = _remote_repository(cwd or Path.cwd())
    if repository:
        return Target(repository=repository, source="repository")

    organisation = _option_value(arguments, ("--org", "--owner"))
    if organisation:
        return Target(organisation=organisation, source="organisation")

    return Target()


def extract_account_override(arguments: Sequence[str]) -> Tuple[Optional[str], list[str]]:
    override: Optional[str] = None
    forwarded: list[str] = []
    index = 0
    while index < len(arguments):
        argument = arguments[index]
        if argument == "--account":
            if index + 1 >= len(arguments):
                raise TargetError("--account requires an account name")
            override = arguments[index + 1]
            index += 2
            continue
        if argument.startswith("--account="):
            override = argument.split("=", 1)[1]
            if not override:
                raise TargetError("--account requires an account name")
            index += 1
            continue
        forwarded.append(argument)
        index += 1
    return override, forwarded
