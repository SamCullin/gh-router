package args

import (
	"fmt"
	"strings"
)

func ExtractAccountOverride(arguments []string) (string, []string, error) {
	override := ""
	forwarded := make([]string, 0, len(arguments))
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch {
		case argument == "--account":
			if index+1 >= len(arguments) || strings.TrimSpace(arguments[index+1]) == "" {
				return "", nil, fmt.Errorf("--account requires an account name")
			}
			if override != "" {
				return "", nil, fmt.Errorf("--account may only be specified once")
			}
			override = arguments[index+1]
			index++
		case strings.HasPrefix(argument, "--account="):
			value := strings.TrimPrefix(argument, "--account=")
			if strings.TrimSpace(value) == "" {
				return "", nil, fmt.Errorf("--account requires an account name")
			}
			if override != "" {
				return "", nil, fmt.Errorf("--account may only be specified once")
			}
			override = value
		default:
			forwarded = append(forwarded, argument)
		}
	}
	return override, forwarded, nil
}
