# Contributing to gh-router

Contributions are welcome through pull requests. Please open an issue first for a large or user-facing change so the approach can be discussed before implementation.

## Development setup

You need Go 1.23 or newer and the GitHub CLI available on `PATH`.

Run the checks locally from the repository root:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/gh-router
```

The project also provides a `Makefile` with `make test`, `make vet`, `make build`, and `make fmt` targets.

## Pull requests

- Keep changes focused and explain the user-facing effect in the pull request description.
- Add or update tests for behaviour changes.
- Update the README or other documentation when commands, configuration, or installation change.
- Do not include tokens, personal GitHub configuration directories, or other credentials in commits or test fixtures.
- Keep the default branch buildable and all required checks passing.

Use Conventional Commits for commit messages, for example `fix: handle repository aliases` or `docs: clarify account setup`.

## Reporting security issues

Please follow [`SECURITY.md`](SECURITY.md) instead of opening a public issue for a vulnerability.
