package commands

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
)

//go:embed ghrllm.text
var llmPrompt string

func PrintLLMPrompt(writer io.Writer) {
	if writer == nil {
		writer = os.Stdout
	}
	fmt.Fprintln(writer, strings.TrimSpace(llmPrompt))
}
