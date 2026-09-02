[![Build & Release](https://github.com/SamCullin/gh-router/actions/workflows/release.yml/badge.svg)](https://github.com/SamCullin/gh-router/actions/workflows/release.yml)
[![CI](https://github.com/SamCullin/gh-router/actions/workflows/ci.yml/badge.svg)](https://github.com/SamCullin/gh-router/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/SamCullin/gh-router)](https://github.com/SamCullin/gh-router/blob/main/go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/SamCullin/gh-router)](https://github.com/SamCullin/gh-router/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/SamCullin/gh-router/graphs/commit-activity)
[![Downloads](https://img.shields.io/github/downloads/SamCullin/gh-router/total.svg)](https://github.com/SamCullin/gh-router/releases)
[![GitHub issues](https://img.shields.io/github/issues/SamCullin/gh-router.svg)](https://github.com/SamCullin/gh-router/issues)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://makeapullrequest.com)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)
[![GoReleaser](https://img.shields.io/badge/release-GoReleaser-00ADD8?logo=go&logoColor=white)](https://goreleaser.com/)

# gh-router

<details>
<summary>Copy this prompt to give an agent</summary>

```text
Install and configure gh-router for me.

Use Homebrew to install it with `brew install --cask SamCullin/tap/gh-router`.
Confirm that `gh-router --version`, `ghr --version`, `gh router llm-text`, and
`ghrllm.text` work.

Ask me which local account labels I need. For each account, run
`gh router auth setup <account>`. This command creates the isolated GitHub CLI
configuration directory, starts the native `gh auth login` flow, and records
the account in `~/.config/gh-router/config.yaml`.

If I want to reuse an existing GitHub CLI configuration, pass it with
`--config-dir`, for example:
`gh router auth setup SamCullin --config-dir ~/.config/gh`.

Do not ask me for a token or print credential files. The first configured
account becomes the default. Confirm the result with `gh router auth status`.

Ask which organisations or repositories should use each account, then add
rules with `gh router auth set --org` or `gh router auth set --repo`.
Resolve a read-only test target with
`gh router auth status --resolve -R OWNER/REPOSITORY` before testing a command.
Use `gh <command>` for normal GitHub CLI work when `gh` points to the router,
or use `ghr <command>` for an explicit routed invocation.
Do not run destructive GitHub commands without my explicit instruction.
```
</details>

[!['Buy Me A Coffee'](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/samcullin)

## Why gh-router exists

The GitHub CLI keeps account selection in mutable shared state. Running `gh auth switch` changes which account the next command uses. That is manageable for one terminal session, but it becomes a race when several agents run at the same time.

For example, Agent A can select a work account while Agent B expects a personal account. The result depends on timing rather than the repository being operated on. `gh-router` resolves an account for each invocation from the repository, organisation, explicit override, and default rules, then runs the native GitHub CLI with that account's isolated credentials.

The normal command surface remains the GitHub CLI. Router behaviour is added as a
namespace, like a GitHub CLI extension:

- `gh --help`, command-specific help, and `gh auth ...` use the native GitHub CLI.
- `gh router ...` exposes router help, account setup, status, and routing rules.
- `gh auth switch` is the one native command intercepted by the router, because
  changing mutable active-account state defeats deterministic routing.
- All other GitHub CLI operations are forwarded to the native CLI after routing.

The direct commands `gh-router` and `ghr` expose the same router and routed
operation surface when an explicit router binary is preferable.

## Caveats

- `gh-router` delegates GitHub operations to a native `gh` installation and does not issue or store tokens itself.
- Routing currently targets `github.com`; each account must have its own authenticated GitHub CLI configuration directory.
- Commands without a repository or organisation target use the configured default account. Use `--account` when the choice must be explicit.

## Installation

### Homebrew Cask

Tagged releases publish checksum-verified archives for macOS and Linux and update the cask in [`SamCullin/homebrew-tap`](https://github.com/SamCullin/homebrew-tap):

```bash
brew install --cask SamCullin/tap/gh-router
```

The cask installs three commands:

- `gh-router`, the full command name
- `ghr`, the shorthand for `gh-router`
- `ghrllm.text`, a prompt for configuring the tool with an LLM

No environment variable or shell alias is required for the normal installation.
Homebrew leaves the native `gh` command intact. To route `gh` by default, point
your shell configuration at `gh-router` and keep the native executable
reachable, for example as `gho`; otherwise use `gh-router` or `ghr` explicitly.
When the router is invoked as `gh`, native help and `gh auth ...` still pass
through unchanged, while ordinary GitHub operations receive account routing.
The same prompt is available from the router namespace with `gh router llm-text`.

Upgrade later with:

```bash
brew update
brew upgrade --cask gh-router
```

If you installed `gh-router` as a Formula before the cask migration, migrate it once:

```bash
brew uninstall gh-router
brew install --cask SamCullin/tap/gh-router
```

### From source

For local development:

```bash
make test
make vet
make build
```

## Setup

After installation, set up each GitHub account through the router namespace:

```bash
gh router auth setup SamCullin
gh router auth setup SamWork
```

The direct forms `gh-router auth setup SamCullin` and `ghr auth setup SamWork`
are equivalent when calling the router binary directly.

For every account, `auth setup`:

1. Creates `~/.config/gh-router/accounts/<account>` with private permissions.
2. Starts the native `gh auth login` flow inside that directory.
3. Records the account and its configuration path in `~/.config/gh-router/config.yaml`.

The first account becomes the default. The command accepts native login options such as `--web`. `gh router auth login` is also accepted as an alias for `auth setup`.

To reuse an existing native GitHub CLI configuration, pass its path explicitly:

```bash
gh router auth setup SamCullin --config-dir ~/.config/gh
```

There is no need to run `mkdir`, set `GH_CONFIG_DIR`, or switch the active GitHub CLI account by hand.

## Configuration

The router stores its routing file at `~/.config/gh-router/config.yaml`. Account directories contain the native GitHub CLI credentials and stay separate from this file. The routing file contains account names, paths, and rules, but not tokens.

After setup, the file will look like this:

```yaml
default: SamCullin

accounts:
  SamCullin:
    config_dir: ~/.config/gh-router/accounts/SamCullin
  SamWork:
    config_dir: ~/.config/gh-router/accounts/SamWork

orgs:
  OpenAI:
    account: SamWork

repos:
  OpenAI/sensitive-repo:
    account: SamCullin
```

The file is ordinary YAML and can be tracked in a dotfiles repository. Keep the account credential directories out of version control. To use a different configuration location, set `GH_ROUTER_CONFIG`; the default installation does not require any environment variables.

Use the command helpers to manage rules without editing YAML:

```bash
gh router auth set --default SamCullin
gh router auth set --org OpenAI --account SamWork
gh router auth set --repo OpenAI/sensitive-repo --account SamCullin
gh router auth unset --repo OpenAI/sensitive-repo
```

The first repository rule for an organisation establishes its implicit organisation account. Later repository rules for that organisation remain exceptions and do not silently change the fallback account.

## Use

Use the normal GitHub CLI surface for native and routed work:

```bash
gh --help
gh auth status
gh pr list
gh pr list -R OpenAI/sensitive-repo
gh --account SamCullin pr create
gh router auth status
gh router auth status --resolve -R OpenAI/sensitive-repo
gh router llm-text
```

If `gh` is still the unwrapped native binary, use `gh-router` or `ghr` for
routed operations instead:

```bash
ghr pr list -R OpenAI/sensitive-repo
ghr --account SamCullin pr create
ghr auth status --resolve -R OpenAI/sensitive-repo
```

For an LLM-oriented setup and usage prompt, run either:

```bash
gh router llm-text
ghrllm.text
```

`gh auth switch` is intentionally a successful no-op when `gh` points to the
router, and explains that account selection is automatic. Use `gh router auth
status` to inspect routing configuration; it never reports a mutable active
account.

## How routing works

Every command resolves an account in this order:

```text
explicit --account override
repository rule
organisation rule
implicit organisation rule
default account
```

The target repository is resolved in this order:

```text
-R / --repo or explicit OWNER/REPO argument
gh api repos/OWNER/REPO/...
GH_REPO
current repository remote
--org / --owner
default account
```

`GH_TOKEN` and `GITHUB_TOKEN` are removed from the child environment before the resolved account token is injected. This prevents an inherited shell token from bypassing routing.

## Contributing

Bug reports, feature requests, and pull requests are welcome. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for development checks and contribution guidelines.

## License

Released under the [MIT License](LICENSE).
