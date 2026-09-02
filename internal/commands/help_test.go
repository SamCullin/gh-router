package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintHelpIncludesRouterCommandsAndSettings(t *testing.T) {
	var output bytes.Buffer
	PrintHelp(&output)
	help := output.String()

	for _, expected := range []string{
		"Router help (available under gh router or gh-router):",
		"gh-router --help",
		"gh router auth setup ACCOUNT",
		"gh router auth status --resolve",
		"gh router auth set --org ORGANISATION --account ACCOUNT",
		"gh router auth set --repo OWNER/REPO --account ACCOUNT",
		"gh router llm-text",
		"Print the setup and usage prompt for an LLM.",
		"gh --account ACCOUNT <gh command> [flags]",
		"Router settings:",
		"Config file:       ~/.config/gh-router/config.yaml",
		"Account configs:    ~/.config/gh-router/accounts/ACCOUNT",
		"Config path override: GH_ROUTER_CONFIG=/path/to/config.yaml",
		"Selection order:   --account, repository, organisation, implicit organisation, default.",
		"Normal GitHub CLI commands, including gh --help and gh auth status, remain native.",
		"All other GitHub CLI commands are forwarded to gh with account routing applied.",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("help output does not contain %q:\n%s", expected, help)
		}
	}

	if strings.Index(help, "Router settings:") < strings.Index(help, "Router help (available under gh router or gh-router):") {
		t.Fatal("router settings appear before router commands")
	}
	if strings.Contains(help, "Native GitHub CLI help follows below.") {
		t.Fatal("router help should not include native GitHub CLI help")
	}
}
