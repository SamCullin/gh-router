package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const routerHelp = `gh-router routes GitHub CLI commands to a configured account without changing gh's active account.

Router help (available under gh router or gh-router):
  gh router --help
      Show this router help.
  gh-router --help
      Show this help when calling the router binary directly.
  gh router auth setup ACCOUNT [--config-dir DIR] [gh auth login flags]
      Create an isolated GitHub CLI config and authenticate an account.
      The equivalent alias is gh router auth login.
  gh router auth status
      Show configured accounts and routing rules.
  gh router auth status --resolve [-R OWNER/REPO]
      Show the account selected for a target.
  gh router auth set --default ACCOUNT
      Set the default account.
  gh router auth set --org ORGANISATION --account ACCOUNT
      Route an organisation to an account.
  gh router auth set --repo OWNER/REPO --account ACCOUNT
      Route a repository to an account.
  gh router auth set --path DIRECTORY --account ACCOUNT
      Route a directory and its descendants to an account.
  gh router auth unset --org ORGANISATION
  gh router auth unset --repo OWNER/REPO
  gh router auth unset --path DIRECTORY
      Remove a routing rule.
  gh --account ACCOUNT <gh command> [flags]
      Override routing for one command.
  gh router auth switch
      Explain that account switching is automatic.
  gh router llm-text
      Print the setup and usage prompt for an LLM.

Router settings:
  Config file:       ~/.config/gh-router/config.yaml
  Account configs:    ~/.config/gh-router/accounts/ACCOUNT
  Config path override: GH_ROUTER_CONFIG=/path/to/config.yaml
  Selection order:   --account, repository, organisation, path, implicit organisation, default.
  The first configured account becomes the default.

Examples:
  gh router auth setup SamWork
  gh router auth set --org OpenAI --account SamWork
  gh router auth set --repo SamCullin/project --account SamPersonal
  gh router auth status --resolve -R OpenAI/project
  gh --account SamPersonal issue list -R SamCullin/project

Normal GitHub CLI commands, including gh --help and gh auth status, remain native.
All other GitHub CLI commands are forwarded to gh with account routing applied.
Only gh auth switch is intercepted with guidance to use router configuration.`

func PrintHelp(writer io.Writer) {
	if writer == nil {
		writer = os.Stdout
	}
	fmt.Fprintln(writer, strings.TrimSpace(routerHelp))
	fmt.Fprintln(writer)
}
