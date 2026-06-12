package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"snailproxy/internal/github"
	"snailproxy/internal/progress"
)

func AssetPath(asset github.Asset) string {
	return filepath.Join(downloadDir(), asset.Name)
}

func DownloadAsset(ctx context.Context, asset github.Asset) (string, error) {
	if asset.BrowserDownloadURL == "" {
		return "", fmt.Errorf("asset %q 没有 browser_download_url", asset.Name)
	}

	tempDir := downloadDir()
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	targetPath := AssetPath(asset)
	downloadPath := targetPath + ".download"
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

	out, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("创建下载文件失败: %w", err)
	}

	total := resp.ContentLength
	if total <= 0 && asset.Size > 0 {
		total = asset.Size
	}
	progressWriter := progress.NewWriter(os.Stdout, "下载进度", total)
	written, err := io.Copy(io.MultiWriter(out, progressWriter), resp.Body)
	progressWriter.Finish()
	closeErr := out.Close()
	if err != nil {
		return "", fmt.Errorf("写入下载文件失败: %w", err)
	}
	if closeErr != nil {
		return "", fmt.Errorf("关闭下载文件失败: %w", closeErr)
	}

	if err := VerifyAssetFile(downloadPath, asset); err != nil {
		_ = os.Remove(downloadPath)
		return "", err
	}

	if err := os.Rename(downloadPath, targetPath); err != nil {
		_ = os.Remove(downloadPath)
		return "", fmt.Errorf("保存下载文件失败: %w", err)
	}

	fmt.Printf("下载完成: %s (%d bytes)\n", targetPath, written)
	return targetPath, nil
}

func VerifyAssetFile(path string, asset github.Asset) error {
	expected, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(asset.Digest)), "sha256:")
	if !ok || expected == "" {
		fmt.Println("未提供 sha256 摘要，跳过校验。")
		return nil
	}

	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开文件进行 sha256 校验失败: %w", err)
	}
	defer in.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, in); err != nil {
		return fmt.Errorf("读取文件进行 sha256 校验失败: %w", err)
	}

	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("sha256 校验失败: 期望 %s，实际 %s", expected, actual)
	}

	fmt.Printf("sha256 校验通过: %s\n", actual)
	return nil
}

func downloadDir() string {
	return filepath.Join(os.TempDir(), "mihomo")
}
