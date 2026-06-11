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

const (
	installDir      = "/opt/mihomo"
	installedBinary = "/opt/mihomo/mihomo"
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

func (i *Installer) PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error {
	if fileExists(installedBinary) {
		if !overwrite {
			fmt.Printf("跳过安装，保留现有程序文件: %s\n", installedBinary)
			return nil
		}
	}

	stagingDir := filepath.Join(os.TempDir(), "mihomo-install")
	binaryPath, err := archive.ExtractMihomoBinary(archivePath, assetName, stagingDir)
	if err != nil {
		return fmt.Errorf("解压 mihomo 失败: %w", err)
	}
	fmt.Printf("已解压 mihomo: %s\n", binaryPath)

	commands := [][]string{
		{"mkdir", "-p", installDir},
		{"install", "-m", "0770", binaryPath, installedBinary},
	}

	for _, command := range commands {
		if err := runCommand(ctx, command[0], command[1:]...); err != nil {
			return err
		}
	}

	fmt.Printf("mihomo 程序文件已安装到 %s。\n", installedBinary)
	return nil
}

func (i *Installer) InstallService(ctx context.Context) error {
	if _, err := os.Stat(installedBinary); err != nil {
		return fmt.Errorf("mihomo 程序文件不存在，请先在菜单选择 1 下载并安装程序文件: %w", err)
	}

	stagingDir := filepath.Join(os.TempDir(), "mihomo-install")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	servicePath := filepath.Join(stagingDir, "mihomo.service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		return fmt.Errorf("写入临时 systemd 文件失败: %w", err)
	}

	commands := [][]string{
		{"useradd", "-r", "-M", "-s", "/usr/sbin/nologin", "-U", "mihomo"},
		{"chown", "-R", "mihomo:mihomo", installDir},
		{"install", "-m", "0644", servicePath, "/etc/systemd/system/mihomo.service"},
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "mihomo.service"},
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

	fmt.Println("mihomo systemd 服务安装完成，默认未启动服务。")
	return nil
}

func (i *Installer) Uninstall(ctx context.Context) error {
	runCommandAllowFailure(ctx, "systemctl", "stop", "mihomo.service")
	runCommandAllowFailure(ctx, "systemctl", "disable", "mihomo.service")

	if err := runCommand(ctx, "rm", "-f", "/etc/systemd/system/mihomo.service"); err != nil {
		return err
	}
	if err := runCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return err
	}
	runCommandAllowFailure(ctx, "systemctl", "reset-failed", "mihomo.service")

	if err := runCommand(ctx, "rm", "-rf", installDir); err != nil {
		return err
	}
	if err := runCommand(
		ctx,
		"rm",
		"-rf",
		filepath.Join(os.TempDir(), "mihomo"),
		filepath.Join(os.TempDir(), "mihomo-install"),
	); err != nil {
		return err
	}
	runCommandAllowFailure(ctx, "userdel", "mihomo")

	fmt.Println("mihomo 已卸载并清理完成。")
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

func runCommandAllowFailure(ctx context.Context, name string, args ...string) {
	if err := runCommand(ctx, name, args...); err != nil {
		fmt.Printf("命令失败，继续执行卸载清理: %v\n", err)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
