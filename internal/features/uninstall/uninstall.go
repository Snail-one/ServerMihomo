package uninstall

import (
	"context"

	"snailproxy/internal/feature"
)

type Feature struct{}

func (Feature) ID() string {
	return "uninstall"
}

func (Feature) Label() string {
	return "卸载"
}

func (Feature) Order() int {
	return 40
}

func (Feature) Run(ctx context.Context, runtime feature.Runtime) error {
	manager, err := runtime.NewPlatformManager()
	if err != nil {
		return err
	}
	return manager.Uninstall(ctx)
}
