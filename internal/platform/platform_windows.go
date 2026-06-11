//go:build windows

package platform

import (
	"context"
	"fmt"

	"snailproxy/internal/platform/windows"
)

func NewInstaller() (Installer, error) {
	return windows.New(), nil
}

func Uninstall(ctx context.Context) error {
	return fmt.Errorf("Windows 暂未支持卸载 mihomo")
}
