import os
import unittest
from pathlib import Path
from unittest.mock import patch

from gh_router.config import RouterConfig
from gh_router.credentials import environment_for
from gh_router.router import resolve_account
from gh_router.target import extract_account_override, resolve_target


class RouterTests(unittest.TestCase):
    def setUp(self):
        self.config = RouterConfig.from_mapping(
            {
                "default": "SamCullin",
                "accounts": {
                    "SamCullin": {"config_dir": "~/.config/gh"},
                    "SamWork": {"config_dir": "~/.config/gh-work"},
                    "SamAdmin": {"config_dir": "~/.config/gh-admin"},
                },
                "orgs": {"OpenAI": {"account": "SamWork"}},
                "repos": {"OpenAI/sensitive-repo": {"account": "SamAdmin"}},
                "implicit_orgs": {
                    "PersonalOrg": {
                        "account": "SamCullin",
                        "source": "PersonalOrg/project",
                    }
                },
            }
        )

    def test_precedence_is_repository_then_org_then_implicit_then_default(self):
        repository = resolve_account(["-R", "OpenAI/sensitive-repo", "pr", "list"], self.config)
        organisation = resolve_account(["-R", "OpenAI/other", "pr", "list"], self.config)
        implicit = resolve_account(["-R", "PersonalOrg/other", "pr", "list"], self.config)
        default = resolve_account(["-R", "OtherOrg/other", "pr", "list"], self.config)

        self.assertEqual(repository.account, "SamAdmin")
        self.assertEqual(organisation.account, "SamWork")
        self.assertEqual(implicit.account, "SamCullin")
        self.assertEqual(default.account, "SamCullin")

    def test_explicit_account_override_wins(self):
        resolution = resolve_account(
            ["--account", "SamCullin", "-R", "OpenAI/sensitive-repo", "pr", "list"],
            self.config,
        )
        self.assertEqual(resolution.account, "SamCullin")
        self.assertEqual(resolution.source, "command override")

    def test_target_resolution_uses_explicit_repo_before_environment(self):
        target = resolve_target(
            ["-R", "OtherOrg/bar", "pr", "list"],
            environ={"GH_REPO": "OpenAI/foo"},
            cwd=Path("/does/not/exist"),
        )
        self.assertEqual(target.repository, "OtherOrg/bar")
        self.assertEqual(target.source, "command")

    def test_api_target_is_resolved(self):
        target = resolve_target(
            ["api", "repos/OtherOrg/bar/issues"],
            environ={},
            cwd=Path("/does/not/exist"),
        )
        self.assertEqual(target.repository, "OtherOrg/bar")

    def test_account_override_is_removed_from_forwarded_arguments(self):
        override, forwarded = extract_account_override(
            ["--account=SamWork", "pr", "list"]
        )
        self.assertEqual(override, "SamWork")
        self.assertEqual(forwarded, ["pr", "list"])

    def test_first_repository_rule_establishes_implicit_organisation(self):
        config = RouterConfig.from_mapping({"default": "SamCullin"})
        config.set_repository_rule("PersonalOrg/project", "SamCullin")
        config.set_implicit_organisation_rule("PersonalOrg", "SamCullin", "PersonalOrg/project")
        config.set_repository_rule("PersonalOrg/admin", "SamAdmin")
        config.set_implicit_organisation_rule("PersonalOrg", "SamAdmin", "PersonalOrg/admin")

        rule = config.implicit_organisation_rule("PersonalOrg")
        self.assertIsNotNone(rule)
        self.assertEqual(rule[0].account, "SamCullin")
        self.assertEqual(rule[0].source, "PersonalOrg/project")

    def test_inherited_credentials_are_replaced(self):
        with patch.dict(
            os.environ,
            {"GH_TOKEN": "wrong", "GITHUB_TOKEN": "also-wrong", "PATH": "/bin"},
            clear=True,
        ):
            child_environment = environment_for(
                self.config, "SamCullin", "resolved-token", os.environ
            )

        self.assertEqual(child_environment["GH_TOKEN"], "resolved-token")
        self.assertNotIn("GITHUB_TOKEN", child_environment)
        self.assertEqual(child_environment["GH_CONFIG_DIR"], str(Path("~/.config/gh").expanduser()))


if __name__ == "__main__":
    unittest.main()
