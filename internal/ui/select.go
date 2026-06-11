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
	ActionInstall
)

func SelectAction() (Action, error) {
	fmt.Println("请选择要执行的操作:")
	fmt.Println("  1. 安装/更新 mihomo")
	fmt.Println("  0. 退出")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("请输入操作编号 [0-1]: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return ActionExit, fmt.Errorf("读取用户输入失败: %w", err)
		}

		switch strings.TrimSpace(line) {
		case "1":
			return ActionInstall, nil
		case "0":
			return ActionExit, nil
		default:
			fmt.Println("输入无效，请重新输入。")
		}
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
