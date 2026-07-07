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
type SubscriptionAction int
type ServiceAction int

type MenuOption[T any] struct {
	Number int
	Label  string
	Value  T
}

const (
	ActionExit Action = iota
	ActionInstall
	ActionManageSubscription
	ActionManageMihomoService
	ActionUninstall
)

const (
	InstallReturn InstallAction = iota
	InstallLocal
	InstallOnline
	InstallService
)

const (
	SubscriptionReturn SubscriptionAction = iota
	SubscriptionCreate
	SubscriptionUpdate
	SubscriptionModify
	SubscriptionDelete
	SubscriptionApply
)

const (
	ServiceReturn ServiceAction = iota
	ServiceStart
	ServiceRestart
	ServiceStop
	ServiceWriteProxyEnv
	ServiceClearProxyEnv
)

var stdinReader = bufio.NewReader(os.Stdin)

func SelectMainAction() (Action, error) {
	return SelectMenu("主菜单:", "[0-4]", []MenuOption[Action]{
		{Number: 1, Label: "安装与更新", Value: ActionInstall},
		{Number: 2, Label: "订阅管理", Value: ActionManageSubscription},
		{Number: 3, Label: "mihomo 服务与代理", Value: ActionManageMihomoService},
		{Number: 4, Label: "卸载", Value: ActionUninstall},
		{Number: 0, Label: "退出", Value: ActionExit},
	})
}

func SelectInstallAction() (InstallAction, error) {
	return SelectMenu("安装与更新:", "[0-3]", []MenuOption[InstallAction]{
		{Number: 1, Label: "本地安装", Value: InstallLocal},
		{Number: 2, Label: "在线下载并安装 mihomo", Value: InstallOnline},
		{Number: 3, Label: "安装/更新 systemd 服务", Value: InstallService},
		{Number: 0, Label: "返回", Value: InstallReturn},
	})
}

func SelectSubscriptionAction(labels []string) (SubscriptionAction, error) {
	options := []MenuOption[SubscriptionAction]{
		{Number: 1, Label: "新增", Value: SubscriptionCreate},
	}
	promptRange := "[0-1]"
	if len(labels) > 0 {
		options = append(options,
			MenuOption[SubscriptionAction]{Number: 2, Label: "更新已有", Value: SubscriptionUpdate},
			MenuOption[SubscriptionAction]{Number: 3, Label: "修改已有", Value: SubscriptionModify},
			MenuOption[SubscriptionAction]{Number: 4, Label: "删除已有", Value: SubscriptionDelete},
			MenuOption[SubscriptionAction]{Number: 5, Label: "应用订阅", Value: SubscriptionApply},
		)
		promptRange = "[0-5]"
	}
	options = append(options, MenuOption[SubscriptionAction]{Number: 0, Label: "返回", Value: SubscriptionReturn})

	return SelectMenu(subscriptionActionTitle(labels), promptRange, options)
}

func subscriptionActionTitle(labels []string) string {
	if len(labels) == 0 {
		return "订阅管理:\n当前没有订阅。"
	}

	lines := []string{"订阅管理:", "已有订阅:"}
	for _, label := range labels {
		lines = append(lines, "  - "+label)
	}
	return strings.Join(lines, "\n")
}

func SelectServiceAction() (ServiceAction, error) {
	return SelectMenu("mihomo 服务与代理:", "[0-5]", []MenuOption[ServiceAction]{
		{Number: 1, Label: "启动 mihomo 服务", Value: ServiceStart},
		{Number: 2, Label: "重启 mihomo 服务", Value: ServiceRestart},
		{Number: 3, Label: "停止 mihomo 服务", Value: ServiceStop},
		{Number: 4, Label: "写入代理环境变量", Value: ServiceWriteProxyEnv},
		{Number: 5, Label: "清除代理环境变量", Value: ServiceClearProxyEnv},
		{Number: 0, Label: "返回", Value: ServiceReturn},
	})
}

func SelectMenu[T any](title string, promptRange string, options []MenuOption[T]) (T, error) {
	var zero T
	if len(options) == 0 {
		return zero, fmt.Errorf("菜单选项不能为空")
	}

	actions := make(map[int]T, len(options))
	for _, option := range options {
		actions[option.Number] = option.Value
	}

	for {
		if strings.TrimSpace(title) != "" {
			fmt.Println(title)
		}
		for _, option := range options {
			fmt.Printf("  %d. %s\n", option.Number, option.Label)
		}
		fmt.Printf("请输入操作编号 %s: ", promptRange)
		line, err := readLine()
		if err != nil {
			return zero, fmt.Errorf("读取用户输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if value == "" {
			fmt.Println("输入不能为空，请输入菜单编号。")
			fmt.Println()
			continue
		}

		number, err := strconv.Atoi(value)
		if err == nil {
			if action, ok := actions[number]; ok {
				return action, nil
			}
		}
		fmt.Println("输入无效，请重新输入。")
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

func ConfirmOverwriteLocalInstall(targetDir string) (bool, error) {
	fmt.Printf("本地安装目录: %s\n", targetDir)
	return ConfirmNoDefault("遇到同名文件时是否覆盖安装? [y/N]: ")
}

func ConfirmDeleteSubscription(label string) (bool, error) {
	fmt.Printf("将删除订阅: %s\n", label)
	return ConfirmNoDefault("确认删除订阅及本地文件? [y/N]: ")
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

func SelectSubscription(labels []string) (int, error) {
	options := make([]MenuOption[int], 0, len(labels)+1)
	for i, label := range labels {
		options = append(options, MenuOption[int]{
			Number: i + 1,
			Label:  label,
			Value:  i,
		})
	}
	options = append(options, MenuOption[int]{Number: 0, Label: "返回", Value: -1})
	return SelectMenu("可用订阅:", fmt.Sprintf("[0-%d]", len(labels)), options)
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
