//go:build linux

package linux

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"snailproxy/internal/archive"
)

const (
	installDir      = "/opt/mihomo"
	installedBinary = "/opt/mihomo/mihomo"
	serviceFile     = "/etc/systemd/system/mihomo.service"
	serviceName     = "mihomo.service"
)

const (
	defaultMihomoMixedPort = 57913
	defaultNoProxy         = "localhost,127.0.0.1,127.0.0.0/8,::1," +
		"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,169.254.0.0/16,*.local"
	bashrcFileName             = ".bashrc"
	proxyEnvironmentBlockBegin = "# >>> snailproxy proxy environment >>>"
	proxyEnvironmentBlockEnd   = "# <<< snailproxy proxy environment <<<"
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

func (i *Installer) StartService(ctx context.Context) error {
	return i.runServiceControl(ctx, "start", "mihomo 服务已启动。")
}

func (i *Installer) RestartService(ctx context.Context) error {
	return i.runServiceControl(ctx, "restart", "mihomo 服务已重启。")
}

func (i *Installer) runServiceControl(ctx context.Context, systemctlAction string, successMessage string) error {
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

func (i *Installer) StopService(ctx context.Context) error {
	if _, err := os.Stat(serviceFile); err != nil {
		return fmt.Errorf("mihomo systemd 服务不存在，请先创建用户并安装 mihomo systemd 服务: %w", err)
	}

	if err := runCommand(ctx, "systemctl", "stop", serviceName); err != nil {
		return err
	}

	fmt.Println("mihomo 服务已停止。")
	return nil
}

func (i *Installer) WriteProxyEnvironment(ctx context.Context) error {
	settings := detectMihomoProxySettings()
	target, err := currentProxyEnvironmentTarget()
	if err != nil {
		return err
	}
	block := proxyEnvironmentBashrcBlock(settings)
	if err := writeProxyEnvironmentToBashrc(target, block); err != nil {
		return err
	}

	fmt.Printf("代理环境变量已写入 .bashrc 底部: %s\n", target.Path)
	fmt.Println("写入内容:")
	fmt.Print(block)
	fmt.Printf("重新登录，或执行 source %s 后生效。\n", target.Path)
	return nil
}

func (i *Installer) ClearProxyEnvironment(ctx context.Context) error {
	target, err := currentProxyEnvironmentTarget()
	if err != nil {
		return err
	}

	removed, err := removeProxyEnvironmentFromBashrc(target)
	if err != nil {
		return err
	}
	if !removed {
		fmt.Printf("未在 %s 中找到 snailproxy 代理环境变量配置，无需清除。\n", target.Path)
		return nil
	}

	fmt.Printf("已从 .bashrc 清除代理环境变量配置: %s\n", target.Path)
	fmt.Println("已打开的终端如果加载过代理变量，需要重新登录或手动 unset 后才会清除。")
	return nil
}

func (i *Installer) Uninstall(ctx context.Context) error {
	runCommandAllowFailure(ctx, "systemctl", "stop", serviceName)
	runCommandAllowFailure(ctx, "systemctl", "disable", serviceName)

	if err := runCommand(ctx, "rm", "-f", serviceFile); err != nil {
		return err
	}
	if err := runCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return err
	}
	runCommandAllowFailure(ctx, "systemctl", "reset-failed", serviceName)

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

type proxyEnvironmentSettings struct {
	HTTPProxy string
	AllProxy  string
	NoProxy   string
}

type proxyEnvironmentTarget struct {
	Path string
	UID  int
	GID  int
}

type systemUser struct {
	HomeDir string
	UID     string
	GID     string
}

func writeProxyEnvironmentToBashrc(target proxyEnvironmentTarget, block string) error {
	data, err := os.ReadFile(target.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取 .bashrc 失败: %w", err)
	}

	updated := appendProxyEnvironmentBlockToBashrc(string(data), block)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("写入 .bashrc 失败: %w", err)
	}
	return chownProxyEnvironmentTarget(target)
}

func removeProxyEnvironmentFromBashrc(target proxyEnvironmentTarget) (bool, error) {
	data, err := os.ReadFile(target.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("读取 .bashrc 失败: %w", err)
	}

	updated, removed := removeProxyEnvironmentBlockFromBashrc(string(data))
	if !removed {
		return false, nil
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("写入 .bashrc 失败: %w", err)
	}
	if err := chownProxyEnvironmentTarget(target); err != nil {
		return false, err
	}
	return true, nil
}

func appendProxyEnvironmentBlockToBashrc(content string, block string) string {
	content, _ = removeProxyEnvironmentBlockFromBashrc(content)
	content = strings.TrimRight(content, "\r\n")
	if content == "" {
		return block
	}
	return content + "\n\n" + block
}

func removeProxyEnvironmentBlockFromBashrc(content string) (string, bool) {
	removed := false
	for {
		start := strings.Index(content, proxyEnvironmentBlockBegin)
		if start < 0 {
			return content, removed
		}
		removed = true

		endOffset := strings.Index(content[start:], proxyEnvironmentBlockEnd)
		if endOffset < 0 {
			return strings.TrimRight(content[:start], "\r\n"), removed
		}

		end := start + endOffset + len(proxyEnvironmentBlockEnd)
		for end < len(content) && (content[end] == '\r' || content[end] == '\n') {
			end++
		}

		before := strings.TrimRight(content[:start], "\r\n")
		after := strings.TrimLeft(content[end:], "\r\n")
		if before != "" && after != "" {
			content = before + "\n\n" + after
		} else {
			content = before + after
		}
	}
}

func proxyEnvironmentBashrcBlock(settings proxyEnvironmentSettings) string {
	return proxyEnvironmentBlockBegin + "\n" +
		strings.TrimRight(proxyEnvironmentContent(settings), "\r\n") + "\n" +
		proxyEnvironmentBlockEnd + "\n"
}

func currentProxyEnvironmentTarget() (proxyEnvironmentTarget, error) {
	return proxyEnvironmentTargetFromEnvironment(os.Getenv, lookupSystemUser, currentSystemUser)
}

func proxyEnvironmentTargetFromEnvironment(getenv func(string) string, lookup func(string) (systemUser, error), current func() (systemUser, error)) (proxyEnvironmentTarget, error) {
	username := strings.TrimSpace(getenv("SUDO_USER"))
	if username != "" && username != "root" {
		account, err := lookup(username)
		if err != nil {
			return proxyEnvironmentTarget{}, fmt.Errorf("查找 sudo 用户 %s 失败: %w", username, err)
		}
		return proxyEnvironmentTargetFromSystemUser(username, account)
	}

	account, err := current()
	if err != nil {
		return proxyEnvironmentTarget{}, fmt.Errorf("查找当前运行用户失败: %w", err)
	}
	return proxyEnvironmentTargetFromSystemUser("当前运行用户", account)
}

func proxyEnvironmentTargetFromSystemUser(label string, account systemUser) (proxyEnvironmentTarget, error) {
	if strings.TrimSpace(account.HomeDir) == "" {
		return proxyEnvironmentTarget{}, fmt.Errorf("%s 缺少 home 目录", label)
	}
	uid, err := strconv.Atoi(account.UID)
	if err != nil {
		return proxyEnvironmentTarget{}, fmt.Errorf("解析 %s uid 失败: %w", label, err)
	}
	gid, err := strconv.Atoi(account.GID)
	if err != nil {
		return proxyEnvironmentTarget{}, fmt.Errorf("解析 %s gid 失败: %w", label, err)
	}
	return proxyEnvironmentTarget{
		Path: filepath.Join(account.HomeDir, bashrcFileName),
		UID:  uid,
		GID:  gid,
	}, nil
}

func chownProxyEnvironmentTarget(target proxyEnvironmentTarget) error {
	if err := os.Chown(target.Path, target.UID, target.GID); err != nil {
		return fmt.Errorf("设置 .bashrc 文件归属失败: %w", err)
	}
	return nil
}

func lookupSystemUser(username string) (systemUser, error) {
	account, err := user.Lookup(username)
	if err != nil {
		return systemUser{}, err
	}
	return systemUser{
		HomeDir: account.HomeDir,
		UID:     account.Uid,
		GID:     account.Gid,
	}, nil
}

func currentSystemUser() (systemUser, error) {
	account, err := user.Current()
	if err != nil {
		return systemUser{}, err
	}
	return systemUser{
		HomeDir: account.HomeDir,
		UID:     account.Uid,
		GID:     account.Gid,
	}, nil
}

func detectMihomoProxySettings() proxyEnvironmentSettings {
	configPaths := []string{
		filepath.Join(installDir, "config.yaml"),
		filepath.Join(installDir, "templates", "mihomo.yaml"),
	}

	for _, configPath := range configPaths {
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}
		if settings, ok := proxyEnvironmentSettingsFromConfig(data); ok {
			return settings
		}
	}

	return proxyEnvironmentSettingsFromMixedPort(defaultMihomoMixedPort)
}

