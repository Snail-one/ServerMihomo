//go:build ignore

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const targetDir = "mihomo"

var resourceFiles = []struct {
	name string
	url  string
}{
	{
		name: "geoip.metadb",
		url:  "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/release/geoip.metadb",
	},
	{
		name: "geosite.dat",
		url:  "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/release/geosite.dat",
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "下载本地资源包失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("创建资源目录失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 90 * time.Second}
	for _, resource := range resourceFiles {
		targetPath := filepath.Join(targetDir, resource.name)
		if err := downloadResource(ctx, client, resource.url, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func downloadResource(ctx context.Context, client *http.Client, rawURL string, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "snailproxy-resource-downloader")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载 %s 失败: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载 %s 返回状态码: %s", rawURL, resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建资源目录失败: %w", err)
	}

	downloadPath := targetPath + ".download"
	out, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建下载文件失败: %w", err)
	}

	written, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("写入 %s 失败: %w", targetPath, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("关闭 %s 失败: %w", targetPath, closeErr)
	}
	if written == 0 {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("下载内容为空: %s", rawURL)
	}

	if err := os.Rename(downloadPath, targetPath); err != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("保存 %s 失败: %w", targetPath, err)
	}

	fmt.Printf("已下载资源: %s (%d bytes)\n", targetPath, written)
	return nil
}
