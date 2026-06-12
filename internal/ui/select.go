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
type SubscriptionDownloadAction int

const (
	ActionExit Action = iota
	ActionDownload
	ActionInstallService
	ActionUninstall
	ActionDownloadSubscription
	ActionSelectSubscription
	ActionReleaseResources
)

const (
	SubscriptionDownloadReturn SubscriptionDownloadAction = iota
	SubscriptionDownloadUpdate
	SubscriptionDownloadCreate
	SubscriptionDownloadDelete
)

var stdinReader = bufio.NewReader(os.Stdin)

func SelectLinuxAction() (Action, error) {
	fmt.Println("Linux 操作菜单:")
	fmt.Println("  1. 下载并安装 mihomo 程序文件")
	fmt.Println("  2. 创建用户并安装 mihomo systemd 服务")
	fmt.Println("  3. 下载/更新/删除 Clash 订阅")
	fmt.Println("  4. 选择订阅并生成 mihomo 配置")
	fmt.Println("  5. 释放本地资源包")
	fmt.Println("  6. 卸载并清理 mihomo")
	fmt.Println("  0. 退出")

	return selectAction("[0-6]", map[string]Action{
		"1": ActionDownload,
		"2": ActionInstallService,
		"3": ActionDownloadSubscription,
		"4": ActionSelectSubscription,
		"5": ActionReleaseResources,
		"6": ActionUninstall,
		"0": ActionExit,
	})
}

func SelectWindowsAction() (Action, error) {
	fmt.Println("Windows 操作菜单:")
	fmt.Println("  1. 下载 mihomo")
	fmt.Println("  2. 下载/更新/删除 Clash 订阅")
	fmt.Println("  3. 选择订阅并生成 mihomo 配置")
	fmt.Println("  4. 释放本地资源包")
	fmt.Println("  0. 退出")

	return selectAction("[0-4]", map[string]Action{
		"1": ActionDownload,
		"2": ActionDownloadSubscription,
		"3": ActionSelectSubscription,
		"4": ActionReleaseResources,
		"0": ActionExit,
	})
}

func selectAction(promptRange string, actions map[string]Action) (Action, error) {
	for {
		fmt.Printf("请输入操作编号 %s: ", promptRange)
		line, err := readLine()
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

func ConfirmOverwriteResourcePackage(targetDir string) (bool, error) {
	fmt.Printf("本地资源包将释放到: %s\n", targetDir)
	return ConfirmNoDefault("遇到同名文件时是否覆盖? [y/N]: ")
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
	fmt.Println("订阅下载/更新/删除:")
	for i, label := range labels {
		fmt.Printf("  %d. 更新 %s\n", i+1, label)
	}
	newOption := len(labels) + 1
	fmt.Printf("  %d. 手动下载新的订阅\n", newOption)
	deleteOption := 0
	maxOption := newOption
	if len(labels) > 0 {
		deleteOption = len(labels) + 2
		maxOption = deleteOption
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
	for {
		fmt.Print("请输入 Clash 订阅链接: ")
		line, err := readLine()
		if err != nil {
			return "", fmt.Errorf("读取用户输入失败: %w", err)
		}
		value := strings.TrimSpace(line)
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
