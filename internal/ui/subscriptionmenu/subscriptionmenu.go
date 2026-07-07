package subscriptionmenu

import (
	"fmt"
	"strings"

	"snailproxy/internal/ui/menu"
)

type Action int

const (
	ActionReturn Action = iota
	ActionCreate
	ActionUpdate
	ActionModify
	ActionDelete
	ActionApply
)

func Select(labels []string) (Action, error) {
	options := []menu.MenuOption[Action]{
		{Number: 1, Label: "新增", Value: ActionCreate},
	}
	promptRange := "[0-1]"
	if len(labels) > 0 {
		options = append(options,
			menu.MenuOption[Action]{Number: 2, Label: "更新已有", Value: ActionUpdate},
			menu.MenuOption[Action]{Number: 3, Label: "修改已有", Value: ActionModify},
			menu.MenuOption[Action]{Number: 4, Label: "删除已有", Value: ActionDelete},
			menu.MenuOption[Action]{Number: 5, Label: "应用订阅", Value: ActionApply},
		)
		promptRange = "[0-5]"
	}
	options = append(options, menu.MenuOption[Action]{Number: 0, Label: "返回", Value: ActionReturn})

	return menu.Select(subscriptionActionTitle(labels), promptRange, options)
}

func SelectSubscription(labels []string) (int, error) {
	options := make([]menu.MenuOption[int], 0, len(labels)+1)
	for i, label := range labels {
		options = append(options, menu.MenuOption[int]{
			Number: i + 1,
			Label:  label,
			Value:  i,
		})
	}
	options = append(options, menu.MenuOption[int]{Number: 0, Label: "返回", Value: -1})
	return menu.Select("可用订阅:", fmt.Sprintf("[0-%d]", len(labels)), options)
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
		line, err := menu.ReadLine()
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

		line, err := menu.ReadLine()
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

func ConfirmDeleteSubscription(label string) (bool, error) {
	fmt.Printf("将删除订阅: %s\n", label)
	return menu.ConfirmNoDefault("确认删除订阅及本地文件? [y/N]: ")
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
