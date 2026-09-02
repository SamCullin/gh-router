package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("configuration file not found")

type Account struct {
	ConfigDir string `yaml:"config_dir,omitempty"`
}

func (account *Account) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		account.ConfigDir = strings.TrimSpace(node.Value)
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("account configuration must be a path or mapping")
	}
	var value struct {
		ConfigDir   string `yaml:"config_dir"`
		GHConfigDir string `yaml:"gh_config_dir"`
	}
	if err := node.Decode(&value); err != nil {
		return err
	}
	account.ConfigDir = strings.TrimSpace(value.ConfigDir)
	if account.ConfigDir == "" {
		account.ConfigDir = strings.TrimSpace(value.GHConfigDir)
	}
	return nil
}

type Rule struct {
	Account string `yaml:"account"`
}

func (rule *Rule) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		rule.Account = strings.TrimSpace(node.Value)
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("routing rule must be an account name or mapping")
	}
	var value struct {
		Account string `yaml:"account"`
	}
	if err := node.Decode(&value); err != nil {
		return err
	}
	rule.Account = strings.TrimSpace(value.Account)
	return nil
}

type ImplicitOrg struct {
	Account string `yaml:"account"`
	Source  string `yaml:"source"`
}

type Config struct {
	Default      string                 `yaml:"default"`
	Accounts     map[string]Account     `yaml:"accounts,omitempty"`
	Orgs         map[string]Rule        `yaml:"orgs,omitempty"`
	Repos        map[string]Rule        `yaml:"repos,omitempty"`
	ImplicitOrgs map[string]ImplicitOrg `yaml:"implicit_orgs,omitempty"`
}

func New(defaultAccount string) Config {
	return Config{
		Default:      strings.TrimSpace(defaultAccount),
		Accounts:     map[string]Account{},
		Orgs:         map[string]Rule{},
		Repos:        map[string]Rule{},
		ImplicitOrgs: map[string]ImplicitOrg{},
	}
}

func DefaultPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("GH_ROUTER_CONFIG")); configured != "" {
		return ExpandPath(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gh-router", "config.yaml"), nil
}

func ManagedAccountConfigDir(account string) (string, error) {
	name, err := ValidateAccountName(account)
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gh-router", "accounts", name), nil
}

func ManagedAccountConfigReference(account string) (string, error) {
	name, err := ValidateAccountName(account)
	if err != nil {
		return "", err
	}
	return filepath.Join("~", ".config", "gh-router", "accounts", name), nil
}

func ValidateAccountName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("account name cannot be empty")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("account name cannot contain path separators")
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("account name cannot contain control characters")
	}
	return name, nil
}

func ExpandPath(value string) (string, error) {
	value = os.ExpandEnv(strings.TrimSpace(value))
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if value == "~" {
			return home, nil
		}
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	return filepath.Clean(value), nil
}

func Load(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("%w at %s", ErrNotFound, path)
		}
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var configuration Config
	if err := yaml.Unmarshal(contents, &configuration); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate %s: %w", path, err)
	}
	return configuration, nil
}

type Store struct {
	Path string
}

func NewStore(path string) Store {
	return Store{Path: path}
}

func (store Store) Load() (Config, error) {
	return Load(store.Path)
}