func proxyEnvironmentSettingsFromConfig(data []byte) (proxyEnvironmentSettings, bool) {
	if mixedPort, ok := parseTopLevelInt(data, "mixed-port"); ok {
		return proxyEnvironmentSettingsFromMixedPort(mixedPort), true
	}

	httpPort, hasHTTPPort := parseTopLevelInt(data, "port")
	socksPort, hasSocksPort := parseTopLevelInt(data, "socks-port")
	switch {
	case hasHTTPPort && hasSocksPort:
		return proxyEnvironmentSettings{
			HTTPProxy: httpProxyURL(httpPort),
			AllProxy:  socksProxyURL(socksPort),
			NoProxy:   defaultNoProxy,
		}, true
	case hasHTTPPort:
		httpProxy := httpProxyURL(httpPort)
		return proxyEnvironmentSettings{
			HTTPProxy: httpProxy,
			AllProxy:  httpProxy,
			NoProxy:   defaultNoProxy,
		}, true
	case hasSocksPort:
		socksProxy := socksProxyURL(socksPort)
		return proxyEnvironmentSettings{
			HTTPProxy: socksProxy,
			AllProxy:  socksProxy,
			NoProxy:   defaultNoProxy,
		}, true
	default:
		return proxyEnvironmentSettings{}, false
	}
}

