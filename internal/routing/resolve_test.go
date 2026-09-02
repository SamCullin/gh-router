package routing

import (
	"path/filepath"
	"testing"

	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/target"
)

func TestResolveAccountPrecedence(t *testing.T) {
	configuration := config.New("SamCullin")
	configuration.Orgs["OpenAI"] = config.Rule{Account: "SamWork"}
	configuration.Repos["OpenAI/sensitive-repo"] = config.Rule{Account: "SamAdmin"}
	configuration.Paths["/source/insurgence-ai"] = config.Rule{Account: "SamInsurgence"}
	configuration.ImplicitOrgs["PersonalOrg"] = config.ImplicitOrg{Account: "SamCullin", Source: "PersonalOrg/project"}

	tests := []struct {
		name     string
		target   target.Target
		override string
		account  string
		source   Source
	}{
		{name: "repository", target: target.Target{Repository: "OpenAI/sensitive-repo"}, account: "SamAdmin", source: SourceRepo},
		{name: "organisation", target: target.Target{Repository: "OpenAI/other"}, account: "SamWork", source: SourceOrg},
		{name: "implicit organisation", target: target.Target{Repository: "PersonalOrg/other"}, account: "SamCullin", source: SourceImplicitOrg},
		{name: "path", target: target.Target{Directory: filepath.Join("/source", "insurgence-ai", "repo")}, account: "SamInsurgence", source: SourcePath},
		{name: "default", target: target.Target{Repository: "OtherOrg/other"}, account: "SamCullin", source: SourceDefault},
		{name: "override", target: target.Target{Repository: "OpenAI/sensitive-repo"}, override: "SamCullin", account: "SamCullin", source: SourceOverride},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolution, err := ResolveAccount(configuration, test.target, test.override)
			if err != nil {
				t.Fatal(err)
			}
			if resolution.Account != test.account || resolution.Source != test.source {
				t.Fatalf("unexpected resolution: %#v", resolution)
			}
		})
	}
}
