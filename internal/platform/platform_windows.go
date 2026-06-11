//go:build windows

package platform

import "snailproxy/internal/platform/windows"

func NewInstaller() (Installer, error) {
	return windows.New(), nil
}
