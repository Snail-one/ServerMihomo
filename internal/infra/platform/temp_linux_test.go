//go:build linux

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupInstallTemporaryFilesRemovesInstallTempDir(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)

	tempFile := filepath.Join(installTempDir(), "mihomo")
	if err := os.MkdirAll(filepath.Dir(tempFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(install temp dir) error = %v", err)
	}
	if err := os.WriteFile(tempFile, []byte("extracted binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(install temp file) error = %v", err)
	}

	if err := CleanupInstallTemporaryFiles(); err != nil {
		t.Fatalf("CleanupInstallTemporaryFiles() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "mihomo-install")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("install temp dir exists after cleanup or stat failed with unexpected error: %v", err)
	}
}
