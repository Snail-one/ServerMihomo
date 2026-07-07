package uninstall

import (
	"context"

	"snailproxy/internal/platform"
)

func Run(ctx context.Context) error {
	manager, err := platform.NewManager()
	if err != nil {
		return err
	}
	return manager.Uninstall(ctx)
}
