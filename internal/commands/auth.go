package commands

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/routing"
	targetpkg "github.com/SamCullin/gh-router/internal/target"
)

type AuthOptions struct {
	Default    string
	HasDefault bool
	Org        string
	Repo       string
	Account    string
	ConfigDir  string
}

func ParseAuthOptions(arguments []string) (AuthOptions, error) {
	var options AuthOptions
	seen := map[string]bool{}
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if !strings.HasPrefix(argument, "--") {
			return AuthOptions{}, fmt.Errorf("unsupported auth argument: %s", argument)
		}
		name, value, inline := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		if !inline {
			if index+1 >= len(arguments) {
				return AuthOptions{}, fmt.Errorf("--%s requires a value", name)
			}
			value = arguments[index+1]
			index++
		}
		if !supportedAuthOption(name) {
			return AuthOptions{}, fmt.Errorf("unsupported auth option: --%s", name)
		}
		if strings.TrimSpace(value) == "" {
			return AuthOptions{}, fmt.Errorf("--%s requires a value", name)
		}
		if seen[name] {
			return AuthOptions{}, fmt.Errorf("--%s may only be specified once", name)
		}
		seen[name] = true
		switch name {
		case "default":
			options.Default = strings.TrimSpace(value)
			options.HasDefault = true
		case "org":
			options.Org = strings.TrimSpace(value)
		case "repo":
			options.Repo = strings.TrimSpace(value)
		case "account":
			options.Account = strings.TrimSpace(value)
		case "config-dir":
			options.ConfigDir = strings.TrimSpace(value)
		}
	}
	return options, nil
}

func Set(store config.Store, arguments []string) error {
	options, err := ParseAuthOptions(arguments)
	if err != nil {
		return err
	}
	if err := options.ValidateForSet(); err != nil {
		return err
	}
	if options.HasDefault {
		configuration, loadErr := store.Load()
		if loadErr != nil {
			if !errors.Is(loadErr, config.ErrNotFound) {
				return loadErr
			}
			configuration = config.New(options.Default)
		}
		configuration.Default = options.Default
		if options.ConfigDir != "" {
			configuration.SetAccountConfig(options.Default, options.ConfigDir)
		}
		if err := store.Save(configuration); err != nil {
			return err
		}
		fmt.Printf("Default account configured: %s\n", options.Default)
		return nil
	}

	configuration, err := store.Load()
	if err != nil {
		return err
	}
	if options.ConfigDir != "" {
		configuration.SetAccountConfig(options.Account, options.ConfigDir)
	}
	if options.Org != "" {
		configuration.SetOrganisationRule(options.Org, options.Account)
		fmt.Printf("Organisation configured: %s -> %s\n", options.Org, options.Account)
	}
	if options.Repo != "" {
		if err := configuration.SetRepositoryRule(options.Repo, options.Account); err != nil {
			return err
		}
		owner := strings.SplitN(options.Repo, "/", 2)[0]
		if _, _, ok := configuration.OrganisationRule(owner); !ok {
			configuration.SetImplicitOrganisationRule(owner, options.Account, options.Repo)
		}
		fmt.Printf("Repository configured: %s -> %s\n", options.Repo, options.Account)
	}
	return store.Save(configuration)
}

func Unset(store config.Store, arguments []string) error {
	options, err := ParseAuthOptions(arguments)
	if err != nil {
		return err
	}
	if options.HasDefault || options.Account != "" || options.ConfigDir != "" {
		return fmt.Errorf("only --org and --repo can be unset")
	}
	if (options.Org == "") == (options.Repo == "") {
		return fmt.Errorf("use exactly one of --org or --repo")
	}
	configuration, err := store.Load()
	if err != nil {
		return err
	}
	if options.Org != "" {
		removed, ok := configuration.RemoveOrganisationRule(options.Org)
		if !ok {
			return fmt.Errorf("no organisation rule exists for %s", options.Org)
		}
		fmt.Printf("Organisation rule removed: %s\n", removed)
	}
	if options.Repo != "" {
		removed, ok := configuration.RemoveRepositoryRule(options.Repo)
		if !ok {
			return fmt.Errorf("no repository rule exists for %s", options.Repo)
		}
		configuration.RemoveImplicitSource(options.Repo)
		fmt.Printf("Repository rule removed: %s\n", removed)
	}
	return store.Save(configuration)
}

