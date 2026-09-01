package credentials

import (
	"path/filepath"
	"testing"

	"github.com/SamCullin/gh-router/internal/config"
)

func TestEnvironmentForReplacesInheritedTokens(t *testing.T) {
	configuration := config.New("SamCullin")
	configuration.SetAccountConfig("SamCullin", filepath.Join(t.TempDir(), "gh"))
	environment, err := EnvironmentFor(configuration, "SamCullin", "resolved-token", map[string]string{
		"GH_TOKEN":     "wrong-token",
		"GITHUB_TOKEN": "also-wrong-token",
		"PATH":         "/bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if environment["GH_TOKEN"] != "resolved-token" {
		t.Fatalf("resolved token was not set: %#v", environment)
	}
	if _, exists := environment["GITHUB_TOKEN"]; exists {
		t.Fatal("GITHUB_TOKEN was not removed")
	}
}
