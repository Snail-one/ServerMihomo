//go:build linux

package platform

import (
	"context"

	"snailproxy/internal/platform/linux"
)

func NewInstaller() (Installer, error) {
	return linux.New(), nil
}

func Uninstall(ctx context.Context) error {
	return linux.New().Uninstall(ctx)
}
