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
type InstallAction int
type SubscriptionDownloadAction int
type MihomoServiceAction int

const (
	ActionExit Action = iota
	ActionInstall
	ActionDownload
	ActionInstallService
	ActionUninstall
	ActionDownloadSubscription
	ActionApplySubscription
	ActionLocalInstall
	ActionVerifyLocalMihomo
	ActionManageMihomoService
)

const (
	InstallReturn InstallAction = iota
	InstallLocal
	InstallOnline
	InstallService
)

const (
	SubscriptionDownloadReturn SubscriptionDownloadAction = iota
	SubscriptionDownloadUpdate
	SubscriptionDownloadCreate
	SubscriptionDownloadModify
	SubscriptionDownloadDelete
)

const (
	MihomoServiceReturn MihomoServiceAction = iota
	MihomoServiceStart
	MihomoServiceRestart
	MihomoServiceStop
	MihomoServiceWriteProxyEnv
	MihomoServiceClearProxyEnv
)

var stdinReader = bufio.NewReader(os.Stdin)

func SelectLinuxAction() (Action, error) {
	printMenu := func() {
		fmt.Println("Linux 操作菜单:")
		fmt.Println("  1. 安装")
		fmt.Println("  2. Clash 订阅管理")
		fmt.Println("  3. 应用订阅")
		fmt.Println("  4. mihomo 服务管理")
		fmt.Println("  5. 验证本地文件")
		fmt.Println("  6. 卸载")
		fmt.Println("  0. 退出")
	}

	return selectAction("[0-6]", printMenu, map[string]Action{
		"1": ActionInstall,
		"2": ActionDownloadSubscription,
		"3": ActionApplySubscription,
		"4": ActionManageMihomoService,
		"5": ActionVerifyLocalMihomo,
		"6": ActionUninstall,
		"0": ActionExit,
	})
}