func proxyEnvironmentSettingsFromMixedPort(port int) proxyEnvironmentSettings {
	return proxyEnvironmentSettings{
		HTTPProxy: httpProxyURL(port),
		AllProxy:  socksProxyURL(port),
		NoProxy:   defaultNoProxy,
	}
}

func httpProxyURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func socksProxyURL(port int) string {
	return fmt.Sprintf("socks5h://127.0.0.1:%d", port)
}

func parseTopLevelInt(data []byte, key string) (int, bool) {
	prefix := key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value = strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
		value = strings.Trim(value, `"'`)
		port, err := strconv.Atoi(value)
		if err == nil && port > 0 && port <= 65535 {
			return port, true
		}
	}
	return 0, false
}

func proxyEnvironmentContent(settings proxyEnvironmentSettings) string {
	return fmt.Sprintf(`# Managed by snailproxy. Use snailproxy to clear this file.
export http_proxy=%q
export HTTP_PROXY=%q
export https_proxy=%q
export HTTPS_PROXY=%q
export ftp_proxy=%q
export FTP_PROXY=%q
export all_proxy=%q
export ALL_PROXY=%q
export no_proxy=%q
export NO_PROXY=%q
`,
		settings.HTTPProxy,
		settings.HTTPProxy,
		settings.HTTPProxy,
		settings.HTTPProxy,
		settings.HTTPProxy,
		settings.HTTPProxy,
		settings.AllProxy,
		settings.AllProxy,
		settings.NoProxy,
		settings.NoProxy,
	)
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
