package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectRouterAuthKeepsOnlyRouterSubcommands(t *testing.T) {
	for _, arguments := range [][]string{
		{"auth"},
		{"auth", "status"},
		{"auth", "setup", "SamWork"},
		{"auth", "login"},
		{"auth", "set", "--default", "SamWork"},
		{"auth", "unset", "--org", "OpenAI"},
		{"auth", "switch"},
		{"auth", "resolve"},
	} {
		if !isDirectRouterCommand(arguments) {
			t.Fatalf("%#v should stay a router command for ghr", arguments)
		}
	}
	for _, arguments := range [][]string{
		{"auth", "token"},
		{"auth", "git-credential", "get"},
		{"auth", "logout"},
		{"auth", "refresh"},
		{"auth", "setup-git"},
	} {
		if isDirectRouterCommand(arguments) {
			t.Fatalf("%#v should reach the GitHub CLI instead of failing as an unsupported router command", arguments)
		}
	}
}

func TestRoutedAuthCommandClassification(t *testing.T) {
	if !isGitCredentialCommand([]string{"auth", "git-credential", "get"}) {
		t.Fatal("git-credential get should be routed")
	}
	if isGitCredentialCommand([]string{"auth", "git-credential", "--help"}) {
		t.Fatal("git-credential help should stay native")
	}
	for _, arguments := range [][]string{
		{"auth", "token"},
		{"auth", "token", "--hostname", "github.com"},
		{"auth", "token", "-h", "GitHub.com"},
		{"auth", "token", "--hostname=github.com"},
	} {
		if !isRoutedAuthTokenCommand(arguments) {
			t.Fatalf("%#v should be routed", arguments)
		}
	}
	for _, arguments := range [][]string{
		{"auth", "token", "--hostname", "github.example.com"},
		{"auth", "token", "--hostname=github.example.com"},
		{"auth", "token", "--user", "someone"},
		{"auth", "token", "-u", "someone"},
		{"auth", "token", "--help"},
		{"auth", "status"},
	} {
		if isRoutedAuthTokenCommand(arguments) {
			t.Fatalf("%#v should keep native behaviour", arguments)
		}
	}
}

// fakeGH records the config directory, arguments and stdin it was run with,
// standing in for the native GitHub CLI.
const fakeGH = `#!/bin/sh
printf 'config_dir=%s\n' "$GH_CONFIG_DIR"
printf 'token_env=%s\n' "${GH_TOKEN:-}"
printf 'args=%s\n' "$*"
printf 'stdin<<\n'
cat
printf '>>\n'
exit ${FAKE_GH_EXIT:-0}
`

type routerFixture struct {
	binary   string
	nativeGH string
	home     string
	config   string
	workDir  string
}

func newRouterFixture(t *testing.T) routerFixture {
	t.Helper()
	directory := t.TempDir()
	binary := filepath.Join(directory, "gh-router")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = os.Environ()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build router: %v\n%s", err, output)
	}
	nativeGH := filepath.Join(directory, "native-gh")
	if err := os.WriteFile(nativeGH, []byte(fakeGH), 0755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(directory, "home")
	workDir := filepath.Join(directory, "work")
	for _, path := range []string{home, workDir} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(directory, "config.yaml")
	contents := strings.Join([]string{
		"default: Personal",
		"accounts:",
		"  Personal:",
		"    config_dir: " + filepath.Join(home, "personal"),
		"  Work:",
		"    config_dir: " + filepath.Join(home, "work"),
		"orgs:",
		"  WorkOrg:",
		"    account: Work",
		"",
	}, "\n")
	if err := os.WriteFile(config, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return routerFixture{binary: binary, nativeGH: nativeGH, home: home, config: config, workDir: workDir}
}

func (fixture routerFixture) run(t *testing.T, argv0 string, stdin string, extraEnv []string, arguments ...string) (string, int) {
	t.Helper()
	executable := fixture.binary
	if argv0 != "gh-router" {
		executable = filepath.Join(filepath.Dir(fixture.binary), argv0)
		if _, err := os.Lstat(executable); os.IsNotExist(err) {
			if err := os.Symlink(fixture.binary, executable); err != nil {
				t.Fatal(err)
			}
		}
	}
	command := exec.Command(executable, arguments...)
	command.Dir = fixture.workDir
	command.Env = append([]string{
		"PATH=/usr/bin:/bin",
		"HOME=" + fixture.home,
		"GH_ROUTER_CONFIG=" + fixture.config,
		"GH_ROUTER_REAL_GH=" + fixture.nativeGH,
		"GH_TOKEN=inherited-token-must-not-leak",
		"GIT_CONFIG_NOSYSTEM=1",
	}, extraEnv...)
	command.Stdin = strings.NewReader(stdin)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	code := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		code = exitError.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return output.String(), code
}

func TestGitCredentialRoutesByRequestPath(t *testing.T) {
	fixture := newRouterFixture(t)
	request := "protocol=https\nhost=github.com\npath=WorkOrg/service.git\n\n"
	output, code := fixture.run(t, "ghr", request, nil, "auth", "git-credential", "get")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "work")+"\n") {
		t.Fatalf("WorkOrg request should use the Work account:\n%s", output)
	}
	if !strings.Contains(output, "args=auth git-credential get\n") {
		t.Fatalf("native helper should receive the git-credential arguments:\n%s", output)
	}
	if !strings.Contains(output, "stdin<<\n"+request+">>") {
		t.Fatalf("native helper should receive the original request:\n%s", output)
	}
	if !strings.Contains(output, "token_env=\n") {
		t.Fatalf("inherited GH_TOKEN must not reach the native helper:\n%s", output)
	}
}

