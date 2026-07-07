//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
Restart=always
RestartSec=5
LimitNPROC=500
LimitNOFILE=51200

[Install]
WantedBy=multi-user.target
`

func (m *Manager) InstallService(ctx context.Context) error {
	if _, err := os.Stat(installedBinary); err != nil {
		return fmt.Errorf("mihomo 程序文件不存在，请先本地安装或下载并安装程序文件: %w", err)
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
		{"install", "-m", "0644", servicePath, serviceFile},
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", serviceName},
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

func (m *Manager) StartService(ctx context.Context) error {
	return m.runServiceControl(ctx, "start", "mihomo 服务已启动。")
}

func (m *Manager) RestartService(ctx context.Context) error {
	return m.runServiceControl(ctx, "restart", "mihomo 服务已重启。")
}

func (m *Manager) runServiceControl(ctx context.Context, systemctlAction string, successMessage string) error {
	if _, err := os.Stat(installedBinary); err != nil {
		return fmt.Errorf("mihomo 程序文件不存在，请先本地安装或下载并安装程序文件: %w", err)
	}
	if _, err := os.Stat(serviceFile); err != nil {
		return fmt.Errorf("mihomo systemd 服务不存在，请先创建用户并安装 mihomo systemd 服务: %w", err)
	}

	commands := [][]string{
		{"chown", "-R", "mihomo:mihomo", installDir},
		{"systemctl", "daemon-reload"},
		{"systemctl", systemctlAction, serviceName},
		{"systemctl", "is-active", serviceName},
	}

	for _, command := range commands {
		if err := runCommand(ctx, command[0], command[1:]...); err != nil {
			return err
		}
	}

	fmt.Println(successMessage)
	return nil
}

func (m *Manager) StopService(ctx context.Context) error {
	if _, err := os.Stat(serviceFile); err != nil {
		return fmt.Errorf("mihomo systemd 服务不存在，请先创建用户并安装 mihomo systemd 服务: %w", err)
	}

	if err := runCommand(ctx, "systemctl", "stop", serviceName); err != nil {
		return err
	}

	fmt.Println("mihomo 服务已停止。")
	return nil
}
