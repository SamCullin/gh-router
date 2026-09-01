from __future__ import annotations

import os
import sys
from typing import Optional, Sequence

from .config import ConfigError, ConfigStore, RouterConfig
from .credentials import CredentialError, environment_for, token_for
from .executable import ExecutableError, find_real_gh
from .router import Resolution, resolve_account
from .target import TargetError, extract_account_override


class CliError(Exception):
    pass


def _parse_auth_options(arguments: Sequence[str]) -> dict[str, str]:
    options: dict[str, str] = {}
    supported = {"--default", "--org", "--repo", "--account", "--config-dir"}
    index = 0
    while index < len(arguments):
        argument = arguments[index]
        if argument in supported:
            if index + 1 >= len(arguments):
                raise CliError("{} requires a value".format(argument))
            options[argument[2:]] = arguments[index + 1]
            index += 2
            continue
        if any(argument.startswith(option + "=") for option in supported):
            name, value = argument[2:].split("=", 1)
            if not value:
                raise CliError("--{} requires a value".format(name))
            options[name] = value
            index += 1
            continue
        raise CliError("Unsupported auth option: {}".format(argument))
    return options


def _load_or_create_for_default(store: ConfigStore, account: str) -> RouterConfig:
    try:
        return store.load()
    except ConfigError as error:
        if not store.path.exists():
            return store.create(account)
        raise error


def _handle_auth_set(store: ConfigStore, arguments: Sequence[str]) -> int:
    options = _parse_auth_options(arguments)
    account = options.get("account")
    default = options.get("default")
    organisation = options.get("org")
    repository = options.get("repo")
    config_dir = options.get("config-dir")

    if default and any(value is not None for value in (organisation, repository, account)):
        raise CliError("--default cannot be combined with --org, --repo, or --account")
    if not default and not (organisation or repository):
        raise CliError("Use --default, --org, or --repo")
    if (organisation or repository) and not account:
        raise CliError("--org and --repo require --account")
    if organisation and repository:
        raise CliError("--org and --repo are mutually exclusive")
    if config_dir and not (default or account):
        raise CliError("--config-dir requires --default, --org, or --repo")

    if default:
        config = _load_or_create_for_default(store, default)
        config.default = default
        if config_dir:
            config.set_account_config(default, config_dir)
        store.save(config)
        print("Default account configured: {}".format(default))
        return 0

    config = store.load()
    if config_dir:
        config.set_account_config(account, config_dir)
    if organisation:
        config.set_organisation_rule(organisation, account)
        print("Organisation configured: {} -> {}".format(organisation, account))
    if repository:
        config.set_repository_rule(repository, account)
        owner = repository.split("/", 1)[0]
        if not config.organisation_rule(owner):
            config.set_implicit_organisation_rule(owner, account, repository)
        print("Repository configured: {} -> {}".format(repository, account))
    store.save(config)
    return 0


def _handle_auth_unset(store: ConfigStore, arguments: Sequence[str]) -> int:
    options = _parse_auth_options(arguments)
    if options.get("default") or options.get("account") or options.get("config-dir"):
        raise CliError("Only --org and --repo can be unset")
    organisation = options.get("org")
    repository = options.get("repo")
    if bool(organisation) == bool(repository):
        raise CliError("Use exactly one of --org or --repo")

    config = store.load()
    if organisation:
        removed = config.remove_organisation_rule(organisation)
        if not removed:
            raise CliError("No organisation rule exists for {}".format(organisation))
        print("Organisation rule removed: {}".format(removed))
    if repository:
        removed = config.remove_repository_rule(repository)
        if not removed:
            raise CliError("No repository rule exists for {}".format(repository))
        config.remove_implicit_source(repository)
        print("Repository rule removed: {}".format(removed))
    store.save(config)
    return 0


def _render_status(config: RouterConfig) -> None:
    print("github.com credentials")
    for account in config.account_names():
        print("  ✓ {}".format(account))
    print()
    print("Routing configuration")
    print("  Default")
    print("    {}".format(config.default))
    if config.orgs:
        print("  Organisations")
        for organisation, account in config.orgs.items():
            print("    {}        → {}".format(organisation, account))
    if config.repos:
        print("  Repositories")
        for repository, account in config.repos.items():
            print("    {}   → {}".format(repository, account))
    if config.implicit_orgs:
        print("  Implicit organisations")
        for organisation, rule in config.implicit_orgs.items():
            print("    {} → {}".format(organisation, rule.account))
            print("      established by {}".format(rule.source))


def _target_name(resolution: Resolution) -> str:
    if resolution.target.repository:
        return resolution.target.repository
    if resolution.target.organisation:
        return "organisation:{}".format(resolution.target.organisation)
    return "<none>"


def _handle_auth_status(
    store: ConfigStore,
    arguments: Sequence[str],
    account_override: Optional[str],
) -> int:
    config = store.load()
    if "--resolve" not in arguments:
        _render_status(config)
        return 0

    resolve_arguments = [argument for argument in arguments if argument != "--resolve"]
    resolution_arguments = list(resolve_arguments)
    if account_override:
        resolution_arguments.extend(["--account", account_override])
    resolution = resolve_account(resolution_arguments, config)
    print("repository: {}".format(_target_name(resolution)))
    print("account:    {}".format(resolution.account))
    print("source:     {}".format(resolution.source))
    return 0


def _exec_real_gh(arguments: Sequence[str], config: RouterConfig, resolution: Resolution) -> int:
    real_gh = find_real_gh(sys.argv[0])
    token = token_for(config, resolution.account, real_gh)
    child_environment = environment_for(config, resolution.account, token)
    os.execvpe(real_gh, [real_gh, *arguments], child_environment)
    return 127


def _is_passthrough_without_auth(arguments: Sequence[str]) -> bool:
    return not arguments or arguments in (["--help"], ["help"], ["--version"])


def main(arguments: Optional[Sequence[str]] = None) -> int:
    raw_arguments = list(arguments if arguments is not None else sys.argv[1:])
    store = ConfigStore()

    if raw_arguments and raw_arguments[0] == "auth":
        command_arguments = raw_arguments
        account_override = None
    else:
        account_override, command_arguments = extract_account_override(raw_arguments)

    if command_arguments[:2] == ["auth", "switch"]:
        print("gh-router: account switching is automatic; no action required.")
        print()
        print("Account selection is determined by repository and organisation configuration.")
        print()
        print("To override the account for a command:")
        print("  gh --account SamCullin <command>")
        return 0

    if command_arguments[:2] == ["auth", "set"]:
        return _handle_auth_set(store, command_arguments[2:])

    if command_arguments[:2] == ["auth", "unset"]:
        return _handle_auth_unset(store, command_arguments[2:])

    if command_arguments[:2] == ["auth", "status"]:
        return _handle_auth_status(store, command_arguments[2:], account_override)

    if command_arguments[:2] == ["auth", "resolve"]:
        return _handle_auth_status(store, ["--resolve", *command_arguments[2:]], account_override)

    if _is_passthrough_without_auth(command_arguments):
        real_gh = find_real_gh(sys.argv[0])
        os.execv(real_gh, [real_gh, *command_arguments])
        return 127

    config = store.load()
    resolution = resolve_account(
        [*command_arguments, *(["--account", account_override] if account_override else [])],
        config,
    )
    return _exec_real_gh(command_arguments, config, resolution)


def run() -> int:
    try:
        return main()
    except (CliError, ConfigError, CredentialError, ExecutableError, TargetError) as error:
        print("gh-router: {}".format(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(run())
