package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"snailproxy/internal/github"
)

func DownloadAsset(ctx context.Context, asset github.Asset) (string, error) {
	if asset.BrowserDownloadURL == "" {
		return "", fmt.Errorf("asset %q 没有 browser_download_url", asset.Name)
	}

	tempDir := filepath.Join(os.TempDir(), "snailproxy-mihomo")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	targetPath := filepath.Join(tempDir, asset.Name)
	fmt.Printf("开始下载: %s\n", asset.Name)
	fmt.Printf("下载 URL: %s\n", asset.BrowserDownloadURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "snailproxy-mihomo-installer")

	proxyURL, proxyErr := http.ProxyFromEnvironment(req)
	if proxyErr != nil {
		fmt.Printf("当前下载方式: 直连（代理配置无效: %v）\n", proxyErr)
	} else if proxyURL != nil {
		fmt.Printf("当前下载方式: 使用代理 %s\n", proxyURL.String())
	} else {
		fmt.Println("当前下载方式: 直连")
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载返回状态码: %s", resp.Status)
	}

	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("创建下载文件失败: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("写入下载文件失败: %w", err)
	}

	fmt.Printf("下载完成: %s (%d bytes)\n", targetPath, written)
	return targetPath, nil
}
