//go:build linux

package platform

import (
	"snailproxy/internal/platform/linux"
)

func NewManager() (Manager, error) {
	return linux.New(), nil
}
