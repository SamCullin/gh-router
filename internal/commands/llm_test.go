package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintLLMPrompt(t *testing.T) {
	var output bytes.Buffer
	PrintLLMPrompt(&output)

	prompt := output.String()
	for _, expected := range []string{
		"You are helping a user install and configure gh-router.",
		"gh router llm-text",
		"Never use `gh auth switch`",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("LLM prompt does not contain %q:\n%s", expected, prompt)
		}
	}
}
