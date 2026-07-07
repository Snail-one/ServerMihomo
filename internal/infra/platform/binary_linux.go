//go:build linux

package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"snailproxy/internal/infra/archive"
)

func (m *linuxManager) PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error {
	if fileExists(installedBinary) {
		if !overwrite {
			fmt.Printf("跳过安装，保留现有程序文件: %s\n", installedBinary)
			return nil
		}
	}

	stagingDir := filepath.Join(os.TempDir(), "mihomo-install")
	binaryPath, err := archive.ExtractMihomoBinary(archivePath, assetName, stagingDir)
	if err != nil {
		return fmt.Errorf("解压 mihomo 失败: %w", err)
	}
	fmt.Printf("已解压 mihomo: %s\n", binaryPath)

	commands := [][]string{
		{"mkdir", "-p", installDir},
		{"install", "-m", "0770", binaryPath, installedBinary},
	}

	for _, command := range commands {
		if err := runCommand(ctx, command[0], command[1:]...); err != nil {
			return err
		}
	}

	fmt.Printf("mihomo 程序文件已安装到 %s。\n", installedBinary)
	return nil
}
