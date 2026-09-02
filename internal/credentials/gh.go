package credentials

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/SamCullin/gh-router/internal/config"
)

func ConfigDirectory(configuration config.Config, account string) (string, error) {
	accountConfiguration, ok := configuration.Account(account)
	if !ok || strings.TrimSpace(accountConfiguration.ConfigDir) == "" {
		return "", fmt.Errorf("account %q needs accounts.%s.config_dir before it can be used", account, account)
	}
	directory, err := config.ExpandPath(accountConfiguration.ConfigDir)
	if err != nil {
		return "", fmt.Errorf("resolve config directory for %s: %w", account, err)
	}
	return directory, nil
}

func EnsureConfigDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create GitHub CLI config directory: %w", err)
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return fmt.Errorf("secure GitHub CLI config directory: %w", err)
	}
	return nil
}

func EnvironmentForDirectory(directory string, environ map[string]string) map[string]string {
	result := withoutInheritedTokensMap(environ)
	result["GH_CONFIG_DIR"] = directory
	return result
}

func TokenFor(configuration config.Config, account, realGH string, environ map[string]string, hostname string) (string, error) {
	directory, err := ConfigDirectory(configuration, account)
	if err != nil {
		return "", err
	}
	command := exec.Command(realGH, "auth", "token", "--hostname", hostname)
	command.Env = withoutInheritedTokens(environ)
	command.Env = setEnvironment(command.Env, "GH_CONFIG_DIR", directory)
	var output bytes.Buffer
	var errors bytes.Buffer
	command.Stdout = &output
	command.Stderr = &errors
	if err := command.Run(); err != nil {
		detail := strings.TrimSpace(errors.String())
		if detail == "" {
			detail = "no token was returned"
		}
		return "", fmt.Errorf("unable to load credentials for %s: %s", account, detail)
	}
	token := strings.TrimSpace(output.String())
	if token == "" {
		return "", fmt.Errorf("unable to load credentials for %s: no token was returned", account)
	}
	return token, nil
}

func EnvironmentFor(configuration config.Config, account, token string, environ map[string]string) (map[string]string, error) {
	directory, err := ConfigDirectory(configuration, account)
	if err != nil {
		return nil, err
	}
	result := EnvironmentForDirectory(directory, environ)
	result["GH_TOKEN"] = token
	return result, nil
}

func withoutInheritedTokens(environ map[string]string) []string {
	return environmentSlice(withoutInheritedTokensMap(environ))
}

func withoutInheritedTokensMap(environ map[string]string) map[string]string {
	result := cloneEnvironment(environ)
	delete(result, "GH_TOKEN")
	delete(result, "GITHUB_TOKEN")
	return result
}

func cloneEnvironment(environ map[string]string) map[string]string {
	if environ != nil {
		result := make(map[string]string, len(environ))
		for key, value := range environ {
			result[key] = value
		}
		return result
	}
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

func setEnvironment(environ []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		result = append(result, entry)
	}
	return append(result, prefix+value)
}

func environmentSlice(environ map[string]string) []string {
	result := make([]string, 0, len(environ))
	for key, value := range environ {
		result = append(result, key+"="+value)
	}
	return result
}
