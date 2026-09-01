# gh-router

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

The release workflow expects a `SamCullin/homebrew-tap` repository and a repository secret named `HOMEBREW_TAP_GITHUB_TOKEN` with permission to push formula updates there. The first release creates `Formula/gh-router.rb` in that tap.

For local development:

```bash
make test
make vet
make build
```

To use the built binary as a transparent `gh` replacement, put a symlink named `gh` earlier on `PATH` than the real GitHub CLI:

```bash
ln -sf "$PWD/bin/gh-router" "$HOME/.local/bin/gh"
```

If automatic real-CLI discovery is unsuitable, set `GH_ROUTER_REAL_GH` to the native executable path.
