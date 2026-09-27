package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SamCullin/gh-router/internal/args"
	"github.com/SamCullin/gh-router/internal/commands"
	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/credentials"
	ghexec "github.com/SamCullin/gh-router/internal/exec"
	"github.com/SamCullin/gh-router/internal/gitcredential"
	"github.com/SamCullin/gh-router/internal/routing"
	"github.com/SamCullin/gh-router/internal/target"
	"github.com/SamCullin/gh-router/internal/version"
)

func main() {
	if filepath.Base(os.Args[0]) == "ghrllm.text" {
		commands.PrintLLMPrompt(os.Stdout)
		return
	}
	if err := run(os.Args[1:]); err != nil {
		var exitError *ghexec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.Code)
		}
		fmt.Fprintf(os.Stderr, "gh-router: %v\n", err)
		os.Exit(2)
	}
}

func run(rawArguments []string) error {
	if len(rawArguments) == 1 && rawArguments[0] == "ghrllm.text" {
		commands.PrintLLMPrompt(os.Stdout)
		return nil
	}

	if isDirectRouterInvocation(os.Args[0]) && isRootHelpRequest(rawArguments) {
		commands.PrintHelp(os.Stdout)
		return nil
	}

	if isRouterNamespace(rawArguments) {
		return runRouterCommand(rawArguments[1:])
	}
	if isDirectRouterInvocation(os.Args[0]) && isDirectRouterCommand(rawArguments) {
		return runRouterCommand(rawArguments)
	}

	commandArguments := append([]string(nil), rawArguments...)
	accountOverride, commandArguments, err := args.ExtractAccountOverride(commandArguments)
	if err != nil {
		return err
	}

	if isNativeAuthSwitch(commandArguments) {
		commands.PrintSwitchMessage(os.Stdout)
		return nil
	}
	if len(commandArguments) == 1 && commandArguments[0] == "--version" {
		if isDirectRouterInvocation(os.Args[0]) {
			fmt.Printf("gh-router %s\n", version.Version)
			return nil
		}
		return executeNative(commandArguments)
	}

	if isGitCredentialCommand(commandArguments) {
		return runGitCredential(commandArguments, accountOverride)
	}
	if isRoutedAuthTokenCommand(commandArguments) {
		return runForResolvedAccount(commandArguments, accountOverride)
	}
	if isNativeAuthCommand(commandArguments) {
		if strings.TrimSpace(accountOverride) != "" {
			return runForResolvedAccount(commandArguments, accountOverride)
		}
		return executeNative(commandArguments)
	}
	if isPassthroughWithoutRouting(commandArguments) {
		return executeNative(commandArguments)
	}

	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	store := config.NewStore(path)

	configuration, err := store.Load()
	if err != nil {
		return err
	}
	targetRepository, err := target.Resolve(commandArguments, nil, "")
	if err != nil {
		return err
	}
	resolution, err := routing.ResolveAccount(configuration, targetRepository, accountOverride)
	if err != nil {
		return err
	}
	return ghexec.Run(commandArguments, configuration, resolution, os.Args[0], nil)
}

// runForResolvedAccount runs a native auth command against the isolated
// config directory of the routed account. It sets GH_CONFIG_DIR rather than
// GH_TOKEN so native commands such as gh auth token read that account.
func runForResolvedAccount(arguments []string, accountOverride string) error {
	configuration, err := loadConfiguration()
	if err != nil {
		return err
	}
	targetRepository, err := target.Resolve(nil, nil, "")
	if err != nil {
		return err
	}
	resolution, err := routing.ResolveAccount(configuration, targetRepository, accountOverride)
	if err != nil {
		return err
	}
	environment, err := accountEnvironment(configuration, resolution.Account)
	if err != nil {
		return err
	}
	realGH, err := ghexec.FindRealGH(os.Args[0], nil)
	if err != nil {
		return err
	}
	return ghexec.Execute(realGH, arguments, environment)
}

// runGitCredential serves git's credential helper protocol. git runs the
// helper from the repository, so the account comes from --account, the
// request path (credential.useHttpPath), the checkout's origin remote and
// path rules, then the default account.
func runGitCredential(arguments []string, accountOverride string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read git credential request: %w", err)
	}
	request := gitcredential.Parse(input)
	realGH, err := ghexec.FindRealGH(os.Args[0], nil)
	if err != nil {
		return err
	}
	if !request.IsGitHub() {
		return ghexec.ExecuteWithInput(realGH, arguments, nil, input)
	}
	configuration, err := loadConfiguration()
	if err != nil {
		return err
	}
	targetRepository, err := gitCredentialTarget(request)
	if err != nil {
		return err
	}
	resolution, err := routing.ResolveAccount(configuration, targetRepository, accountOverride)
	if err != nil {
		return err
	}
	environment, err := accountEnvironment(configuration, resolution.Account)
	if err != nil {
		return err
	}
	return ghexec.ExecuteWithInput(realGH, arguments, environment, input)
}

func gitCredentialTarget(request gitcredential.Request) (target.Target, error) {
	if repository := request.Repository(); repository != "" {
		return target.Target{Repository: repository, Directory: target.RepositoryRoot(""), Source: target.SourceCommand}, nil
	}
	return target.Resolve(nil, nil, "")
}

func accountEnvironment(configuration config.Config, account string) (map[string]string, error) {
	directory, err := credentials.ConfigDirectory(configuration, account)
	if err != nil {
		return nil, err
	}
	return credentials.EnvironmentForDirectory(directory, nil), nil
}

