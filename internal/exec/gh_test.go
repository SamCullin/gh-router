package ghexec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SamCullin/gh-router/internal/override"
)

func TestFindRealGHUsesNativeBackupForInstalledOverride(t *testing.T) {
	directory := t.TempDir()
	routerPath := filepath.Join(directory, "gh-router")
	targetPath := filepath.Join(directory, "gh")
	nativePath := filepath.Join(directory, "native-gh")
	writeExecutable(t, routerPath)
	writeExecutable(t, nativePath)
	if err := os.Rename(nativePath, override.NativeBackupPath(targetPath)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(routerPath, targetPath); err != nil {
		t.Fatal(err)
	}

	got, err := FindRealGH(routerPath, map[string]string{"PATH": directory})
	if err != nil {
		t.Fatal(err)
	}
	if got != override.NativeBackupPath(targetPath) {
		t.Fatalf("FindRealGH() = %s, want %s", got, override.NativeBackupPath(targetPath))
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("gh"), 0755); err != nil {
		t.Fatal(err)
	}
}