func (store Store) Save(configuration Config) error {
	if err := configuration.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	parent := filepath.Dir(store.Path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	temporary, err := os.CreateTemp(parent, ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0600); err != nil {
		return fmt.Errorf("set configuration permissions: %w", err)
	}
	encoder := yaml.NewEncoder(temporary)
	encoder.SetIndent(2)
	if err := encoder.Encode(configuration); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("write configuration: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(temporaryPath, store.Path); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	removeTemporary = false
	return nil
}

func (configuration Config) Validate() error {
	if strings.TrimSpace(configuration.Default) == "" {
		return fmt.Errorf("default must name an account")
	}
	for name, account := range configuration.Accounts {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("account name cannot be empty")
		}
		if strings.ContainsAny(name, "\r\n") || strings.ContainsAny(account.ConfigDir, "\r\n") {
			return fmt.Errorf("account values cannot contain newlines")
		}
	}
	for name, rule := range configuration.Orgs {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(rule.Account) == "" {
			return fmt.Errorf("organisation rules require a name and account")
		}
	}
	for name, rule := range configuration.Repos {
		if _, err := NormalizeRepo(name); err != nil {
			return err
		}
		if strings.TrimSpace(rule.Account) == "" {
			return fmt.Errorf("repository %s must name an account", name)
		}
	}
	for name, rule := range configuration.ImplicitOrgs {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(rule.Account) == "" {
			return fmt.Errorf("implicit organisation rules require a name and account")
		}
		if _, err := NormalizeRepo(rule.Source); err != nil {
			return fmt.Errorf("implicit organisation %s: %w", name, err)
		}
	}
	return nil
}

func NormalizeOrg(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeRepo(value string) (string, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("repository must use OWNER/REPO form")
	}
	return strings.ToLower(strings.TrimSpace(parts[0])) + "/" + strings.ToLower(strings.TrimSpace(parts[1])), nil
}

func (configuration Config) AccountName(requested string) string {
	for name := range configuration.Accounts {
		if strings.EqualFold(name, requested) {
			return name
		}
	}
	for _, name := range configuration.AccountNames() {
		if strings.EqualFold(name, requested) {
			return name
		}
	}
	return requested
}

func (configuration Config) Account(name string) (Account, bool) {
	canonical := configuration.AccountName(name)
	account, ok := configuration.Accounts[canonical]
	return account, ok
}

func (configuration Config) AccountNames() []string {
	seen := map[string]bool{}
	for name := range configuration.Accounts {
		seen[name] = true
	}
	seen[configuration.Default] = true
	for _, rule := range configuration.Orgs {
		seen[rule.Account] = true
	}
	for _, rule := range configuration.Repos {
		seen[rule.Account] = true
	}
	for _, rule := range configuration.ImplicitOrgs {
		seen[rule.Account] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for index, name := range names {
		if name == configuration.Default {
			copy(names[1:index+1], names[0:index])
			names[0] = name
			break
		}
	}
	return names
}

func (configuration Config) RepositoryRule(repository string) (string, string, bool) {
	wanted, err := NormalizeRepo(repository)
	if err != nil {
		return "", "", false
	}
	for configured, rule := range configuration.Repos {
		candidate, candidateErr := NormalizeRepo(configured)
		if candidateErr == nil && candidate == wanted {
			return rule.Account, configured, true
		}
	}
	return "", "", false
}

func (configuration Config) OrganisationRule(organisation string) (string, string, bool) {
	wanted := NormalizeOrg(organisation)
	for configured, rule := range configuration.Orgs {
		if NormalizeOrg(configured) == wanted {
			return rule.Account, configured, true
		}
	}
	return "", "", false
}

func (configuration Config) ImplicitOrganisationRule(organisation string) (ImplicitOrg, string, bool) {
	wanted := NormalizeOrg(organisation)
	for configured, rule := range configuration.ImplicitOrgs {
		if NormalizeOrg(configured) == wanted {
			return rule, configured, true
		}
	}
	return ImplicitOrg{}, "", false
}

func (configuration *Config) SetAccountConfig(name, configDir string) {
	canonical := configuration.AccountName(name)
	if configuration.Accounts == nil {
		configuration.Accounts = map[string]Account{}
	}
	configuration.Accounts[canonical] = Account{ConfigDir: strings.TrimSpace(configDir)}
}

func (configuration *Config) SetRepositoryRule(repository, account string) error {
	if _, err := NormalizeRepo(repository); err != nil {
		return err
	}
	if configuration.Repos == nil {
		configuration.Repos = map[string]Rule{}
	}
	key := repository
	if _, configured, ok := configuration.RepositoryRule(repository); ok {
		key = configured
	}
	configuration.Repos[key] = Rule{Account: strings.TrimSpace(account)}
	return nil
}

func (configuration *Config) RemoveRepositoryRule(repository string) (string, bool) {
	if _, configured, ok := configuration.RepositoryRule(repository); ok {
		delete(configuration.Repos, configured)
		return configured, true
	}
	return "", false
}

func (configuration *Config) SetOrganisationRule(organisation, account string) {
	if configuration.Orgs == nil {
		configuration.Orgs = map[string]Rule{}
	}
	key := organisation
	if _, configured, ok := configuration.OrganisationRule(organisation); ok {
		key = configured
	}
	configuration.Orgs[key] = Rule{Account: strings.TrimSpace(account)}
}

func (configuration *Config) RemoveOrganisationRule(organisation string) (string, bool) {
	if _, configured, ok := configuration.OrganisationRule(organisation); ok {
		delete(configuration.Orgs, configured)
		return configured, true
	}
	return "", false
}

func (configuration *Config) SetImplicitOrganisationRule(organisation, account, source string) {
	if _, _, ok := configuration.ImplicitOrganisationRule(organisation); ok {
		return
	}
	if configuration.ImplicitOrgs == nil {
		configuration.ImplicitOrgs = map[string]ImplicitOrg{}
	}
	configuration.ImplicitOrgs[organisation] = ImplicitOrg{Account: strings.TrimSpace(account), Source: source}
}

func (configuration *Config) RemoveImplicitSource(repository string) {
	wanted, err := NormalizeRepo(repository)
	if err != nil {
		return
	}
	for organisation, rule := range configuration.ImplicitOrgs {
		configured, configuredErr := NormalizeRepo(rule.Source)
		if configuredErr == nil && configured == wanted {
			delete(configuration.ImplicitOrgs, organisation)
		}
	}
}
