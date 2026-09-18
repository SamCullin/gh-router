package override

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPreservesAndRestoresNativeSymlink(t *testing.T) {
	directory := t.TempDir()
	nativePath := filepath.Join(directory, "native-gh")
	targetPath := filepath.Join(directory, "gh")
	routerPath := filepath.Join(directory, "gh-router")
	writeExecutable(t, nativePath, "native")
	writeExecutable(t, routerPath, "router")
	if err := os.Symlink(nativePath, targetPath); err != nil {
		t.Fatal(err)
	}

	installed, err := Install(targetPath, routerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Installed || installed.NativePath != NativeBackupPath(targetPath) {
		t.Fatalf("unexpected install status: %#v", installed)
	}
	if got := symlinkTarget(t, targetPath); got != routerPath {
		t.Fatalf("target points to %s, want %s", got, routerPath)
	}
	if got := symlinkTarget(t, NativeBackupPath(targetPath)); got != nativePath {
		t.Fatalf("backup points to %s, want %s", got, nativePath)
	}

	uninstalled, err := Uninstall(targetPath, routerPath)
	if err != nil {
		t.Fatal(err)
	}
	if uninstalled.Installed {
		t.Fatalf("unexpected uninstall status: %#v", uninstalled)
	}
	if got := symlinkTarget(t, targetPath); got != nativePath {
		t.Fatalf("restored target points to %s, want %s", got, nativePath)
	}
	if _, err := os.Lstat(NativeBackupPath(targetPath)); !os.IsNotExist(err) {
		t.Fatalf("native backup still exists: %v", err)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "gh")
	routerPath := filepath.Join(directory, "gh-router")
	writeExecutable(t, targetPath, "native")
	writeExecutable(t, routerPath, "router")

	first, err := Install(targetPath, routerPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Install(targetPath, routerPath)
	if err != nil {
		t.Fatal(err)
	}
	if first.NativePath != second.NativePath || !second.Installed {
		t.Fatalf("unexpected idempotent status: first=%#v second=%#v", first, second)
	}
}

func TestInstallRefusesUnexpectedExistingBackup(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "gh")
	routerPath := filepath.Join(directory, "gh-router")
	writeExecutable(t, targetPath, "native")
	writeExecutable(t, routerPath, "router")
	writeExecutable(t, NativeBackupPath(targetPath), "unrelated")

	if _, err := Install(targetPath, routerPath); err == nil {
		t.Fatal("expected existing backup error")
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func symlinkTarget(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	return target
}
