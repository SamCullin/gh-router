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

[!['Buy Me A Coffee'](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/samcullin)

`gh-router` is a deterministic wrapper for the GitHub CLI. It selects credentials from the repository and organisation being acted on, rather than relying on GitHub CLI's mutable active-account state.

## Resolution

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

## Configuration

Copy [`config.example.yaml`](config.example.yaml) to `~/.config/gh-router/config.yaml`, then adjust the account-specific `config_dir` values. Each directory is a separate native GitHub CLI configuration containing one logged-in account.

```bash
mkdir -p ~/.config/gh-router
cp config.example.yaml ~/.config/gh-router/config.yaml
```

The first repository rule for an organisation creates its implicit organisation account. Later repository rules for that organisation remain exceptions and do not silently change the fallback account.

For example:

```yaml
default: SamCullin

accounts:
  SamCullin:
    config_dir: ~/.config/gh
  SamWork:
    config_dir: ~/.config/gh-router/accounts/SamWork

orgs:
  OpenAI:
    account: SamWork

repos:
  OpenAI/sensitive-repo:
    account: SamCullin
```

## Account setup

Log in to each account using its own `GH_CONFIG_DIR`:

```bash
mkdir -p ~/.config/gh-router/accounts/SamWork
GH_CONFIG_DIR="$HOME/.config/gh-router/accounts/SamWork" gh auth login --hostname github.com
```

The real GitHub CLI is used for login. Once credentials exist, route commands through the wrapper:

```bash
gh-router pr list
gh-router --account SamCullin pr create
gh-router auth status
gh-router auth status --resolve -R OpenAI/sensitive-repo
```

To configure rules without editing YAML:

```bash
gh-router auth set --default SamCullin --config-dir ~/.config/gh
gh-router auth set --org OpenAI --account SamWork --config-dir ~/.config/gh-router/accounts/SamWork
gh-router auth set --repo OpenAI/sensitive-repo --account SamCullin
gh-router auth unset --repo OpenAI/sensitive-repo
```

`gh-router auth switch` is intentionally a successful no-op and explains that account selection is automatic. `gh-router auth status` reports routing configuration and never reports a mutable active account.

## Installation with Homebrew

Tagged releases build checksum-verified archives for macOS and Linux and publish a formula to `SamCullin/homebrew-tap`:

```bash
brew tap SamCullin/tap
brew install gh-router
```

Upgrade later with:

```bash
brew update
brew upgrade gh-router
```

The tap is maintained in [`SamCullin/homebrew-tap`](https://github.com/SamCullin/homebrew-tap). Tagged releases publish checksum-verified archives for macOS and Linux and update the formula automatically.

For local development:

```bash
make test
make vet
make build
```

To use the built binary as a transparent `gh` replacement, put a symlink named `gh` earlier on `PATH` than the real GitHub CLI:

```bash
mkdir -p "$HOME/.local/bin"
ln -sf "$(brew --prefix gh-router)/bin/gh-router" "$HOME/.local/bin/gh"
export PATH="$HOME/.local/bin:$PATH"
```

If automatic real-CLI discovery is unsuitable, set `GH_ROUTER_REAL_GH` to the native executable path.

## Contributing

Bug reports, feature requests, and pull requests are welcome. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for development checks and contribution guidelines.

## License

Released under the [MIT License](LICENSE).