func TestGitCredentialFallsBackToDefaultAccount(t *testing.T) {
	fixture := newRouterFixture(t)
	output, code := fixture.run(t, "gh", "protocol=https\nhost=github.com\n\n", nil, "auth", "git-credential", "get")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "personal")+"\n") {
		t.Fatalf("a request without a repository should use the default account:\n%s", output)
	}
}

func TestGitCredentialHonoursAccountOverride(t *testing.T) {
	fixture := newRouterFixture(t)
	request := "protocol=https\nhost=github.com\npath=Personal/repo.git\n\n"
	output, code := fixture.run(t, "ghr", request, nil, "--account", "Work", "auth", "git-credential", "get")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "work")+"\n") {
		t.Fatalf("--account should win over the request path:\n%s", output)
	}
}

func TestGitCredentialLeavesOtherHostsNative(t *testing.T) {
	fixture := newRouterFixture(t)
	output, code := fixture.run(t, "ghr", "protocol=https\nhost=gitlab.com\npath=WorkOrg/service.git\n\n", nil, "auth", "git-credential", "get")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if !strings.Contains(output, "config_dir=\n") {
		t.Fatalf("other hosts must not get a routed config directory:\n%s", output)
	}
}

func TestGitCredentialPropagatesNativeExitStatus(t *testing.T) {
	fixture := newRouterFixture(t)
	output, code := fixture.run(t, "ghr", "protocol=https\nhost=github.com\n\n", []string{"FAKE_GH_EXIT=1"}, "auth", "git-credential", "get")
	if code != 1 {
		t.Fatalf("exit %d, want the native status 1:\n%s", code, output)
	}
	if strings.Contains(output, "gh-router:") {
		t.Fatalf("a native failure should not be wrapped as a router error:\n%s", output)
	}
}

func TestAuthTokenRoutesAndHonoursAccountOverride(t *testing.T) {
	fixture := newRouterFixture(t)
	output, code := fixture.run(t, "ghr", "", nil, "auth", "token")
	if code != 0 || !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "personal")+"\n") {
		t.Fatalf("ghr auth token should use the routed default account (exit %d):\n%s", code, output)
	}
	output, code = fixture.run(t, "ghr", "", nil, "--account", "Work", "auth", "token")
	if code != 0 || !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "work")+"\n") {
		t.Fatalf("--account should select the Work account (exit %d):\n%s", code, output)
	}
}

func TestNativeAuthCommandsHonourAccountOverride(t *testing.T) {
	fixture := newRouterFixture(t)
	output, code := fixture.run(t, "gh", "", nil, "--account", "Work", "auth", "status")
	if code != 0 || !strings.Contains(output, "config_dir="+filepath.Join(fixture.home, "work")+"\n") {
		t.Fatalf("--account must not be dropped for native auth commands (exit %d):\n%s", code, output)
	}
	output, code = fixture.run(t, "gh", "", nil, "auth", "status")
	if code != 0 || !strings.Contains(output, "config_dir=\n") {
		t.Fatalf("gh auth status without --account should stay native (exit %d):\n%s", code, output)
	}
}
