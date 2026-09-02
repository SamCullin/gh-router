package target

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveUsesExplicitRepositoryBeforeOtherSources(t *testing.T) {
	directory := t.TempDir()
	resolved, err := Resolve([]string{"-R", "OtherOrg/bar", "pr", "list"}, map[string]string{"GH_REPO": "OpenAI/foo"}, directory)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Repository != "OtherOrg/bar" || resolved.Source != SourceCommand || resolved.Directory != directory {
		t.Fatalf("unexpected target: %#v", resolved)
	}
}

func TestResolveUsesCurrentDirectoryForPathRouting(t *testing.T) {
	directory := t.TempDir()
	resolved, err := Resolve(nil, map[string]string{}, directory)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Directory != directory {
		t.Fatalf("expected directory %s, got %#v", directory, resolved)
	}
}

func TestResolveFindsAPIRepository(t *testing.T) {
	resolved, err := Resolve([]string{"api", "repos/OtherOrg/bar/issues"}, map[string]string{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Repository != "OtherOrg/bar" {
		t.Fatalf("unexpected target: %#v", resolved)
	}
}

func TestResolveFindsRepositoryFromRemote(t *testing.T) {
	directory := t.TempDir()
	gitCommand(t, directory, "init")
	gitCommand(t, directory, "remote", "add", "origin", "git@github.com:OpenAI/foo.git")
	resolved, err := Resolve(nil, map[string]string{}, directory)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Repository != "OpenAI/foo" || resolved.Source != SourceRepository {
		t.Fatalf("unexpected target: %#v", resolved)
	}
}

func TestResolveUsesOrganisationWhenThereIsNoRepository(t *testing.T) {
	resolved, err := Resolve([]string{"repo", "create", "--org", "OpenAI"}, map[string]string{}, filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Organisation != "OpenAI" || resolved.Source != SourceOrganisation {
		t.Fatalf("unexpected target: %#v", resolved)
	}
}

func gitCommand(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", arguments, err, output)
	}
}
