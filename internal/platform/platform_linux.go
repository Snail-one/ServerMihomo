//go:build linux

package platform

import "snailproxy/internal/platform/linux"

func NewInstaller() (Installer, error) {
	return linux.New(), nil
}
