from __future__ import annotations

import os
import subprocess
from pathlib import Path
from typing import Mapping, Optional

from .config import RouterConfig


class CredentialError(Exception):
    pass


def config_directory(
    config: RouterConfig, account: str, environ: Optional[Mapping[str, str]] = None
) -> Path:
    account_config = config.account(account)
    if not account_config or not account_config.config_dir:
        raise CredentialError(
            "Account '{}' needs accounts.{}.config_dir before it can be used".format(
                account, account
            )
        )
    return Path(account_config.config_dir).expanduser()


def token_for(
    config: RouterConfig,
    account: str,
    real_gh: str,
    environ: Optional[Mapping[str, str]] = None,
    hostname: str = "github.com",
) -> str:
    source_environment = dict(environ or os.environ)
    source_environment.pop("GH_TOKEN", None)
    source_environment.pop("GITHUB_TOKEN", None)
    source_environment["GH_CONFIG_DIR"] = str(config_directory(config, account, source_environment))
    result = subprocess.run(
        [real_gh, "auth", "token", "--hostname", hostname],
        env=source_environment,
        capture_output=True,
        text=True,
        check=False,
    )
    token = result.stdout.strip()
    if result.returncode != 0 or not token:
        detail = result.stderr.strip() or "no token was returned"
        raise CredentialError("Unable to load credentials for {}: {}".format(account, detail))
    return token


def environment_for(
    config: RouterConfig,
    account: str,
    token: str,
    environ: Optional[Mapping[str, str]] = None,
) -> dict[str, str]:
    child_environment = dict(environ or os.environ)
    child_environment.pop("GH_TOKEN", None)
    child_environment.pop("GITHUB_TOKEN", None)
    child_environment["GH_CONFIG_DIR"] = str(config_directory(config, account, child_environment))
    child_environment["GH_TOKEN"] = token
    return child_environment
