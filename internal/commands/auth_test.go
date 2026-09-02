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

func TestSetAndUnsetPathRule(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	configuration := config.New("SamCullin")
	if err := store.Save(configuration); err != nil {
		t.Fatal(err)
	}

	if err := Set(store, []string{"--path", "~/src/insurgence-ai", "--account", "SamInsurgence"}); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, []string{"--path", "~/src", "--account", "SamWork"}); err != nil {
		t.Fatal(err)
	}
	configuration, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Paths["~/src/insurgence-ai"].Account != "SamInsurgence" {
		t.Fatalf("unexpected path rules: %#v", configuration.Paths)
	}
	if configuration.Paths["~/src"].Account != "SamWork" || len(configuration.Paths) != 2 {
		t.Fatalf("nested path rule overwrote its parent: %#v", configuration.Paths)
	}

	if err := Unset(store, []string{"--path", "~/src/insurgence-ai"}); err != nil {
		t.Fatal(err)
	}
	configuration, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.Paths) != 1 || configuration.Paths["~/src"].Account != "SamWork" {
		t.Fatalf("path rule was not removed: %#v", configuration.Paths)
	}
	if err := Unset(store, []string{"--path", "~/src"}); err != nil {
		t.Fatal(err)
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
