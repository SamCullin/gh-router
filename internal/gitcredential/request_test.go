package gitcredential

import "testing"

func TestParseReadsRoutingFieldsAndKeepsRawInput(t *testing.T) {
	input := []byte("protocol=https\nhost=github.com\npath=SamCullin/web-agent.git\nusername=ignored\n\n")
	request := Parse(input)
	if request.Protocol != "https" || request.Host != "github.com" || request.Path != "SamCullin/web-agent.git" {
		t.Fatalf("Parse() = %#v", request)
	}
	if string(request.Raw) != string(input) {
		t.Fatal("Parse() must keep the raw request for the native helper")
	}
	if got := request.Repository(); got != "SamCullin/web-agent" {
		t.Fatalf("Repository() = %q, want SamCullin/web-agent", got)
	}
	if !request.IsGitHub() {
		t.Fatal("github.com requests should be routed")
	}
}

func TestParseStopsAtBlankLine(t *testing.T) {
	request := Parse([]byte("host=github.com\n\npath=other/repo\n"))
	if request.Path != "" {
		t.Fatalf("fields after the blank line must be ignored, got path %q", request.Path)
	}
}

func TestRepositoryRequiresOwnerAndName(t *testing.T) {
	for _, path := range []string{"", "SamCullin", "/", "SamCullin/"} {
		if got := (Request{Path: path}).Repository(); got != "" {
			t.Fatalf("Repository() for path %q = %q, want empty", path, got)
		}
	}
}

func TestIsGitHubRejectsOtherHosts(t *testing.T) {
	if (Request{Host: "gitlab.com"}).IsGitHub() {
		t.Fatal("other hosts must not be routed")
	}
	if !(Request{}).IsGitHub() {
		t.Fatal("a request without a host defaults to github.com")
	}
	if !(Request{Host: "GitHub.com"}).IsGitHub() {
		t.Fatal("host matching is case-insensitive")
	}
}
