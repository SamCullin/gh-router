package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/credentials"
	ghrouterexec "github.com/SamCullin/gh-router/internal/exec"
)

type LoginExecutor func(realGH string, arguments []string, environ map[string]string) error

type SetupOptions struct {
	Account        string
	ConfigDir      string
	LoginArguments []string
}

func ParseSetupOptions(arguments []string) (SetupOptions, error) {
	var options SetupOptions
	accountSet := false
	configDirSet := false

	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch {
		case argument == "--account":
			if accountSet {
				return SetupOptions{}, fmt.Errorf("--account may only be specified once")
			}
			value, nextIndex, err := setupOptionValue(arguments, index, "--account")
			if err != nil {
				return SetupOptions{}, err
			}
			options.Account = value
			accountSet = true
			index = nextIndex
		case strings.HasPrefix(argument, "--account="):
			if accountSet {
				return SetupOptions{}, fmt.Errorf("--account may only be specified once")
			}
			options.Account = strings.TrimSpace(strings.TrimPrefix(argument, "--account="))
			accountSet = true
		case argument == "--config-dir":
			if configDirSet {
				return SetupOptions{}, fmt.Errorf("--config-dir may only be specified once")
			}
			value, nextIndex, err := setupOptionValue(arguments, index, "--config-dir")
			if err != nil {
				return SetupOptions{}, err
			}
			options.ConfigDir = value
			configDirSet = true
			index = nextIndex
		case strings.HasPrefix(argument, "--config-dir="):
			if configDirSet {
				return SetupOptions{}, fmt.Errorf("--config-dir may only be specified once")
			}
			options.ConfigDir = strings.TrimSpace(strings.TrimPrefix(argument, "--config-dir="))
			if options.ConfigDir == "" {
				return SetupOptions{}, fmt.Errorf("--config-dir requires a path")
			}
			configDirSet = true
		case argument == "--":
			options.LoginArguments = append(options.LoginArguments, arguments[index+1:]...)
			index = len(arguments)
		case strings.HasPrefix(argument, "-"):
			options.LoginArguments = append(options.LoginArguments, argument)
			if loginOptionTakesValue(argument) && index+1 < len(arguments) {
				options.LoginArguments = append(options.LoginArguments, arguments[index+1])
				index++
			}
		default:
			if accountSet {
				return SetupOptions{}, fmt.Errorf("account may only be specified once")
			}
			options.Account = argument
			accountSet = true
		}
	}

	account, err := config.ValidateAccountName(options.Account)
	if err != nil {
		return SetupOptions{}, fmt.Errorf("setup requires an account name: %w", err)
	}
	options.Account = account
	return options, nil
}

func Setup(store config.Store, writer io.Writer, arguments []string, argv0 string, environ map[string]string, login LoginExecutor) error {
	options, err := ParseSetupOptions(arguments)
	if err != nil {
		return err
	}

	configuration, err := store.Load()
	configurationExists := true
	if err != nil {
		if !errors.Is(err, config.ErrNotFound) {
			return err
		}
		configuration = config.New(options.Account)
		configurationExists = false
	}

	account := configuration.AccountName(options.Account)
	configReference := strings.TrimSpace(options.ConfigDir)
	if configReference == "" {
		if accountConfiguration, ok := configuration.Account(account); ok {
			configReference = strings.TrimSpace(accountConfiguration.ConfigDir)
		}
	}
	if configReference == "" {
		configReference, err = config.ManagedAccountConfigReference(account)
		if err != nil {
			return err
		}
	}
	configDirectory, err := config.ExpandPath(configReference)
	if err != nil {
		return fmt.Errorf("resolve account config directory: %w", err)
	}

	realGH, err := ghrouterexec.FindRealGH(argv0, environ)
	if err != nil {
		return err
	}
	if err := credentials.EnsureConfigDirectory(configDirectory); err != nil {
		return err
	}

	loginArguments := []string{"auth", "login"}
	if !containsHostnameOption(options.LoginArguments) {
		loginArguments = append(loginArguments, "--hostname", "github.com")
	}
	loginArguments = append(loginArguments, options.LoginArguments...)
	if login == nil {
		login = LoginExecutor(ghrouterexec.RunInteractive)
	}
	if err := login(realGH, loginArguments, credentials.EnvironmentForDirectory(configDirectory, environ)); err != nil {
		return fmt.Errorf("authenticate account %s: %w", account, err)
	}

	configuration.SetAccountConfig(account, configReference)
	if !configurationExists {
		configuration.Default = account
	}
	if err := store.Save(configuration); err != nil {
		return err
	}

	if writer == nil {
		writer = os.Stdout
	}
	fmt.Fprintf(writer, "Account configured: %s\n", account)
	fmt.Fprintf(writer, "GitHub CLI config: %s\n", configReference)
	return nil
}

func setupOptionValue(arguments []string, index int, option string) (string, int, error) {
	if index+1 >= len(arguments) || strings.TrimSpace(arguments[index+1]) == "" {
		return "", index, fmt.Errorf("%s requires a value", option)
	}
	return strings.TrimSpace(arguments[index+1]), index + 1, nil
}

func containsHostnameOption(arguments []string) bool {
	for _, argument := range arguments {
		if argument == "--hostname" || strings.HasPrefix(argument, "--hostname=") {
			return true
		}
	}
	return false
}

func loginOptionTakesValue(argument string) bool {
	switch argument {
	case "--hostname", "--git-protocol":
		return true
	default:
		return false
	}
}
