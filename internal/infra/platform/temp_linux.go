//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

func installTempDir() string {
	return filepath.Join(os.TempDir(), "mihomo-install")
}

func CleanupInstallTemporaryFiles() error {
	if err := os.RemoveAll(installTempDir()); err != nil {
		return fmt.Errorf("清理安装临时目录失败: %w", err)
	}
	return nil
}
