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
	configuration.Paths["~/src/insurgence-ai"] = Rule{Account: "SamWork"}
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
	if loaded.Paths["~/src/insurgence-ai"].Account != "SamWork" {
		t.Fatalf("path rule was not preserved: %#v", loaded.Paths)
	}
	if account, _, ok := loaded.RepositoryRule("openai/SENSITIVE-rePO"); !ok || account != "SamAdmin" {
		t.Fatalf("repository rule was not preserved: %#v", loaded.Repos)
	}
}

func TestPathRuleMatchesDescendantsAndPrefersMostSpecific(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configuration := New("SamCullin")
	configuration.Paths["~/src"] = Rule{Account: "SamWork"}
	configuration.Paths["~/src/insurgence-ai"] = Rule{Account: "SamInsurgence"}

	account, path, ok := configuration.PathRule(filepath.Join(home, "src", "insurgence-ai", "repo", "main"))
	if !ok || account != "SamInsurgence" || path != "~/src/insurgence-ai" {
		t.Fatalf("unexpected nested path resolution: %q, %q, %v", account, path, ok)
	}

	account, path, ok = configuration.PathRule(filepath.Join(home, "src", "other-org", "repo"))
	if !ok || account != "SamWork" || path != "~/src" {
		t.Fatalf("unexpected parent path resolution: %q, %q, %v", account, path, ok)
	}

	if _, _, ok = configuration.PathRule(filepath.Join(home, "src-other", "repo")); ok {
		t.Fatal("path rule matched a sibling directory")
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

func TestManagedAccountConfigDirUsesAccountSpecificDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	directory, err := ManagedAccountConfigDir("SamWork")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, ".config", "gh-router", "accounts", "SamWork")
	if directory != expected {
		t.Fatalf("expected %s, got %s", expected, directory)
	}

	reference, err := ManagedAccountConfigReference("SamWork")
	if err != nil {
		t.Fatal(err)
	}
	if reference != filepath.Join("~", ".config", "gh-router", "accounts", "SamWork") {
		t.Fatalf("unexpected config reference: %s", reference)
	}
}

func TestValidateAccountNameRejectsPathTraversal(t *testing.T) {
	for _, account := range []string{"", ".", "..", "../work", `work\\admin`, "work\nadmin"} {
		if _, err := ValidateAccountName(account); err == nil {
			t.Errorf("expected account name %q to be rejected", account)
		}
	}
}
