package commands

import "testing"

func TestParseOverrideOptions(t *testing.T) {
	tests := []struct {
		name       string
		arguments  []string
		action     string
		targetPath string
	}{
		{name: "install", arguments: []string{"install"}, action: "install"},
		{name: "status with path", arguments: []string{"status", "--path", "/usr/local/bin/gh"}, action: "status", targetPath: "/usr/local/bin/gh"},
		{name: "uninstall with equals path", arguments: []string{"uninstall", "--path=/opt/homebrew/bin/gh"}, action: "uninstall", targetPath: "/opt/homebrew/bin/gh"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, err := ParseOverrideOptions(test.arguments)
			if err != nil {
				t.Fatal(err)
			}
			if options.Action != test.action || options.TargetPath != test.targetPath {
				t.Fatalf("unexpected options: %#v", options)
			}
		})
	}
}

func TestParseOverrideOptionsRejectsInvalidInput(t *testing.T) {
	for _, arguments := range [][]string{
		{},
		{"repair"},
		{"install", "--unknown"},
		{"install", "--path"},
		{"install", "--path", "/one", "--path", "/two"},
	} {
		if _, err := ParseOverrideOptions(arguments); err == nil {
			t.Fatalf("ParseOverrideOptions(%#v) succeeded", arguments)
		}
	}
}
