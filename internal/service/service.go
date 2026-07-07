package service

import (
	"context"
	"fmt"

	"snailproxy/internal/platform"
	"snailproxy/internal/ui/servicemenu"
)

func Run(ctx context.Context) error {
	action, err := servicemenu.Select()
	if err != nil {
		return err
	}
	if action == servicemenu.ActionReturn {
		fmt.Println("已返回。")
		return nil
	}

	manager, err := platform.NewManager()
	if err != nil {
		return err
	}

	switch action {
	case servicemenu.ActionStart:
		return manager.StartService(ctx)
	case servicemenu.ActionRestart:
		return manager.RestartService(ctx)
	case servicemenu.ActionStop:
		return manager.StopService(ctx)
	case servicemenu.ActionWriteProxyEnv:
		return manager.WriteProxyEnvironment(ctx)
	case servicemenu.ActionClearProxyEnv:
		return manager.ClearProxyEnvironment(ctx)
	default:
		return fmt.Errorf("未知 mihomo 服务操作: %d", action)
	}
}
