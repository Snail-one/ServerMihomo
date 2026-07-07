package app

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"snailproxy/internal/install"
	"snailproxy/internal/platform"
	"snailproxy/internal/service"
	"snailproxy/internal/subscription"
	"snailproxy/internal/ui/mainmenu"
	"snailproxy/internal/uninstall"
	"snailproxy/internal/version"
)

func Run(ctx context.Context, args []string) error {
	if handleVersionArg(args) {
		fmt.Println(version.Info())
		return nil
	}

	fmt.Printf("当前 Linux 系统: linux/%s\n", runtime.GOARCH)

	if err := platform.RequireSudo(); err != nil {
		return err
	}

	for {
		action, err := mainmenu.Select()
		if err != nil {
			return err
		}
		if action == mainmenu.ActionExit {
			fmt.Println("已退出。")
			return nil
		}

		if err := runAction(ctx, action); err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println("操作完成。")
		}
		fmt.Println()
	}
}

func handleVersionArg(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch strings.TrimSpace(args[0]) {
	case "-v", "--version", "version":
		return true
	default:
		return false
	}
}

func runAction(ctx context.Context, action mainmenu.Action) error {
	switch action {
	case mainmenu.ActionInstall:
		return install.Run(ctx)
	case mainmenu.ActionManageSubscription:
		return subscription.Run(ctx)
	case mainmenu.ActionManageMihomoService:
		return service.Run(ctx)
	case mainmenu.ActionUninstall:
		return uninstall.Run(ctx)
	default:
		return fmt.Errorf("未知操作: %d", action)
	}
}
