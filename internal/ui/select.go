package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"snailproxy/internal/github"
)

type Action int

const (
	ActionExit Action = iota
	ActionDownload
	ActionInstallService
	ActionUninstall
)

func SelectLinuxAction() (Action, error) {
	fmt.Println("Linux 操作菜单:")
	fmt.Println("  1. 下载并安装 mihomo 程序文件")
	fmt.Println("  2. 创建用户并安装 mihomo systemd 服务")
	fmt.Println("  3. 卸载并清理 mihomo")
	fmt.Println("  0. 退出")

	return selectAction("[0-3]", map[string]Action{
		"1": ActionDownload,
		"2": ActionInstallService,
		"3": ActionUninstall,
		"0": ActionExit,
	})
}

func SelectWindowsAction() (Action, error) {
	fmt.Println("Windows 操作菜单:")
	fmt.Println("  1. 下载 mihomo")
	fmt.Println("  0. 退出")

	return selectAction("[0-1]", map[string]Action{
		"1": ActionDownload,
		"0": ActionExit,
	})
}

func selectAction(promptRange string, actions map[string]Action) (Action, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("请输入操作编号 %s: ", promptRange)
		line, err := reader.ReadString('\n')
		if err != nil {
			return ActionExit, fmt.Errorf("读取用户输入失败: %w", err)
		}

		if action, ok := actions[strings.TrimSpace(line)]; ok {
			return action, nil
		}
		fmt.Println("输入无效，请重新输入。")
	}
}

func SelectAsset(assets []github.Asset) (github.Asset, error) {
	fmt.Println("可下载的 mihomo 版本包:")
	for i, asset := range assets {
		fmt.Printf("%3d. %-70s %12d bytes\n", i+1, asset.Name, asset.Size)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("请选择要下载的包编号 [1-%d]: ", len(assets))
		line, err := reader.ReadString('\n')
		if err != nil {
			return github.Asset{}, fmt.Errorf("读取用户输入失败: %w", err)
		}

		index, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || index < 1 || index > len(assets) {
			fmt.Println("输入无效，请重新输入。")
			continue
		}

		return assets[index-1], nil
	}
}

func ConfirmOverwrite(path string) (bool, error) {
	fmt.Printf("本地文件已存在: %s\n", path)
	fmt.Print("是否覆盖重新下载? [y/N]: ")

	return confirmYesNo()
}

func ConfirmOverwriteInstall(path string) (bool, error) {
	fmt.Printf("安装目录中已存在程序文件: %s\n", path)
	fmt.Print("是否覆盖安装? [y/N]: ")

	return confirmYesNo()
}

func confirmYesNo() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("读取用户输入失败: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
