package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	metacubexdArchiveURL = "https://github.com/MetaCubeX/metacubexd/archive/refs/heads/gh-pages.zip"
)

var targetDirFlag = flag.String("target", filepath.Join("internal", "assets", "mihomo"), "offline resource output directory")

type mihomoDownloadSource struct {
	name   string
	apiURL string
}

var stableMihomoSource = mihomoDownloadSource{
	name:   "正式版",
	apiURL: "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest",
}

var alphaMihomoSource = mihomoDownloadSource{
	name:   "开发版",
	apiURL: "https://api.github.com/repos/MetaCubeX/mihomo/releases/tags/Prerelease-Alpha",
}

var resourceFiles = []struct {
	name string
	url  string
}{
	{
		name: "geoip.metadb",
		url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/release/geoip.metadb",
	},
	{
		name: "geosite.dat",
		url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/release/geosite.dat",
	},
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func main() {
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: resourcegen [-target dir]")
		os.Exit(2)
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "下载本地安装资源失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	targetDir := filepath.Clean(strings.TrimSpace(*targetDirFlag))
	if targetDir == "" || targetDir == "." {
		return fmt.Errorf("资源目录不能为空")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("创建资源目录失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 120 * time.Second}
	for _, resource := range resourceFiles {
		targetPath := filepath.Join(targetDir, resource.name)
		if err := downloadResource(ctx, client, resource.url, targetPath); err != nil {
			return err
		}
	}

	if err := downloadMetaCubeXD(ctx, client, filepath.Join(targetDir, "metacubexd")); err != nil {
		return err
	}
	if err := downloadMihomoPackages(ctx, client, filepath.Join(targetDir, "packages")); err != nil {
		return err
	}
	return nil
}

func downloadMetaCubeXD(ctx context.Context, client *http.Client, metacubexdDir string) error {
	if err := os.RemoveAll(metacubexdDir); err != nil {
		return fmt.Errorf("清理 metacubexd 目录失败: %w", err)
	}
	if err := os.MkdirAll(metacubexdDir, 0o755); err != nil {
		return fmt.Errorf("创建 metacubexd 目录失败: %w", err)
	}

	archivePath := filepath.Join(os.TempDir(), "metacubexd.zip")
	if err := downloadResource(ctx, client, metacubexdArchiveURL, archivePath); err != nil {
		return err
	}
	if err := extractZipStripTopDir(archivePath, metacubexdDir); err != nil {
		return err
	}
	_ = os.Remove(archivePath)
	fmt.Printf("已解压 metacubexd 到: %s\n", metacubexdDir)
	return nil
}

func downloadMihomoPackages(ctx context.Context, client *http.Client, packagesDir string) error {
	if err := os.RemoveAll(packagesDir); err != nil {
		return fmt.Errorf("清理 mihomo 安装包目录失败: %w", err)
	}
	if err := os.MkdirAll(packagesDir, 0o755); err != nil {
		return fmt.Errorf("创建 mihomo 安装包目录失败: %w", err)
	}

	source := selectedMihomoSource()
	release, err := fetchRelease(ctx, client, source.apiURL)
	if err != nil {
		return fmt.Errorf("获取 mihomo GitHub API 失败，无法下载带 sha256 校验的安装包: %w", err)
	}
	fmt.Printf("mihomo 下载信息来源: GitHub API（%s，包含 sha256，可校验下载包）\n", source.name)
	fmt.Printf("mihomo 最新版本: %s\n", release.TagName)

	for _, assetName := range mihomoAssetNames(release.TagName) {
		asset, ok := findAsset(release.Assets, assetName)
		if !ok {
			return fmt.Errorf("release %s 没有找到安装包 %s", release.TagName, assetName)
		}
		if asset.BrowserDownloadURL == "" {
			return fmt.Errorf("release %s 的资源 %s 缺少下载地址", release.TagName, asset.Name)
		}

		targetPath := filepath.Join(packagesDir, asset.Name)
		if err := downloadResource(ctx, client, asset.BrowserDownloadURL, targetPath); err != nil {
			return err
		}
		if err := verifyFileDigest(targetPath, asset.Digest); err != nil {
			return err
		}
	}
	return nil
}

func selectedMihomoSource() mihomoDownloadSource {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MIHOMO_RELEASE_CHANNEL"))) {
	case "alpha", "dev", "development", "prerelease-alpha":
		return alphaMihomoSource
	default:
		return stableMihomoSource
	}
}

func mihomoAssetNames(tagName string) []string {
	version := strings.TrimPrefix(strings.TrimSpace(tagName), "v")
	return []string{
		"mihomo-linux-amd64-v3-v" + version + ".gz",
		"mihomo-linux-arm64-v" + version + ".gz",
	}
}

func fetchRelease(ctx context.Context, client *http.Client, apiURL string) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "snailproxy-resource-downloader")

	proxyURL, proxyErr := http.ProxyFromEnvironment(req)
	if proxyErr != nil {
		fmt.Printf("mihomo GitHub API 访问方式: 直连（代理配置无效: %v）\n", proxyErr)
	} else if proxyURL != nil {
		fmt.Printf("mihomo GitHub API 访问方式: 使用代理 %s\n", proxyURL.String())
	} else {
		fmt.Println("mihomo GitHub API 访问方式: 直连")
	}

	release, err := fetchReleaseWithClient(client, req)
	if err == nil {
		return release, nil
	}
	if proxyErr != nil || proxyURL == nil {
		return githubRelease{}, err
	}

	fmt.Printf("通过代理获取 mihomo GitHub API 失败，切换为直连: %v\n", err)
	directTransport := http.DefaultTransport.(*http.Transport).Clone()
	directTransport.Proxy = nil
	directClient := &http.Client{
		Timeout:   client.Timeout,
		Transport: directTransport,
	}
	release, directErr := fetchReleaseWithClient(directClient, req)
	if directErr != nil {
		return githubRelease{}, fmt.Errorf("直连请求 mihomo 最新版本失败: %w", directErr)
	}
	return release, nil
}

