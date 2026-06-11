package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func FetchLatestRelease(ctx context.Context, apiURL string) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "snailproxy-mihomo-installer")

	proxyURL, proxyErr := http.ProxyFromEnvironment(req)
	if proxyErr != nil {
		fmt.Printf("当前访问方式: 直连（代理配置无效: %v）\n", proxyErr)
	} else if proxyURL != nil {
		fmt.Printf("当前访问方式: 使用代理 %s\n", proxyURL.String())
	} else {
		fmt.Println("当前访问方式: 直连")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	release, err := fetchRelease(client, req)
	if err != nil {
		if proxyErr == nil && proxyURL != nil {
			fmt.Printf("通过代理获取 GitHub API 内容失败，切换为直连: %v\n", err)
			directTransport := http.DefaultTransport.(*http.Transport).Clone()
			directTransport.Proxy = nil

			directClient := &http.Client{
				Timeout:   30 * time.Second,
				Transport: directTransport,
			}
			release, err = fetchRelease(directClient, req)
			if err != nil {
				return nil, fmt.Errorf("直连请求 GitHub API 失败: %w", err)
			}
			return release, nil
		}
		return nil, fmt.Errorf("请求 GitHub API 失败: %w", err)
	}

	return release, nil
}

func fetchRelease(client *http.Client, req *http.Request) (*Release, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return decodeReleaseResponse(resp)
}

func decodeReleaseResponse(resp *http.Response) (*Release, error) {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API 返回状态码: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析 GitHub API 响应失败: %w", err)
	}

	return &release, nil
}
