//go:build !linux && !windows

package platform

import (
	"fmt"
	"runtime"
)

func NewInstaller() (Installer, error) {
	return nil, fmt.Errorf("暂不支持当前系统: %s", runtime.GOOS)
}
