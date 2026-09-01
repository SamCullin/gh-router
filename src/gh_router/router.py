from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Optional, Sequence

from .config import RouterConfig
from .target import Target, extract_account_override, resolve_target


@dataclass(frozen=True)
class Resolution:
    account: str
    target: Target
    source: str


def resolve_account(
    arguments: Sequence[str],
    config: RouterConfig,
    environ: Optional[Mapping[str, str]] = None,
    cwd: Optional[Path] = None,
) -> Resolution:
    account_override, forwarded = extract_account_override(arguments)
    target = resolve_target(forwarded, environ=environ, cwd=cwd)

    if account_override:
        return Resolution(config.account_name(account_override), target, "command override")

    if target.repository:
        repository_rule = config.repository_rule(target.repository)
        if repository_rule:
            account, configured_repository = repository_rule
            return Resolution(
                config.account_name(account),
                target,
                "repo:{}".format(configured_repository),
            )

    organisation = target.organisation
    if not organisation and target.repository:
        organisation = target.repository.split("/", 1)[0]

    if organisation:
        organisation_rule = config.organisation_rule(organisation)
        if organisation_rule:
            account, configured_organisation = organisation_rule
            return Resolution(
                config.account_name(account),
                target,
                "org:{}".format(configured_organisation),
            )

        implicit_rule = config.implicit_organisation_rule(organisation)
        if implicit_rule:
            rule, configured_organisation = implicit_rule
            return Resolution(
                config.account_name(rule.account),
                target,
                "implicit-org:{}".format(configured_organisation),
            )

    return Resolution(config.account_name(config.default), target, "default")
