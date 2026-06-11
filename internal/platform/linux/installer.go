//go:build linux

package linux

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"snailproxy/internal/archive"
)

const serviceContent = `[Unit]
Description=Mihomo Proxy Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mihomo
WorkingDirectory=/opt/mihomo
ExecStart=/opt/mihomo/mihomo -d /opt/mihomo
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
LimitNPROC=500
LimitNOFILE=51200

[Install]
WantedBy=multi-user.target
`

type Installer struct{}

func (i *Installer) Install(ctx context.Context, archivePath string, assetName string) error {
	stagingDir := filepath.Join(os.TempDir(), "snailproxy-mihomo-install")
	binaryPath, err := archive.ExtractMihomoBinary(archivePath, assetName, stagingDir)
	if err != nil {
		return fmt.Errorf("解压 mihomo 失败: %w", err)
	}
	fmt.Printf("已解压 mihomo: %s\n", binaryPath)

	servicePath := filepath.Join(stagingDir, "mihomo.service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		return fmt.Errorf("写入临时 systemd 文件失败: %w", err)
	}

	commands := [][]string{
		{"useradd", "-r", "-M", "-s", "/usr/sbin/nologin", "-U", "mihomo"},
		{"mkdir", "-p", "/opt/mihomo"},
		{"install", "-m", "0755", binaryPath, "/opt/mihomo/mihomo"},
		{"chown", "-R", "mihomo:mihomo", "/opt/mihomo"},
		{"install", "-m", "0644", servicePath, "/etc/systemd/system/mihomo.service"},
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "mihomo.service"},
		{"systemctl", "start", "mihomo.service"},
	}

	for _, command := range commands {
		if err := runCommand(ctx, command[0], command[1:]...); err != nil {
			if command[0] == "useradd" {
				fmt.Println("创建 mihomo 用户失败，可能用户已存在；继续检测用户。")
				if checkErr := runCommand(ctx, "id", "mihomo"); checkErr == nil {
					continue
				}
			}
			return err
		}
	}

	fmt.Println("mihomo Linux systemd 服务安装完成。")
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	command := append([]string{name}, args...)

	fmt.Printf("\n执行命令: %s\n", joinCommand(command))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = os.Stdin

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	fmt.Print("命令输出:\n")
	if output.Len() == 0 {
		fmt.Println("(无输出)")
	} else {
		fmt.Print(output.String())
	}

	if err != nil {
		return fmt.Errorf("命令执行失败 %q: %w", joinCommand(command), err)
	}
	return nil
}

func joinCommand(parts []string) string {
	var buf bytes.Buffer
	for i, part := range parts {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(part)
	}
	return buf.String()
}
