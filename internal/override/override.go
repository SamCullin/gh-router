package override

import (
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
)

const NativeBackupSuffix = ".gh-router-real"

var ErrNotInstalled = errors.New("gh-router override is not installed")

type Status struct {
	TargetPath string
	BackupPath string
	RouterPath string
	NativePath string
	Installed  bool
}

func NativeBackupPath(targetPath string) string {
	return targetPath + NativeBackupSuffix
}

func FindTarget() (string, error) {
	targetPath, err := osexec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("find gh on PATH: %w", err)
	}
	return normalisePath(targetPath)
}

func Install(targetPath, routerPath string) (Status, error) {
	targetPath, err := normalisePath(targetPath)
	if err != nil {
		return Status{}, err
	}
	routerPath, err = normalisePath(routerPath)
	if err != nil {
		return Status{}, err
	}
	if !isExecutable(routerPath) {
		return Status{}, fmt.Errorf("router executable is not executable: %s", routerPath)
	}
	if _, err := os.Lstat(targetPath); err != nil {
		return Status{}, fmt.Errorf("inspect gh target %s: %w", targetPath, err)
	}

	backupPath := NativeBackupPath(targetPath)
	targetResolved := resolvedPath(targetPath)
	routerResolved := resolvedPath(routerPath)
	_, backupErr := os.Lstat(backupPath)
	if targetResolved == routerResolved {
		if backupErr != nil {
			return Status{}, fmt.Errorf("gh already points to the router but native gh backup is missing: %s", backupPath)
		}
		if !isExecutable(backupPath) {
			return Status{}, fmt.Errorf("native gh backup is not executable: %s", backupPath)
		}
		return Inspect(targetPath, routerPath)
	}
	if backupErr == nil {
		return Status{}, fmt.Errorf("refusing to overwrite existing backup: %s", backupPath)
	}
	if !errors.Is(backupErr, os.ErrNotExist) {
		return Status{}, fmt.Errorf("inspect native gh backup %s: %w", backupPath, backupErr)
	}
	if !isExecutable(targetPath) {
		return Status{}, fmt.Errorf("gh target is not executable: %s", targetPath)
	}

	temporaryPath := fmt.Sprintf("%s.gh-router-tmp-%d", targetPath, os.Getpid())
	if _, err := os.Lstat(temporaryPath); err == nil {
		return Status{}, fmt.Errorf("temporary override path already exists: %s", temporaryPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, fmt.Errorf("inspect temporary override path %s: %w", temporaryPath, err)
	}
	if err := os.Symlink(routerPath, temporaryPath); err != nil {
		return Status{}, fmt.Errorf("create router link %s: %w", temporaryPath, err)
	}

	if err := os.Rename(targetPath, backupPath); err != nil {
		_ = os.Remove(temporaryPath)
		return Status{}, fmt.Errorf("preserve native gh at %s: %w", backupPath, err)
	}
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		restoreErr := os.Rename(backupPath, targetPath)
		if restoreErr != nil {
			return Status{}, fmt.Errorf("install router link at %s: %w; restore native gh: %v", targetPath, err, restoreErr)
		}
		return Status{}, fmt.Errorf("install router link at %s: %w", targetPath, err)
	}

	return Inspect(targetPath, routerPath)
}

func Uninstall(targetPath, routerPath string) (Status, error) {
	targetPath, err := normalisePath(targetPath)
	if err != nil {
		return Status{}, err
	}
	routerPath, err = normalisePath(routerPath)
	if err != nil {
		return Status{}, err
	}
	status, err := Inspect(targetPath, routerPath)
	if err != nil {
		return Status{}, err
	}
	if !status.Installed {
		return status, ErrNotInstalled
	}
	targetInfo, err := os.Lstat(targetPath)
	if err != nil {
		return Status{}, fmt.Errorf("inspect gh target %s: %w", targetPath, err)
	}
	if targetInfo.Mode()&os.ModeSymlink == 0 {
		return Status{}, fmt.Errorf("refusing to remove non-symlink gh target: %s", targetPath)
	}

	temporaryPath := fmt.Sprintf("%s.gh-router-tmp-%d", targetPath, os.Getpid())
	if _, err := os.Lstat(temporaryPath); err == nil {
		return Status{}, fmt.Errorf("temporary override path already exists: %s", temporaryPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, fmt.Errorf("inspect temporary override path %s: %w", temporaryPath, err)
	}
	if err := os.Rename(targetPath, temporaryPath); err != nil {
		return Status{}, fmt.Errorf("remove router link %s: %w", targetPath, err)
	}
	if err := os.Rename(status.BackupPath, targetPath); err != nil {
		restoreErr := os.Rename(temporaryPath, targetPath)
		if restoreErr != nil {
			return Status{}, fmt.Errorf("restore native gh at %s: %w; restore router link: %v", targetPath, err, restoreErr)
		}
		return Status{}, fmt.Errorf("restore native gh at %s: %w", targetPath, err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return Status{}, fmt.Errorf("remove temporary router link %s: %w", temporaryPath, err)
	}

	return Inspect(targetPath, routerPath)
}

func Inspect(targetPath, routerPath string) (Status, error) {
	targetPath, err := normalisePath(targetPath)
	if err != nil {
		return Status{}, err
	}
	status := Status{
		TargetPath: targetPath,
		BackupPath: NativeBackupPath(targetPath),
	}
	if strings.TrimSpace(routerPath) != "" {
		status.RouterPath, err = normalisePath(routerPath)
		if err != nil {
			return Status{}, err
		}
	}
	if _, err := os.Lstat(status.BackupPath); err == nil {
		status.NativePath = status.BackupPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, fmt.Errorf("inspect native gh backup %s: %w", status.BackupPath, err)
	}
	targetInfo, err := os.Lstat(status.TargetPath)
	if errors.Is(err, os.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("inspect gh target %s: %w", status.TargetPath, err)
	}
	if status.RouterPath == "" || targetInfo.Mode()&os.ModeSymlink == 0 || status.NativePath == "" {
		return status, nil
	}
	status.Installed = resolvedPath(status.TargetPath) == resolvedPath(status.RouterPath) && isExecutable(status.NativePath)
	return status, nil
}

func normalisePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func resolvedPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0
}
