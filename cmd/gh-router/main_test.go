package main

import "testing"

func TestRouterNamespaceAndHelpRequests(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		namespace bool
		help      bool
	}{
		{name: "router help", arguments: []string{"router", "--help"}, namespace: true, help: true},
		{name: "router command", arguments: []string{"router", "auth", "status"}, namespace: true, help: false},
		{name: "router llm text", arguments: []string{"router", "llm-text"}, namespace: true, help: false},
		{name: "native help", arguments: []string{"--help"}, namespace: false, help: false},
		{name: "native command help", arguments: []string{"issue", "list", "--help"}, namespace: false, help: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isRouterNamespace(test.arguments); got != test.namespace {
				t.Fatalf("isRouterNamespace(%#v) = %t, want %t", test.arguments, got, test.namespace)
			}
			if test.namespace {
				if got := isRouterHelpRequest(test.arguments[1:]); got != test.help {
					t.Fatalf("isRouterHelpRequest(%#v) = %t, want %t", test.arguments[1:], got, test.help)
				}
			}
		})
	}
}

func TestDirectRouterInvocation(t *testing.T) {
	for _, argv0 := range []string{"gh-router", "ghr", "/opt/homebrew/bin/gh-router"} {
		if !isDirectRouterInvocation(argv0) {
			t.Fatalf("%q should be a direct router invocation", argv0)
		}
	}
	if isDirectRouterInvocation("gh") {
		t.Fatal("gh should remain the native CLI invocation")
	}
	if !isRouterAuthCommand([]string{"auth", "status"}) {
		t.Fatal("auth status should be a router command for direct router invocations")
	}
	if !isDirectRouterCommand([]string{"llm-text"}) {
		t.Fatal("llm-text should be a router command for direct router invocations")
	}
	if isRouterAuthCommand([]string{"issue", "list"}) {
		t.Fatal("issue list should remain a GitHub CLI command")
	}
	if isDirectRouterCommand([]string{"issue", "list"}) {
		t.Fatal("issue list should remain a GitHub CLI command")
	}
	for _, arguments := range [][]string{{}, {"--help"}, {"help"}} {
		if !isRootHelpRequest(arguments) {
			t.Fatalf("%#v should be direct router help", arguments)
		}
	}
	if isRootHelpRequest([]string{"issue", "list", "--help"}) {
		t.Fatal("command-specific help should remain native")
	}
	for _, arguments := range [][]string{{}, {"--help"}, {"help"}, {"issue", "list", "--help"}, {"help", "issue"}} {
		if !isPassthroughWithoutRouting(arguments) {
			t.Fatalf("%#v should be forwarded to native gh", arguments)
		}
	}
	if isPassthroughWithoutRouting([]string{"issue", "list"}) {
		t.Fatal("normal commands should remain routed")
	}
}

func TestNativeAuthSwitchIsTheOnlyNativeAuthInterception(t *testing.T) {
	if !isNativeAuthSwitch([]string{"auth", "switch"}) {
		t.Fatal("auth switch should be intercepted")
	}
	if !isNativeAuthSwitch([]string{"auth", "switch", "--help"}) {
		t.Fatal("auth switch with flags should be intercepted")
	}
	for _, arguments := range [][]string{
		{"auth", "status"},
		{"auth", "login"},
		{"auth", "logout"},
		{"issue", "list"},
	} {
		if isNativeAuthSwitch(arguments) {
			t.Fatalf("%#v should remain native", arguments)
		}
	}
	for _, arguments := range [][]string{
		{"auth", "status"},
		{"auth", "login"},
		{"auth", "logout"},
	} {
		if !isNativeAuthCommand(arguments) {
			t.Fatalf("%#v should be forwarded to native gh", arguments)
		}
	}
	if isNativeAuthCommand([]string{"auth", "switch"}) {
		t.Fatal("auth switch should not be classified as a native command")
	}
}
