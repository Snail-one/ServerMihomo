package resources

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestReleaseMihomoBundleFromFSCopiesFilesIntoTargetRoot(t *testing.T) {
	targetDir := t.TempDir()
	source := fstest.MapFS{
		"mihomo/.gitkeep": {
			Data: []byte{},
		},
		"mihomo/.gitignore": {
			Data: []byte("*\n"),
		},
		"mihomo/packages/manifest.json": {
			Data: []byte("{}\n"),
		},
		"mihomo/config.yaml": {
			Data: []byte("mode: rule\n"),
		},
		"mihomo/metacubexd/index.html": {
			Data: []byte("<html></html>\n"),
		},
	}

	result, err := releaseMihomoBundleFromFS(source, ReleaseOptions{TargetDir: targetDir})
	if err != nil {
		t.Fatalf("releaseMihomoBundleFromFS() error = %v", err)
	}

	assertFileContent(t, filepath.Join(targetDir, "config.yaml"), "mode: rule\n")
	assertFileContent(t, filepath.Join(targetDir, "metacubexd", "index.html"), "<html></html>\n")
	if len(result.Released) != 2 {
		t.Fatalf("Released = %#v, want 2 files", result.Released)
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".gitkeep")); !os.IsNotExist(err) {
		t.Fatalf(".gitkeep should not be released, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf(".gitignore should not be released, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "packages", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("packages/manifest.json should not be released, stat err = %v", err)
	}
}

func TestReleaseMihomoBundleFromFSSkipsExistingFiles(t *testing.T) {
	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "config.yaml")
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	source := fstest.MapFS{
		"mihomo/config.yaml": {
			Data: []byte("new\n"),
		},
	}

	result, err := releaseMihomoBundleFromFS(source, ReleaseOptions{TargetDir: targetDir})
	if err != nil {
		t.Fatalf("releaseMihomoBundleFromFS() error = %v", err)
	}

	assertFileContent(t, targetPath, "existing\n")
	if len(result.Released) != 0 {
		t.Fatalf("Released = %#v, want no released files", result.Released)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != targetPath {
		t.Fatalf("Skipped = %#v, want %s", result.Skipped, targetPath)
	}
}

func TestReleaseMihomoBundleFromFSOverwritesExistingFiles(t *testing.T) {
	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "config.yaml")
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	source := fstest.MapFS{
		"mihomo/config.yaml": {
			Data: []byte("new\n"),
		},
	}

	result, err := releaseMihomoBundleFromFS(source, ReleaseOptions{TargetDir: targetDir, Overwrite: true})
	if err != nil {
		t.Fatalf("releaseMihomoBundleFromFS() error = %v", err)
	}

	assertFileContent(t, targetPath, "new\n")
	if len(result.Released) != 1 || result.Released[0] != targetPath {
		t.Fatalf("Released = %#v, want %s", result.Released, targetPath)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("Skipped = %#v, want none", result.Skipped)
	}
}

func assertFileContent(t *testing.T, targetPath string, want string) {
	t.Helper()
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", targetPath, err)
	}
	if string(data) != want {
		t.Fatalf("ReadFile(%s) = %q, want %q", targetPath, data, want)
	}
}
