package commands

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/SamCullin/gh-router/internal/override"
)

type OverrideOptions struct {
	Action     string
	TargetPath string
}

func ParseOverrideOptions(arguments []string) (OverrideOptions, error) {
	if len(arguments) == 0 {
		return OverrideOptions{}, fmt.Errorf("override requires install, status, or uninstall")
	}
	options := OverrideOptions{Action: arguments[0]}
	pathSet := false
	for index := 1; index < len(arguments); index++ {
		argument := arguments[index]
		switch {
		case argument == "--path":
			if pathSet {
				return OverrideOptions{}, fmt.Errorf("--path may only be specified once")
			}
			if index+1 >= len(arguments) || strings.TrimSpace(arguments[index+1]) == "" {
				return OverrideOptions{}, fmt.Errorf("--path requires a path")
			}
			options.TargetPath = arguments[index+1]
			pathSet = true
			index++
		case strings.HasPrefix(argument, "--path="):
			if pathSet {
				return OverrideOptions{}, fmt.Errorf("--path may only be specified once")
			}
			options.TargetPath = strings.TrimSpace(strings.TrimPrefix(argument, "--path="))
			if options.TargetPath == "" {
				return OverrideOptions{}, fmt.Errorf("--path requires a path")
			}
			pathSet = true
		default:
			return OverrideOptions{}, fmt.Errorf("unknown override option: %s", argument)
		}
	}

	switch options.Action {
	case "install", "status", "uninstall":
		return options, nil
	default:
		return OverrideOptions{}, fmt.Errorf("unsupported override command: %s", options.Action)
	}
}

func Override(writer io.Writer, arguments []string, argv0 string) error {
	options, err := ParseOverrideOptions(arguments)
	if err != nil {
		return err
	}
	if writer == nil {
		writer = os.Stdout
	}
	targetPath := options.TargetPath
	if targetPath == "" {
		targetPath, err = override.FindTarget()
		if err != nil {
			return err
		}
	}
	routerPath, err := currentExecutable(argv0)
	if err != nil {
		return fmt.Errorf("find router executable: %w", err)
	}

	switch options.Action {
	case "install":
		status, err := override.Install(targetPath, routerPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "gh override installed at %s\n", status.TargetPath)
		fmt.Fprintf(writer, "Native gh preserved at %s\n", status.NativePath)
	case "status":
		status, err := override.Inspect(targetPath, routerPath)
		if err != nil {
			return err
		}
		printOverrideStatus(writer, status)
	case "uninstall":
		status, err := override.Uninstall(targetPath, routerPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "gh override removed from %s\n", status.TargetPath)
		fmt.Fprintf(writer, "Native gh restored at %s\n", status.TargetPath)
	}
	return nil
}

func printOverrideStatus(writer io.Writer, status override.Status) {
	state := "not installed"
	if status.Installed {
		state = "installed"
	}
	fmt.Fprintf(writer, "gh override: %s\n", state)
	fmt.Fprintf(writer, "Target: %s\n", status.TargetPath)
	if status.RouterPath != "" {
		fmt.Fprintf(writer, "Router: %s\n", status.RouterPath)
	}
	if status.NativePath != "" {
		fmt.Fprintf(writer, "Native backup: %s\n", status.NativePath)
	}
}

func currentExecutable(argv0 string) (string, error) {
	if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
		return filepath.Abs(executable)
	}
	if strings.ContainsRune(argv0, os.PathSeparator) {
		return filepath.Abs(argv0)
	}
	return osexec.LookPath(argv0)
}
