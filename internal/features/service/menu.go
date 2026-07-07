package service

import "snailproxy/internal/terminal"

type Action int

const (
	ActionReturn Action = iota
	ActionStart
	ActionRestart
	ActionStop
	ActionWriteProxyEnv
	ActionClearProxyEnv
)

func Select() (Action, error) {
	return terminal.Select("mihomo 服务与代理:", "[0-5]", []terminal.MenuOption[Action]{
		{Number: 1, Label: "启动 mihomo 服务", Value: ActionStart},
		{Number: 2, Label: "重启 mihomo 服务", Value: ActionRestart},
		{Number: 3, Label: "停止 mihomo 服务", Value: ActionStop},
		{Number: 4, Label: "写入代理环境变量", Value: ActionWriteProxyEnv},
		{Number: 5, Label: "清除代理环境变量", Value: ActionClearProxyEnv},
		{Number: 0, Label: "返回", Value: ActionReturn},
	})
}
