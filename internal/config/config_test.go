package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTripPreservesRoutingConfiguration(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	configuration := New("SamCullin")
	configuration.SetAccountConfig("SamCullin", "~/.config/gh")
	configuration.SetOrganisationRule("OpenAI", "SamWork")
	if err := configuration.SetRepositoryRule("OpenAI/sensitive-repo", "SamAdmin"); err != nil {
		t.Fatal(err)
	}
	configuration.SetImplicitOrganisationRule("PersonalOrg", "SamCullin", "PersonalOrg/project")

	if err := store.Save(configuration); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Default != "SamCullin" || loaded.Orgs["OpenAI"].Account != "SamWork" {
		t.Fatalf("unexpected configuration: %#v", loaded)
	}
	if account, _, ok := loaded.RepositoryRule("openai/SENSITIVE-rePO"); !ok || account != "SamAdmin" {
		t.Fatalf("repository rule was not preserved: %#v", loaded.Repos)
	}
}

func TestLoadReportsMissingConfiguration(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store := NewStore(path)
	if err := store.Save(New("SamCullin")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}
