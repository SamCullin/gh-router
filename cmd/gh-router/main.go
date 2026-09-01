package main

import (
	"fmt"
	"os"

	"github.com/SamCullin/gh-router/internal/args"
	"github.com/SamCullin/gh-router/internal/commands"
	"github.com/SamCullin/gh-router/internal/config"
	ghexec "github.com/SamCullin/gh-router/internal/exec"
	"github.com/SamCullin/gh-router/internal/routing"
	"github.com/SamCullin/gh-router/internal/target"
	"github.com/SamCullin/gh-router/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gh-router: %v\n", err)
		os.Exit(2)
	}
}

func run(rawArguments []string) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	store := config.NewStore(path)

	commandArguments := append([]string(nil), rawArguments...)
	accountOverride := ""
	if len(commandArguments) == 0 || commandArguments[0] != "auth" {
		accountOverride, commandArguments, err = args.ExtractAccountOverride(commandArguments)
		if err != nil {
			return err
		}
	}

	if len(commandArguments) >= 2 && commandArguments[0] == "auth" {
		switch commandArguments[1] {
		case "switch":
			commands.PrintSwitchMessage(os.Stdout)
			return nil
		case "set":
			if accountOverride != "" {
				commandArguments = append(commandArguments, "--account", accountOverride)
			}
			return commands.Set(store, commandArguments[2:])
		case "unset":
			return commands.Unset(store, commandArguments[2:])
		case "status":
			return commands.Status(store, os.Stdout, commandArguments[2:], accountOverride, nil, "")
		case "resolve":
			return commands.Status(store, os.Stdout, append([]string{"--resolve"}, commandArguments[2:]...), accountOverride, nil, "")
		}
	}
	if len(commandArguments) == 1 && commandArguments[0] == "--version" {
		fmt.Printf("gh-router %s\n", version.Version)
		return nil
	}

	if isPassthroughWithoutRouting(commandArguments) {
		realGH, err := ghexec.FindRealGH(os.Args[0], nil)
		if err != nil {
			return err
		}
		return ghexec.Execute(realGH, commandArguments, nil)
	}

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

func isPassthroughWithoutRouting(arguments []string) bool {
	if len(arguments) == 0 {
		return true
	}
	if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "help") {
		return true
	}
	return false
}
