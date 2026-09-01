package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SamCullin/gh-router/internal/config"
)

func TestSetCreatesStableImplicitOrganisationRule(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err := Set(store, []string{"--default", "SamCullin"}); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, []string{"--repo", "PersonalOrg/project", "--account", "SamCullin"}); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, []string{"--repo", "PersonalOrg/admin", "--account", "SamAdmin"}); err != nil {
		t.Fatal(err)
	}
	configuration, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	rule, _, ok := configuration.ImplicitOrganisationRule("PersonalOrg")
	if !ok || rule.Account != "SamCullin" || rule.Source != "PersonalOrg/project" {
		t.Fatalf("unexpected implicit rule: %#v", configuration.ImplicitOrgs)
	}
}

func TestStatusResolveReportsSource(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	configuration := config.New("SamCullin")
	configuration.Orgs["OpenAI"] = config.Rule{Account: "SamWork"}
	if err := store.Save(configuration); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := Status(store, &output, []string{"--resolve", "-R", "OpenAI/foo"}, "", map[string]string{}, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "account:    SamWork") || !strings.Contains(output.String(), "source:     org:OpenAI") {
		t.Fatalf("unexpected status output: %s", output.String())
	}
}