func Status(store config.Store, writer io.Writer, arguments []string, override string, environ map[string]string, cwd string) error {
	configuration, err := store.Load()
	if err != nil {
		return err
	}
	resolve := false
	filtered := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if argument == "--resolve" {
			resolve = true
			continue
		}
		filtered = append(filtered, argument)
	}
	if !resolve {
		RenderStatus(configuration, writer)
		return nil
	}
	resolvedTarget, err := targetpkg.Resolve(filtered, environ, cwd)
	if err != nil {
		return err
	}
	resolution, err := routing.ResolveAccount(configuration, resolvedTarget, override)
	if err != nil {
		return err
	}
	targetName := "<none>"
	if resolvedTarget.Repository != "" {
		targetName = resolvedTarget.Repository
	} else if resolvedTarget.Organisation != "" {
		targetName = "organisation:" + resolvedTarget.Organisation
	}
	source := string(resolution.Source)
	if resolution.Rule != "" {
		source += ":" + resolution.Rule
	}
	fmt.Fprintf(writer, "repository: %s\naccount:    %s\nsource:     %s\n", targetName, resolution.Account, source)
	return nil
}

func RenderStatus(configuration config.Config, writer io.Writer) {
	fmt.Fprintln(writer, "github.com credentials")
	for _, account := range configuration.AccountNames() {
		fmt.Fprintf(writer, "  ✓ %s\n", account)
	}
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Routing configuration")
	fmt.Fprintln(writer, "  Default")
	fmt.Fprintf(writer, "    %s\n", configuration.Default)
	if len(configuration.Orgs) > 0 {
		fmt.Fprintln(writer, "  Organisations")
		for _, organisation := range sortedKeys(configuration.Orgs) {
			fmt.Fprintf(writer, "    %s → %s\n", organisation, configuration.Orgs[organisation].Account)
		}
	}
	if len(configuration.Repos) > 0 {
		fmt.Fprintln(writer, "  Repositories")
		for _, repository := range sortedKeys(configuration.Repos) {
			fmt.Fprintf(writer, "    %s → %s\n", repository, configuration.Repos[repository].Account)
		}
	}
	if len(configuration.ImplicitOrgs) > 0 {
		fmt.Fprintln(writer, "  Implicit organisations")
		for _, organisation := range sortedKeys(configuration.ImplicitOrgs) {
			rule := configuration.ImplicitOrgs[organisation]
			fmt.Fprintf(writer, "    %s → %s\n", organisation, rule.Account)
			fmt.Fprintf(writer, "      established by %s\n", rule.Source)
		}
	}
}

func PrintSwitchMessage(writer io.Writer) {
	fmt.Fprintln(writer, "gh-router: account switching is automatic; no action required.")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Account selection is determined by repository and organisation configuration.")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "To override the account for a command:")
	fmt.Fprintln(writer, "  gh --account SamCullin <command>")
}

func (options AuthOptions) ValidateForSet() error {
	scopes := 0
	if options.HasDefault {
		scopes++
	}
	if options.Org != "" {
		scopes++
	}
	if options.Repo != "" {
		scopes++
	}
	if scopes != 1 {
		return fmt.Errorf("use exactly one of --default, --org, or --repo")
	}
	if options.HasDefault && (options.Account != "" || options.Org != "" || options.Repo != "") {
		return fmt.Errorf("--default cannot be combined with --org, --repo, or --account")
	}
	if !options.HasDefault && options.Account == "" {
		return fmt.Errorf("--org and --repo require --account")
	}
	if options.ConfigDir != "" && options.Default == "" && options.Account == "" {
		return fmt.Errorf("--config-dir requires an account")
	}
	if options.Repo != "" {
		if _, err := config.NormalizeRepo(options.Repo); err != nil {
			return err
		}
	}
	return nil
}

func supportedAuthOption(name string) bool {
	switch name {
	case "default", "org", "repo", "account", "config-dir":
		return true
	default:
		return false
	}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
