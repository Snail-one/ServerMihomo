package mainmenu

import "snailproxy/internal/ui/menu"

type Action int

const (
	ActionExit Action = iota
	ActionInstall
	ActionManageSubscription
	ActionManageMihomoService
	ActionUninstall
)

func Select() (Action, error) {
	return menu.Select("主菜单:", "[0-4]", []menu.MenuOption[Action]{
		{Number: 1, Label: "安装与更新", Value: ActionInstall},
		{Number: 2, Label: "订阅管理", Value: ActionManageSubscription},
		{Number: 3, Label: "mihomo 服务与代理", Value: ActionManageMihomoService},
		{Number: 4, Label: "卸载", Value: ActionUninstall},
		{Number: 0, Label: "退出", Value: ActionExit},
	})
}
