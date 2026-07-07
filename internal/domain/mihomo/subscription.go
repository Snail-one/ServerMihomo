package mihomo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"snailproxy/internal/infra/progress"
)

type Subscription struct {
	Name string `json:"name"`
	File string `json:"file"`
	URL  string `json:"url"`
}

func (s Store) LoadSubscriptions() ([]Subscription, error) {
	data, err := os.ReadFile(s.MetadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取订阅信息失败: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}

	var subscriptions []Subscription
	if err := json.Unmarshal(data, &subscriptions); err != nil {
		return nil, fmt.Errorf("解析订阅信息失败: %w", err)
	}
	for i := range subscriptions {
		subscriptions[i].File = filepath.Base(subscriptions[i].File)
	}
	return subscriptions, nil
}

func (s Store) SaveSubscriptions(subscriptions []Subscription) error {
	if err := os.MkdirAll(s.ProfilesDir(), 0o755); err != nil {
		return fmt.Errorf("创建 profiles 目录失败: %w", err)
	}

	normalized := make([]Subscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		subscription.Name = strings.TrimSpace(subscription.Name)
		subscription.File = filepath.Base(strings.TrimSpace(subscription.File))
		subscription.URL = strings.TrimSpace(subscription.URL)
		if subscription.Name == "" || subscription.File == "" || subscription.URL == "" {
			continue
		}
		normalized = append(normalized, subscription)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("编码订阅信息失败: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.MetadataPath(), data, 0o644); err != nil {
		return fmt.Errorf("保存订阅信息失败: %w", err)
	}
	return nil
}

func (s Store) NewSubscriptionFileName() (string, error) {
	if err := os.MkdirAll(s.ProfilesDir(), 0o755); err != nil {
		return "", fmt.Errorf("创建 profiles 目录失败: %w", err)
	}

	for i := 0; i < 20; i++ {
		token, err := randomHex(12)
		if err != nil {
			return "", err
		}
		file := "subscription-" + token + ".yaml"
		if _, err := os.Stat(s.ProfilePath(file)); os.IsNotExist(err) {
			return file, nil
		}
	}
	return "", fmt.Errorf("生成唯一订阅文件名失败")
}

func DownloadSubscription(ctx context.Context, rawURL string, targetPath string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("订阅链接无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("订阅链接只支持 http/https: %s", parsedURL.Scheme)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建订阅目录失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "snailproxy-subscription")

	directTransport := http.DefaultTransport.(*http.Transport).Clone()
	directTransport.Proxy = nil
	directClient := &http.Client{
		Timeout:   90 * time.Second,
		Transport: directTransport,
	}

	fmt.Println("当前订阅下载方式: 直连")
	err = downloadSubscriptionWithClient(directClient, req, targetPath)
	if err != nil {
		var validationErr subscriptionValidationError
		if errors.As(err, &validationErr) {
			return err
		}

		proxyURL, proxyErr := http.ProxyFromEnvironment(req)
		if proxyErr != nil {
			return fmt.Errorf("直连下载订阅失败: %w；代理配置无效: %v", err, proxyErr)
		}
		if proxyURL == nil {
			return fmt.Errorf("直连下载订阅失败，且未配置代理: %w", err)
		}

		fmt.Printf("直连下载订阅失败，切换为代理 %s: %v\n", proxyURL.String(), err)
		fmt.Printf("当前订阅下载方式: 使用代理 %s\n", proxyURL.String())
		proxyClient := &http.Client{Timeout: 90 * time.Second}
		if proxyErr := downloadSubscriptionWithClient(proxyClient, req, targetPath); proxyErr != nil {
			return fmt.Errorf("代理下载订阅失败: %w", proxyErr)
		}
		return nil
	}
	return nil
}

func downloadSubscriptionWithClient(client *http.Client, req *http.Request, targetPath string) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载订阅失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载订阅返回状态码: %s", resp.Status)
	}

	downloadPath := targetPath + ".download"
	out, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建订阅下载文件失败: %w", err)
	}
	progressWriter := progress.NewWriter(os.Stdout, "订阅下载进度", resp.ContentLength)
	written, copyErr := io.Copy(io.MultiWriter(out, progressWriter), resp.Body)
	progressWriter.Finish()
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("写入订阅文件失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("关闭订阅文件失败: %w", closeErr)
	}
	if written == 0 {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("订阅下载内容为空")
	}
	data, err := os.ReadFile(downloadPath)
	if err != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("读取订阅下载文件失败: %w", err)
	}
	if err := ValidateSubscriptionData(data); err != nil {
		_ = os.Remove(downloadPath)
		return subscriptionValidationError{err: err}
	}

	if err := os.Rename(downloadPath, targetPath); err != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("保存订阅文件失败: %w", err)
	}

	fmt.Printf("订阅下载完成: %s (%d bytes)\n", targetPath, written)
	return nil
}

type subscriptionValidationError struct {
	err error
}

func (e subscriptionValidationError) Error() string {
	return e.err.Error()
}

func (e subscriptionValidationError) Unwrap() error {
	return e.err
}

func DefaultSubscriptionName(rawURL string, fileName string) string {
	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		if base := path.Base(parsedURL.Path); base != "." && base != "/" && base != "" {
			return strings.TrimSuffix(base, path.Ext(base))
		}
		if parsedURL.Hostname() != "" {
			return parsedURL.Hostname()
		}
	}
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func randomHex(byteCount int) (string, error) {
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机值失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
