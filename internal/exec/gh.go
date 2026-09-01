package ghexec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/credentials"
	"github.com/SamCullin/gh-router/internal/routing"
)

func FindRealGH(argv0 string, environ map[string]string) (string, error) {
	if configured, found := environmentValue(environ, "GH_ROUTER_REAL_GH"); found && strings.TrimSpace(configured) != "" {
		path, err := osexec.LookPath(configured)
		if err != nil {
			return "", fmt.Errorf("GH_ROUTER_REAL_GH does not point to an executable: %s", configured)
		}
		return path, nil
	}

	routerPath := resolvedPath(argv0)
	pathValue, _ := environmentValue(environ, "PATH")
	for _, directory := range strings.Split(pathValue, string(os.PathListSeparator)) {
		if directory == "" {
			continue
		}
		candidate := filepath.Join(directory, "gh")
		if isExecutable(candidate) && resolvedPath(candidate) != routerPath {
			return candidate, nil
		}
	}
	for _, candidate := range []string{"/opt/homebrew/bin/gh", "/usr/local/bin/gh", "/usr/bin/gh"} {
		if isExecutable(candidate) && resolvedPath(candidate) != routerPath {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to find the real gh executable; set GH_ROUTER_REAL_GH")
}

func Run(arguments []string, configuration config.Config, resolution routing.AccountResolution, argv0 string, environ map[string]string) error {
	realGH, err := FindRealGH(argv0, environ)
	if err != nil {
		return err
	}
	token, err := credentials.TokenFor(configuration, resolution.Account, realGH, environ, "github.com")
	if err != nil {
		return err
	}
	childEnvironment, err := credentials.EnvironmentFor(configuration, resolution.Account, token, environ)
	if err != nil {
		return err
	}
	return Execute(realGH, arguments, childEnvironment)
}

func Execute(realGH string, arguments []string, environ map[string]string) error {
	argv := append([]string{realGH}, arguments...)
	if environ == nil {
		environ = environmentMap()
	}
	return syscall.Exec(realGH, argv, environmentSlice(environ))
}

func environmentValue(environ map[string]string, key string) (string, bool) {
	if environ != nil {
		value, ok := environ[key]
		return value, ok
	}
	return os.LookupEnv(key)
}

func environmentMap() map[string]string {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

func environmentSlice(environ map[string]string) []string {
	result := make([]string, 0, len(environ))
	for key, value := range environ {
		result = append(result, key+"="+value)
	}
	return result
}

func resolvedPath(path string) string {
	if path == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	if absolute, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(path)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}
