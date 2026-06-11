//go:build windows

package windows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"snailproxy/internal/archive"
)

type Installer struct{}

func (i *Installer) PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error {
	installDir := filepath.Join(os.Getenv("ProgramData"), "mihomo")
	if installDir == "mihomo" {
		installDir = filepath.Join(os.TempDir(), "mihomo")
	}

	binaryPath, err := archive.ExtractMihomoBinary(archivePath, assetName, installDir)
	if err != nil {
		return fmt.Errorf("安装 mihomo 失败: %w", err)
	}

	fmt.Printf("mihomo Windows 安装完成: %s\n", binaryPath)
	fmt.Println("Windows 服务安装逻辑预留，后续可以在 internal/platform/windows 扩展。")
	return nil
}

func (i *Installer) InstallService(ctx context.Context) error {
	return fmt.Errorf("Windows 暂未支持安装 mihomo 服务")
}
