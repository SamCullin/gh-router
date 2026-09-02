package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/SamCullin/gh-router/internal/config"
)

func TestParseSetupOptionsSeparatesRouterOptionsFromLoginOptions(t *testing.T) {
	options, err := ParseSetupOptions([]string{"--account", "SamWork", "--config-dir", "~/work", "--web"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Account != "SamWork" || options.ConfigDir != "~/work" {
		t.Fatalf("unexpected setup options: %#v", options)
	}
	if !reflect.DeepEqual(options.LoginArguments, []string{"--web"}) {
		t.Fatalf("unexpected login arguments: %#v", options.LoginArguments)
	}
}

func TestSetupCreatesDirectoryRunsIsolatedLoginAndSavesConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ghDirectory := t.TempDir()
	realGH := filepath.Join(ghDirectory, "gh")
	if err := os.WriteFile(realGH, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	store := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	var loginRealGH string
	var loginArguments []string
	var loginEnvironment map[string]string
	login := func(path string, arguments []string, environment map[string]string) error {
		loginRealGH = path
		loginArguments = append([]string(nil), arguments...)
		loginEnvironment = environment
		return nil
	}
	var output bytes.Buffer

	err := Setup(store, &output, []string{"SamWork", "--web"}, filepath.Join(t.TempDir(), "gh-router"), map[string]string{
		"PATH":         ghDirectory,
		"GH_TOKEN":     "inherited-token",
		"GITHUB_TOKEN": "inherited-token",
	}, login)
	if err != nil {
		t.Fatal(err)
	}

	expectedDirectory := filepath.Join(home, ".config", "gh-router", "accounts", "SamWork")
	if loginRealGH != realGH {
		t.Fatalf("expected %s, got %s", realGH, loginRealGH)
	}
	if !reflect.DeepEqual(loginArguments, []string{"auth", "login", "--hostname", "github.com", "--web"}) {
		t.Fatalf("unexpected login arguments: %#v", loginArguments)
	}
	if loginEnvironment["GH_CONFIG_DIR"] != expectedDirectory {
		t.Fatalf("unexpected GH_CONFIG_DIR: %#v", loginEnvironment)
	}
	if _, exists := loginEnvironment["GH_TOKEN"]; exists {
		t.Fatal("GH_TOKEN was inherited by login")
	}
	if _, exists := loginEnvironment["GITHUB_TOKEN"]; exists {
		t.Fatal("GITHUB_TOKEN was inherited by login")
	}
	info, err := os.Stat(expectedDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("expected private config directory, got %o", info.Mode().Perm())
	}

	configuration, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Default != "SamWork" {
		t.Fatalf("expected first account to become default: %#v", configuration)
	}
	account, ok := configuration.Account("SamWork")
	if !ok || account.ConfigDir != "~/.config/gh-router/accounts/SamWork" {
		t.Fatalf("unexpected account configuration: %#v", configuration.Accounts)
	}
	if !strings.Contains(output.String(), "Account configured: SamWork") {
		t.Fatalf("unexpected setup output: %s", output.String())
	}
}

func TestSetupPreservesExistingDefaultAndHonoursHostname(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	configuration := config.New("SamCullin")
	if err := store.Save(configuration); err != nil {
		t.Fatal(err)
	}
	ghDirectory := t.TempDir()
	realGH := filepath.Join(ghDirectory, "gh")
	if err := os.WriteFile(realGH, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	var loginArguments []string
	login := func(_ string, arguments []string, _ map[string]string) error {
		loginArguments = append([]string(nil), arguments...)
		return nil
	}
	if err := Setup(store, nil, []string{"SamWork", "--config-dir", "~/work", "--hostname", "github.example"}, realGH, map[string]string{"PATH": ghDirectory}, login); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Default != "SamCullin" {
		t.Fatalf("existing default changed: %#v", loaded)
	}
	account, ok := loaded.Account("SamWork")
	if !ok || account.ConfigDir != "~/work" {
		t.Fatalf("explicit config directory was not saved: %#v", loaded.Accounts)
	}
	if !reflect.DeepEqual(loginArguments, []string{"auth", "login", "--hostname", "github.example"}) {
		t.Fatalf("unexpected login arguments: %#v", loginArguments)
	}
}
