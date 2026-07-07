package service

import (
	"context"
	"fmt"

	"snailproxy/internal/feature"
)

type Feature struct{}

func (Feature) ID() string {
	return "service"
}

func (Feature) Label() string {
	return "mihomo 服务与代理"
}

func (Feature) Order() int {
	return 30
}

func (Feature) Run(ctx context.Context, runtime feature.Runtime) error {
	action, err := Select()
	if err != nil {
		return err
	}
	if action == ActionReturn {
		fmt.Println("已返回。")
		return feature.ErrReturn
	}

	manager, err := runtime.NewPlatformManager()
	if err != nil {
		return err
	}

	switch action {
	case ActionStart:
		return manager.StartService(ctx)
	case ActionRestart:
		return manager.RestartService(ctx)
	case ActionStop:
		return manager.StopService(ctx)
	case ActionWriteProxyEnv:
		return manager.WriteProxyEnvironment(ctx)
	case ActionClearProxyEnv:
		return manager.ClearProxyEnvironment(ctx)
	default:
		return fmt.Errorf("未知 mihomo 服务操作: %d", action)
	}
}
