package service

import (
	"context"
	"fmt"

	"snailproxy/internal/feature"
	"snailproxy/internal/infra/platform"
	"snailproxy/internal/terminal"
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
	manager, err := runtime.NewPlatformManager()
	if err != nil {
		return err
	}

	for {
		action, err := Select()
		if err != nil {
			return err
		}
		if action == ActionReturn {
			fmt.Println("已返回。")
			return feature.ErrReturn
		}

		if err := pauseAfterServiceAction(runServiceAction(ctx, manager, action)); err != nil {
			return err
		}
	}
}

func runServiceAction(ctx context.Context, manager platform.Manager, action Action) error {
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

func pauseAfterServiceAction(actionErr error) error {
	if actionErr != nil {
		fmt.Printf("错误: %v\n", actionErr)
	} else {
		fmt.Println("操作完成。")
	}

	fmt.Println()
	if err := terminal.Pause("按 Enter 返回 mihomo 服务与代理菜单..."); err != nil {
		return err
	}
	fmt.Println()
	return nil
}