func SelectInstallAction() (InstallAction, error) {
	actions := map[string]InstallAction{
		"1": InstallLocal,
		"2": InstallOnline,
		"3": InstallService,
		"0": InstallReturn,
	}

	for {
		fmt.Println("安装:")
		fmt.Println("  1. 本地安装")
		fmt.Println("  2. 在线安装")
		fmt.Println("  3. 创建用户并安装 mihomo systemd 服务")
		fmt.Println("  0. 返回主菜单")
		fmt.Print("请输入操作编号 [0-3]: ")
		line, err := readLine()
		if err != nil {
			return InstallReturn, fmt.Errorf("读取用户输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if action, ok := actions[value]; ok {
			return action, nil
		}
		if value == "" {
			fmt.Println("输入不能为空，请输入菜单编号。")
		} else {
			fmt.Println("输入无效，请重新输入。")
		}
		fmt.Println()
	}
}

func SelectMihomoServiceAction() (MihomoServiceAction, error) {
	actions := map[string]MihomoServiceAction{
		"1": MihomoServiceStart,
		"2": MihomoServiceRestart,
		"3": MihomoServiceStop,
		"4": MihomoServiceWriteProxyEnv,
		"5": MihomoServiceClearProxyEnv,
		"0": MihomoServiceReturn,
	}

	for {
		fmt.Println("mihomo 服务和代理管理:")
		fmt.Println("  1. 启动 mihomo 服务")
		fmt.Println("  2. 重启 mihomo 服务")
		fmt.Println("  3. 停止 mihomo 服务")
		fmt.Println("  4. 写入代理环境变量")
		fmt.Println("  5. 清除代理环境变量")
		fmt.Println("  0. 返回主菜单")
		fmt.Print("请输入操作编号 [0-5]: ")
		line, err := readLine()
		if err != nil {
			return MihomoServiceReturn, fmt.Errorf("读取用户输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if action, ok := actions[value]; ok {
			return action, nil
		}
		if value == "" {
			fmt.Println("输入不能为空，请输入菜单编号。")
		} else {
			fmt.Println("输入无效，请重新输入。")
		}
		fmt.Println()
	}
}

func selectAction(promptRange string, printMenu func(), actions map[string]Action) (Action, error) {
	for {
		printMenu()
		fmt.Printf("请输入操作编号 %s: ", promptRange)
		line, err := readLine()
		if err != nil {
			return ActionExit, fmt.Errorf("读取用户输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if action, ok := actions[value]; ok {
			return action, nil
		}
		if value == "" {
			fmt.Println("输入不能为空，请输入菜单编号。")
		} else {
			fmt.Println("输入无效，请重新输入。")
		}
		fmt.Println()
	}
}

func SelectAsset(assets []github.Asset) (github.Asset, error) {
	fmt.Println("可下载的 mihomo 版本包:")
	for i, asset := range assets {
		fmt.Printf("%3d. %-70s %12d bytes\n", i+1, asset.Name, asset.Size)
	}

	for {
		fmt.Printf("请选择要下载的包编号 [1-%d]: ", len(assets))
		line, err := readLine()
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
	return ConfirmNoDefault("是否覆盖重新下载? [y/N]: ")
}

func ConfirmOverwriteInstall(path string) (bool, error) {
	fmt.Printf("安装目录中已存在程序文件: %s\n", path)
	return ConfirmNoDefault("是否覆盖安装? [y/N]: ")
}

func ConfirmOverwriteDefaultConfig(path string) (bool, error) {
	fmt.Printf("默认配置已存在: %s\n", path)
	return ConfirmNoDefault("是否覆盖默认配置? [y/N]: ")
}

func ConfirmOverwriteLocalInstall(targetDir string) (bool, error) {
	fmt.Printf("本地安装目录: %s\n", targetDir)
	return ConfirmNoDefault("遇到同名文件时是否覆盖安装? [y/N]: ")
}

func ConfirmDeleteSubscription(label string) (bool, error) {
	fmt.Printf("将删除订阅: %s\n", label)
	return ConfirmNoDefault("确认删除订阅及本地文件? [y/N]: ")
}

func ConfirmYesDefault(prompt string) (bool, error) {
	fmt.Print(prompt)
	line, err := readLine()
	if err != nil {
		return false, fmt.Errorf("读取用户输入失败: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "" {
		return true, nil
	}
	return answer == "y" || answer == "yes", nil
}

func ConfirmNoDefault(prompt string) (bool, error) {
	fmt.Print(prompt)
	line, err := readLine()
	if err != nil {
		return false, fmt.Errorf("读取用户输入失败: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func SelectSubscriptionDownloadTarget(labels []string) (int, SubscriptionDownloadAction, error) {
	fmt.Println("订阅下载/更新/修改/删除:")
	for i, label := range labels {
		fmt.Printf("  %d. 更新 %s\n", i+1, label)
	}
	newOption := len(labels) + 1
	fmt.Printf("  %d. 手动下载新的订阅\n", newOption)
	modifyOption := 0
	deleteOption := 0
	maxOption := newOption
	if len(labels) > 0 {
		modifyOption = len(labels) + 2
		deleteOption = len(labels) + 3
		maxOption = deleteOption
		fmt.Printf("  %d. 修改已有订阅\n", modifyOption)
		fmt.Printf("  %d. 删除已有订阅\n", deleteOption)
	}
	fmt.Println("  0. 返回")

	for {
		fmt.Printf("请选择订阅操作 [0-%d]: ", maxOption)
		line, err := readLine()
		if err != nil {
			return -1, SubscriptionDownloadReturn, fmt.Errorf("读取用户输入失败: %w", err)
		}

		index, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || index < 0 || index > maxOption {
			fmt.Println("输入无效，请重新输入。")
			continue
		}
		if index == 0 {
			return -1, SubscriptionDownloadReturn, nil
		}
		if index == newOption {
			return -1, SubscriptionDownloadCreate, nil
		}
		if modifyOption > 0 && index == modifyOption {
			return -1, SubscriptionDownloadModify, nil
		}
		if deleteOption > 0 && index == deleteOption {
			return -1, SubscriptionDownloadDelete, nil
		}
		return index - 1, SubscriptionDownloadUpdate, nil
	}
}

func SelectSubscription(labels []string) (int, error) {
	fmt.Println("可用订阅:")
	for i, label := range labels {
		fmt.Printf("  %d. %s\n", i+1, label)
	}
	fmt.Println("  0. 返回")

	for {
		fmt.Printf("请选择订阅 [0-%d]: ", len(labels))
		line, err := readLine()
		if err != nil {
			return -1, fmt.Errorf("读取用户输入失败: %w", err)
		}

		index, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || index < 0 || index > len(labels) {
			fmt.Println("输入无效，请重新输入。")
			continue
		}
		if index == 0 {
			return -1, nil
		}
		return index - 1, nil
	}
}

func PromptSubscriptionURL() (string, error) {
	return PromptSubscriptionURLDefault("")
}

func PromptSubscriptionURLDefault(defaultURL string) (string, error) {
	for {
		if strings.TrimSpace(defaultURL) == "" {
			fmt.Print("请输入 Clash 订阅链接: ")
		} else {
			fmt.Printf("请输入 Clash 订阅链接 [%s]: ", defaultURL)
		}
		line, err := readLine()
		if err != nil {
			return "", fmt.Errorf("读取用户输入失败: %w", err)
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = strings.TrimSpace(defaultURL)
		}
		if value == "" {
			fmt.Println("订阅链接不能为空。")
			continue
		}
		return value, nil
	}
}

func PromptSubscriptionName(defaultName string) (string, error) {
	for {
		if strings.TrimSpace(defaultName) == "" {
			fmt.Print("请输入保存的订阅名称: ")
		} else {
			fmt.Printf("请输入保存的订阅名称 [%s]: ", defaultName)
		}

		line, err := readLine()
		if err != nil {
			return "", fmt.Errorf("读取用户输入失败: %w", err)
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = strings.TrimSpace(defaultName)
		}
		if value == "" {
			fmt.Println("订阅名称不能为空。")
			continue
		}
		return value, nil
	}
}

func readLine() (string, error) {
	return stdinReader.ReadString('\n')
}
