package downloader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupTemporaryFilesRemovesDownloadDir(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)

	tempFile := filepath.Join(downloadDir(), "mihomo.gz.download")
	if err := os.MkdirAll(filepath.Dir(tempFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(download dir) error = %v", err)
	}
	if err := os.WriteFile(tempFile, []byte("partial download"), 0o644); err != nil {
		t.Fatalf("WriteFile(download temp file) error = %v", err)
	}

	if err := CleanupTemporaryFiles(); err != nil {
		t.Fatalf("CleanupTemporaryFiles() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "mihomo")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("download temp dir exists after cleanup or stat failed with unexpected error: %v", err)
	}
}
