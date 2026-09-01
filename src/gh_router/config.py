from __future__ import annotations

import os
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Mapping, Optional, Tuple

import yaml


class ConfigError(Exception):
    pass


def normalise_org(value: str) -> str:
    return value.strip().lower()


def normalise_repo(value: str) -> str:
    parts = value.strip().split("/", 1)
    if len(parts) != 2 or not parts[0] or not parts[1]:
        raise ConfigError("Repository must use OWNER/REPO form")
    return "{}/{}".format(parts[0].lower(), parts[1].lower())


def _require_mapping(value: Any, name: str) -> Mapping[str, Any]:
    if value is None:
        return {}
    if not isinstance(value, Mapping):
        raise ConfigError("{} must be a mapping".format(name))
    return value


def _require_account(value: Any, name: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ConfigError("{} must name an account".format(name))
    return value.strip()


@dataclass
class Account:
    name: str
    config_dir: Optional[str] = None


@dataclass
class ImplicitOrg:
    account: str
    source: str


@dataclass
class RouterConfig:
    default: str
    accounts: Dict[str, Account]
    orgs: Dict[str, str]
    repos: Dict[str, str]
    implicit_orgs: Dict[str, ImplicitOrg]

    @classmethod
    def from_mapping(cls, raw: Mapping[str, Any]) -> "RouterConfig":
        if not isinstance(raw, Mapping):
            raise ConfigError("Configuration must be a mapping")

        default = _require_account(raw.get("default"), "default")
        accounts: Dict[str, Account] = {}
        for name, value in _require_mapping(raw.get("accounts"), "accounts").items():
            account_name = _require_account(name, "account name")
            if isinstance(value, str):
                config_dir = value.strip() or None
            elif value is None:
                config_dir = None
            elif isinstance(value, Mapping):
                configured_dir = value.get("config_dir", value.get("gh_config_dir"))
                if configured_dir is not None and not isinstance(configured_dir, str):
                    raise ConfigError("accounts.{}.config_dir must be a path".format(account_name))
                config_dir = configured_dir.strip() if configured_dir else None
            else:
                raise ConfigError("accounts.{} must be a mapping or path".format(account_name))
            accounts[account_name] = Account(account_name, config_dir)

        orgs: Dict[str, str] = {}
        for name, value in _require_mapping(raw.get("orgs"), "orgs").items():
            org_name = _require_account(name, "organisation name")
            account = value.get("account") if isinstance(value, Mapping) else value
            orgs[org_name] = _require_account(account, "orgs.{}.account".format(org_name))

        repos: Dict[str, str] = {}
        for name, value in _require_mapping(raw.get("repos"), "repos").items():
            repo_name = _require_account(name, "repository name")
            normalise_repo(repo_name)
            account = value.get("account") if isinstance(value, Mapping) else value
            repos[repo_name] = _require_account(account, "repos.{}.account".format(repo_name))

        implicit_orgs: Dict[str, ImplicitOrg] = {}
        for name, value in _require_mapping(raw.get("implicit_orgs"), "implicit_orgs").items():
            org_name = _require_account(name, "implicit organisation name")
            if not isinstance(value, Mapping):
                raise ConfigError("implicit_orgs.{} must be a mapping".format(org_name))
            account = _require_account(
                value.get("account"), "implicit_orgs.{}.account".format(org_name)
            )
            source = _require_account(
                value.get("source"), "implicit_orgs.{}.source".format(org_name)
            )
            normalise_repo(source)
            implicit_orgs[org_name] = ImplicitOrg(account, source)

        return cls(default, accounts, orgs, repos, implicit_orgs)

    def to_mapping(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"default": self.default}
        if self.accounts:
            result["accounts"] = {
                name: {"config_dir": account.config_dir}
                if account.config_dir is not None
                else {}
                for name, account in self.accounts.items()
            }
        if self.orgs:
            result["orgs"] = {name: {"account": account} for name, account in self.orgs.items()}
        if self.repos:
            result["repos"] = {name: {"account": account} for name, account in self.repos.items()}
        if self.implicit_orgs:
            result["implicit_orgs"] = {
                name: {"account": value.account, "source": value.source}
                for name, value in self.implicit_orgs.items()
            }
        return result

    def account_name(self, requested: str) -> str:
        for name in self.account_names():
            if name.lower() == requested.lower():
                return name
        return requested

    def account_names(self) -> list[str]:
        names = list(self.accounts)
        for account in [self.default, *self.orgs.values(), *self.repos.values()]:
            if account not in names:
                names.append(account)
        for value in self.implicit_orgs.values():
            if value.account not in names:
                names.append(value.account)
        return names

    def account(self, name: str) -> Optional[Account]:
        canonical = self.account_name(name)
        return self.accounts.get(canonical)

    def set_account_config(self, name: str, config_dir: Optional[str]) -> None:
        canonical = self.account_name(name)
        self.accounts[canonical] = Account(canonical, config_dir)

    def repository_rule(self, repo: str) -> Optional[Tuple[str, str]]:
        wanted = normalise_repo(repo)
        for configured_repo, account in self.repos.items():
            if normalise_repo(configured_repo) == wanted:
                return account, configured_repo
        return None

    def organisation_rule(self, organisation: str) -> Optional[Tuple[str, str]]:
        wanted = normalise_org(organisation)
        for configured_org, account in self.orgs.items():
            if normalise_org(configured_org) == wanted:
                return account, configured_org
        return None

    def implicit_organisation_rule(self, organisation: str) -> Optional[Tuple[ImplicitOrg, str]]:
        wanted = normalise_org(organisation)
        for configured_org, rule in self.implicit_orgs.items():
            if normalise_org(configured_org) == wanted:
                return rule, configured_org
        return None

    def set_repository_rule(self, repo: str, account: str) -> None:
        existing = self.repository_rule(repo)
        key = existing[1] if existing else repo
        self.repos[key] = account

    def remove_repository_rule(self, repo: str) -> Optional[str]:
        existing = self.repository_rule(repo)
        if not existing:
            return None
        del self.repos[existing[1]]
        return existing[1]

    def set_organisation_rule(self, organisation: str, account: str) -> None:
        existing = self.organisation_rule(organisation)
        key = existing[1] if existing else organisation
        self.orgs[key] = account

    def remove_organisation_rule(self, organisation: str) -> Optional[str]:
        existing = self.organisation_rule(organisation)
        if not existing:
            return None
        del self.orgs[existing[1]]
        return existing[1]

    def set_implicit_organisation_rule(self, organisation: str, account: str, source: str) -> None:
        if self.implicit_organisation_rule(organisation):
            return
        self.implicit_orgs[organisation] = ImplicitOrg(account, source)

    def remove_implicit_source(self, repo: str) -> None:
        wanted = normalise_repo(repo)
        for organisation, rule in list(self.implicit_orgs.items()):
            if normalise_repo(rule.source) == wanted:
                del self.implicit_orgs[organisation]


class ConfigStore:
    def __init__(self, path: Optional[Path] = None):
        configured_path = path or Path(
            os.environ.get("GH_ROUTER_CONFIG", "~/.config/gh-router/config.yaml")
        )
        self.path = configured_path.expanduser()

    def load(self) -> RouterConfig:
        if not self.path.exists():
            raise ConfigError(
                "Configuration file not found at {}. Copy config.example.yaml or run gh auth set --default ACCOUNT.".format(
                    self.path
                )
            )
        try:
            with self.path.open("r", encoding="utf-8") as stream:
                raw = yaml.safe_load(stream) or {}
        except OSError as error:
            raise ConfigError("Unable to read {}: {}".format(self.path, error)) from error
        except yaml.YAMLError as error:
            raise ConfigError("Unable to parse {}: {}".format(self.path, error)) from error
        return RouterConfig.from_mapping(raw)

    def save(self, config: RouterConfig) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        file_descriptor, temporary_name = tempfile.mkstemp(
            dir=str(self.path.parent), prefix=".config-", text=True
        )
        temporary_path = Path(temporary_name)
        try:
            with os.fdopen(file_descriptor, "w", encoding="utf-8") as stream:
                yaml.safe_dump(config.to_mapping(), stream, sort_keys=False, default_flow_style=False)
            temporary_path.chmod(0o600)
            temporary_path.replace(self.path)
        except OSError as error:
            temporary_path.unlink(missing_ok=True)
            raise ConfigError("Unable to write {}: {}".format(self.path, error)) from error

    def create(self, default: str) -> RouterConfig:
        return RouterConfig(default, {}, {}, {}, {})