func loadConfiguration() (config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, err
	}
	return config.NewStore(path).Load()
}

func executeNative(arguments []string) error {
	realGH, err := ghexec.FindRealGH(os.Args[0], nil)
	if err != nil {
		return err
	}
	return ghexec.Execute(realGH, arguments, nil)
}

func runRouterCommand(rawArguments []string) error {
	if isRouterHelpRequest(rawArguments) {
		commands.PrintHelp(os.Stdout)
		return nil
	}
	if len(rawArguments) == 1 && rawArguments[0] == "--version" {
		fmt.Printf("gh-router %s\n", version.Version)
		return nil
	}
	if len(rawArguments) == 1 && rawArguments[0] == "llm-text" {
		commands.PrintLLMPrompt(os.Stdout)
		return nil
	}

	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	store := config.NewStore(path)

	if len(rawArguments) == 0 {
		return fmt.Errorf("router command is required")
	}

	switch rawArguments[0] {
	case "auth":
		if len(rawArguments) < 2 {
			return fmt.Errorf("router auth command is required")
		}
		switch rawArguments[1] {
		case "switch":
			commands.PrintSwitchMessage(os.Stdout)
			return nil
		case "set":
			return commands.Set(store, rawArguments[2:])
		case "setup", "login":
			return commands.Setup(store, os.Stdout, rawArguments[2:], os.Args[0], nil, nil)
		case "unset":
			return commands.Unset(store, rawArguments[2:])
		case "status":
			return commands.Status(store, os.Stdout, rawArguments[2:], "", nil, "")
		case "resolve":
			return commands.Status(store, os.Stdout, append([]string{"--resolve"}, rawArguments[2:]...), "", nil, "")
		default:
			return fmt.Errorf("unsupported router auth command: %s", rawArguments[1])
		}
	case "override":
		return commands.Override(os.Stdout, rawArguments[1:], os.Args[0])
	default:
		return fmt.Errorf("unsupported router command: %s", rawArguments[0])
	}
}

func isRouterNamespace(arguments []string) bool {
	return len(arguments) > 0 && arguments[0] == "router"
}

func isDirectRouterInvocation(argv0 string) bool {
	switch filepath.Base(argv0) {
	case "gh-router", "ghr":
		return true
	default:
		return false
	}
}

func isRouterHelpRequest(arguments []string) bool {
	if len(arguments) == 0 {
		return true
	}
	for _, argument := range arguments {
		if argument == "--help" || argument == "help" {
			return true
		}
	}
	return false
}

func isRootHelpRequest(arguments []string) bool {
	return len(arguments) == 0 || (len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "help"))
}

func isNativeAuthSwitch(arguments []string) bool {
	return len(arguments) >= 2 && arguments[0] == "auth" && arguments[1] == "switch"
}

func isNativeAuthCommand(arguments []string) bool {
	return len(arguments) > 0 && arguments[0] == "auth" && !isNativeAuthSwitch(arguments)
}

func isRouterAuthCommand(arguments []string) bool {
	return len(arguments) > 0 && arguments[0] == "auth"
}

func isDirectRouterCommand(arguments []string) bool {
	return isDirectRouterAuthCommand(arguments) || isRouterOverrideCommand(arguments) || (len(arguments) == 1 && arguments[0] == "llm-text")
}

// isDirectRouterAuthCommand keeps ghr auth <router subcommand> in the router
// while other auth subcommands, such as token or git-credential, reach the
// GitHub CLI with account routing.
func isDirectRouterAuthCommand(arguments []string) bool {
	if !isRouterAuthCommand(arguments) {
		return false
	}
	if len(arguments) == 1 {
		return true
	}
	switch arguments[1] {
	case "switch", "set", "setup", "login", "unset", "status", "resolve":
		return true
	default:
		return false
	}
}

func isGitCredentialCommand(arguments []string) bool {
	return len(arguments) >= 2 && arguments[0] == "auth" && arguments[1] == "git-credential" && !isPassthroughWithoutRouting(arguments)
}

// isRoutedAuthTokenCommand routes gh auth token for github.com. An explicit
// --user or a different --hostname keeps the native behaviour.
func isRoutedAuthTokenCommand(arguments []string) bool {
	if len(arguments) < 2 || arguments[0] != "auth" || arguments[1] != "token" || isPassthroughWithoutRouting(arguments) {
		return false
	}
	for index, argument := range arguments[2:] {
		switch {
		case argument == "--user" || argument == "-u" || strings.HasPrefix(argument, "--user="):
			return false
		case argument == "--hostname" || argument == "-h":
			rest := arguments[2:]
			if index+1 < len(rest) && !strings.EqualFold(rest[index+1], "github.com") {
				return false
			}
		case strings.HasPrefix(argument, "--hostname="):
			if !strings.EqualFold(strings.TrimPrefix(argument, "--hostname="), "github.com") {
				return false
			}
		}
	}
	return true
}

func isRouterOverrideCommand(arguments []string) bool {
	return len(arguments) > 0 && arguments[0] == "override"
}

func isPassthroughWithoutRouting(arguments []string) bool {
	if len(arguments) == 0 {
		return true
	}
	for _, argument := range arguments {
		if argument == "--help" || argument == "help" {
			return true
		}
	}
	return false
}
