package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	subscriptionMetadataFile = "subscriptions.json"
	finalConfigFile          = "config.yaml"
)

type Store struct {
	BaseDir string
}

func NewStore() Store {
	return Store{BaseDir: DefaultBaseDir()}
}

func DefaultBaseDir() string {
	return "/opt/mihomo"
}

func (s Store) ProfilesDir() string {
	return filepath.Join(s.BaseDir, "profiles")
}

func (s Store) MetadataPath() string {
	return filepath.Join(s.ProfilesDir(), subscriptionMetadataFile)
}

func (s Store) FinalConfigPath() string {
	return filepath.Join(s.BaseDir, finalConfigFile)
}

func (s Store) ProfilePath(file string) string {
	return filepath.Join(s.ProfilesDir(), filepath.Base(file))
}

func (s Store) EnsureDirs() error {
	for _, dir := range []string{s.BaseDir, s.ProfilesDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}
	return nil
}

func (s Store) CopySubscriptionToFinalConfig(subscription Subscription) (string, error) {
	subscriptionFile := filepath.Base(subscription.File)
	if strings.TrimSpace(subscriptionFile) == "" || subscriptionFile == "." {
		return "", fmt.Errorf("订阅 %s 缺少实际文件名", subscription.Name)
	}

	subscriptionPath := s.ProfilePath(subscriptionFile)
	data, err := os.ReadFile(subscriptionPath)
	if err != nil {
		return "", fmt.Errorf("读取订阅文件 %s 失败: %w", subscriptionPath, err)
	}
	if err := ValidateSubscriptionData(data); err != nil {
		return "", err
	}

	if err := os.MkdirAll(s.BaseDir, 0o755); err != nil {
		return "", fmt.Errorf("创建 mihomo 目录失败: %w", err)
	}
	if err := os.WriteFile(s.FinalConfigPath(), data, 0o644); err != nil {
		return "", fmt.Errorf("写入最终配置失败: %w", err)
	}
	return s.FinalConfigPath(), nil
}