func fetchReleaseWithClient(client *http.Client, req *http.Request) (githubRelease, error) {
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("请求 mihomo 最新版本失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubRelease{}, fmt.Errorf("请求 mihomo 最新版本返回状态码: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("解析 mihomo 最新版本失败: %w", err)
	}
	return release, nil
}

func findAsset(assets []githubAsset, wantName string) (githubAsset, bool) {
	for _, asset := range assets {
		if asset.Name == wantName {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func downloadResource(ctx context.Context, client *http.Client, rawURL string, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "snailproxy-resource-downloader")

	fmt.Printf("准备下载资源: %s\n", targetPath)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载 %s 失败: %w", rawURL, err)
	}
	defer resp.Body.Close()

	actualURL := actualDownloadURL(resp, rawURL)
	fmt.Printf("实际下载地址: %s\n", actualURL)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载 %s 返回状态码: %s", actualURL, resp.Status)
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
		return fmt.Errorf("下载内容为空: %s", actualURL)
	}

	if err := os.Rename(downloadPath, targetPath); err != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("保存 %s 失败: %w", targetPath, err)
	}

	fmt.Printf("已下载资源: %s (%d bytes)\n", targetPath, written)
	return nil
}

func actualDownloadURL(resp *http.Response, fallback string) string {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return fallback
}

func verifyFileDigest(targetPath string, digest string) error {
	expected := normalizedSHA256(digest)
	if expected == "" {
		fmt.Printf("资源 %s 未提供 sha256，跳过校验。\n", filepath.Base(targetPath))
		return nil
	}

	input, err := os.Open(targetPath)
	if err != nil {
		return fmt.Errorf("打开 %s 进行 sha256 校验失败: %w", targetPath, err)
	}
	defer input.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return fmt.Errorf("读取 %s 进行 sha256 校验失败: %w", targetPath, err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("sha256 校验失败 %s: 期望 %s，实际 %s", targetPath, expected, actual)
	}
	fmt.Printf("sha256 校验通过: %s\n", actual)
	return nil
}

func normalizedSHA256(digest string) string {
	expected, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(digest)), "sha256:")
	if !ok {
		return ""
	}
	return expected
}

func extractZipStripTopDir(archivePath string, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开 zip 失败: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		relativePath := stripTopDir(file.Name)
		if relativePath == "" {
			continue
		}
		targetPath, err := safeJoin(targetDir, relativePath)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("创建目录 %s 失败: %w", targetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(targetPath), err)
		}
		if err := extractZipFile(file, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func stripTopDir(name string) string {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." {
		return ""
	}
	parts := strings.SplitN(cleaned, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func safeJoin(root string, relativePath string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(relativePath))
	if cleaned == "." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) || cleaned == ".." {
		return "", fmt.Errorf("zip 文件包含不安全路径: %s", relativePath)
	}
	return filepath.Join(root, cleaned), nil
}

func extractZipFile(file *zip.File, targetPath string) error {
	input, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开 zip 文件 %s 失败: %w", file.Name, err)
	}
	defer input.Close()

	mode := file.FileInfo().Mode()
	if mode&0o111 == 0 {
		mode = 0o644
	} else {
		mode = 0o755
	}
	output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("写入 %s 失败: %w", targetPath, err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("写入 %s 失败: %w", targetPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭 %s 失败: %w", targetPath, closeErr)
	}
	return nil
}
