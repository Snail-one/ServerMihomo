package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"snailproxy/internal/domain/mihomo"
	"snailproxy/internal/feature"
	"snailproxy/internal/infra/platform"
	"snailproxy/internal/terminal"
	"snailproxy/internal/version"
)

type appRuntime struct{}

func (appRuntime) Terminal() terminal.Terminal {
	return terminal.Default()
}

func (appRuntime) NewMihomoStore() mihomo.Store {
	return mihomo.NewStore()
}

func (appRuntime) NewPlatformManager() (platform.Manager, error) {
	return platform.NewManager()
}

type mainMenuAction struct {
	feature feature.Feature
	exit    bool
}

func Run(ctx context.Context, args []string, registry feature.Registry) error {
	if handleVersionArg(args) {
		fmt.Println(version.Info())
		return nil
	}

	fmt.Printf("当前 Linux 系统: linux/%s\n", runtime.GOARCH)

	if err := platform.RequireSudo(); err != nil {
		return err
	}

	runtimeEnv := appRuntime{}
	for {
		action, err := selectMainMenu(registry)
		if err != nil {
			return err
		}
		if action.exit {
			fmt.Println("已退出。")
			return nil
		}

		if err := runSelectedFeature(ctx, runtimeEnv, action.feature); err != nil {
			return err
		}
	}
}

func runSelectedFeature(ctx context.Context, runtimeEnv feature.Runtime, selected feature.Feature) error {
	if err := selected.Run(ctx, runtimeEnv); err != nil {
		if errors.Is(err, feature.ErrReturn) {
			return nil
		}
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("操作完成。")
	}

	fmt.Println()
	if err := terminal.Pause("按 Enter 返回主菜单..."); err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func selectMainMenu(registry feature.Registry) (mainMenuAction, error) {
	features := registry.Features()
	options := make([]terminal.MenuOption[mainMenuAction], 0, len(features)+1)
	for i, feature := range features {
		options = append(options, terminal.MenuOption[mainMenuAction]{
			Number: i + 1,
			Label:  feature.Label(),
			Value:  mainMenuAction{feature: feature},
		})
	}
	options = append(options, terminal.MenuOption[mainMenuAction]{
		Number: 0,
		Label:  "退出",
		Value:  mainMenuAction{exit: true},
	})

	return terminal.Select("主菜单:", mainMenuPromptRange(len(features)), options)
}

func mainMenuPromptRange(featureCount int) string {
	if featureCount == 0 {
		return "[0]"
	}
	return fmt.Sprintf("[0-%d]", featureCount)
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
