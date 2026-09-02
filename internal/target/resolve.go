package target

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Source string

const (
	SourceCommand      Source = "command"
	SourceEnvironment  Source = "environment"
	SourceRepository   Source = "repository"
	SourceOrganisation Source = "organisation"
	SourceDefault      Source = "default"
)

type Target struct {
	Repository   string
	Organisation string
	Directory    string
	Source       Source
}

var repoPartPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func Resolve(arguments []string, environ map[string]string, cwd string) (Target, error) {
	directory := RepositoryRoot(cwd)
	repository, found, err := explicitRepository(arguments)
	if err != nil {
		return Target{}, err
	}
	if found {
		return Target{Repository: repository, Directory: directory, Source: SourceCommand}, nil
	}

	repository, found = apiRepository(arguments)
	if found {
		return Target{Repository: repository, Directory: directory, Source: SourceCommand}, nil
	}

	if repository, found = environmentValue(environ, "GH_REPO"); found && strings.TrimSpace(repository) != "" {
		parsed := parseRepoReference(repository)
		if parsed == "" {
			return Target{}, fmt.Errorf("GH_REPO must use OWNER/REPO form")
		}
		return Target{Repository: parsed, Directory: directory, Source: SourceEnvironment}, nil
	}

	if repository := remoteRepository(cwd); repository != "" {
		return Target{Repository: repository, Directory: directory, Source: SourceRepository}, nil
	}

	organisation, found, err := optionValue(arguments, "--org", "--owner")
	if err != nil {
		return Target{}, err
	}
	if found && strings.TrimSpace(organisation) != "" {
		return Target{Organisation: organisation, Directory: directory, Source: SourceOrganisation}, nil
	}
	return Target{Directory: directory, Source: SourceDefault}, nil
}

func explicitRepository(arguments []string) (string, bool, error) {
	value, found, err := optionValue(arguments, "-R", "--repo")
	if err != nil {
		return "", false, err
	}
	if found {
		parsed := parseRepoReference(value)
		if parsed == "" {
			return "", false, fmt.Errorf("repository target must use OWNER/REPO form")
		}
		return parsed, true, nil
	}
	for _, argument := range arguments {
		if strings.HasPrefix(argument, "-R") && argument != "-R" {
			parsed := parseRepoReference(strings.TrimPrefix(argument, "-R"))
			if parsed != "" {
				return parsed, true, nil
			}
			return "", false, fmt.Errorf("repository target must use OWNER/REPO form")
		}
	}
	for index, argument := range arguments {
		if strings.HasPrefix(argument, "-") || (index > 0 && (arguments[index-1] == "-R" || arguments[index-1] == "--repo")) {
			continue
		}
		if strings.HasPrefix(argument, "repos/") {
			if parsed := parseRepoReference(argument); parsed != "" {
				return parsed, true, nil
			}
		}
		if strings.Count(argument, "/") == 1 {
			if parsed := parseRepoReference(argument); parsed != "" {
				return parsed, true, nil
			}
		}
	}
	return "", false, nil
}

func apiRepository(arguments []string) (string, bool) {
	for _, argument := range arguments {
		if strings.HasPrefix(argument, "repos/") || strings.HasPrefix(argument, "/repos/") {
			if parsed := parseRepoReference(argument); parsed != "" {
				return parsed, true
			}
		}
	}
	return "", false
}

func optionValue(arguments []string, options ...string) (string, bool, error) {
	for index, argument := range arguments {
		for _, option := range options {
			if argument == option {
				if index+1 >= len(arguments) || strings.TrimSpace(arguments[index+1]) == "" {
					return "", false, fmt.Errorf("%s requires a value", option)
				}
				return arguments[index+1], true, nil
			}
			if strings.HasPrefix(argument, option+"=") {
				value := strings.TrimPrefix(argument, option+"=")
				if strings.TrimSpace(value) == "" {
					return "", false, fmt.Errorf("%s requires a value", option)
				}
				return value, true, nil
			}
		}
	}
	return "", false, nil
}

func parseRepoReference(value string) string {
	candidate := strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(candidate, "repos/"):
		candidate = strings.TrimPrefix(candidate, "repos/")
	case strings.HasPrefix(candidate, "/repos/"):
		candidate = strings.TrimPrefix(candidate, "/repos/")
	case strings.HasPrefix(candidate, "git@github.com:"):
		candidate = strings.TrimPrefix(candidate, "git@github.com:")
	case strings.HasPrefix(candidate, "ssh://git@github.com/"):
		candidate = strings.TrimPrefix(candidate, "ssh://git@github.com/")
	case strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "http://"):
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Hostname() != "github.com" {
			return ""
		}
		candidate = strings.TrimPrefix(parsed.Path, "/")
	case strings.HasPrefix(candidate, "github.com/"):
		candidate = strings.TrimPrefix(candidate, "github.com/")
	}
	parts := strings.Split(strings.Trim(candidate, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	owner := parts[0]
	repository := strings.TrimSuffix(parts[1], ".git")
	if !repoPartPattern.MatchString(owner) || !repoPartPattern.MatchString(repository) {
		return ""
	}
	return owner + "/" + repository
}

func remoteRepository(cwd string) string {
	if strings.TrimSpace(cwd) == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	command := exec.Command("git", "-C", cwd, "remote", "get-url", "origin")
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return parseRepoReference(strings.TrimSpace(string(output)))
}

func environmentValue(environ map[string]string, key string) (string, bool) {
	if environ != nil {
		value, ok := environ[key]
		return value, ok
	}
	return os.LookupEnv(key)
}

func RepositoryRoot(cwd string) string {
	if strings.TrimSpace(cwd) == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	return root
}
