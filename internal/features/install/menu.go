package install

import (
	"fmt"
	"strconv"
	"strings"

	"snailproxy/internal/infra/github"
	"snailproxy/internal/terminal"
)

type Action int

const (
	ActionReturn Action = iota
	ActionLocal
	ActionOnline
	ActionService
)

func Select() (Action, error) {
	return terminal.Select("安装与更新:", "[0-3]", []terminal.MenuOption[Action]{
		{Number: 1, Label: "本地安装", Value: ActionLocal},
		{Number: 2, Label: "在线下载并安装 mihomo", Value: ActionOnline},
		{Number: 3, Label: "安装/更新 systemd 服务", Value: ActionService},
		{Number: 0, Label: "返回", Value: ActionReturn},
	})
}

func SelectAsset(assets []github.Asset) (github.Asset, error) {
	fmt.Println("可下载的 mihomo 版本包:")
	for i, asset := range assets {
		fmt.Printf("%3d. %-70s %12d bytes\n", i+1, asset.Name, asset.Size)
	}

	for {
		fmt.Printf("请选择要下载的包编号 [1-%d]: ", len(assets))
		line, err := terminal.ReadLine()
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
	return terminal.ConfirmNoDefault("是否覆盖重新下载? [y/N]: ")
}

func ConfirmOverwriteInstall(path string) (bool, error) {
	fmt.Printf("安装目录中已存在程序文件: %s\n", path)
	return terminal.ConfirmNoDefault("是否覆盖安装? [y/N]: ")
}

func ConfirmOverwriteLocalInstall(targetDir string) (bool, error) {
	fmt.Printf("本地安装目录: %s\n", targetDir)
	return terminal.ConfirmNoDefault("遇到同名文件时是否覆盖安装? [y/N]: ")
}
