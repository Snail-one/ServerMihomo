//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultMihomoMixedPort = 57913
	defaultNoProxy         = "localhost,127.0.0.1,127.0.0.0/8,::1," +
		"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,169.254.0.0/16,*.local"
	bashrcFileName             = ".bashrc"
	proxyEnvironmentBlockBegin = "# >>> snailproxy proxy environment >>>"
	proxyEnvironmentBlockEnd   = "# <<< snailproxy proxy environment <<<"
)

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

func (m *Manager) WriteProxyEnvironment(ctx context.Context) error {
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

func (m *Manager) ClearProxyEnvironment(ctx context.Context) error {
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
	data, err := os.ReadFile(filepath.Join(installDir, "config.yaml"))
	if err == nil {
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
