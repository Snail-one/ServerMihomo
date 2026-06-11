package app

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"snailproxy/internal/downloader"
	"snailproxy/internal/github"
	"snailproxy/internal/platform"
	"snailproxy/internal/ui"
)

const latestReleaseURL = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"

func Run(ctx context.Context, args []string) error {
	fmt.Printf("当前系统: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	if err := platform.RequireSudo(); err != nil {
		return err
	}

	action, err := ui.SelectAction()
	if err != nil {
		return err
	}
	if action == ui.ActionExit {
		fmt.Println("已退出。")
		return nil
	}

	return installMihomo(ctx)
}

func installMihomo(ctx context.Context) error {
	fmt.Printf("正在连接 GitHub API: %s\n", latestReleaseURL)

	release, err := github.FetchLatestRelease(ctx, latestReleaseURL)
	if err != nil {
		return err
	}

	if len(release.Assets) == 0 {
		return fmt.Errorf("release %s 没有可下载的 assets", release.TagName)
	}

	fmt.Printf("最新版本: %s\n", release.TagName)
	assets := filterMihomoAssets(release.Assets)
	if len(assets) == 0 {
		return fmt.Errorf("release %s 没有找到适用于当前系统 %s/%s 的 mihomo 下载包", release.TagName, runtime.GOOS, runtime.GOARCH)
	}

	asset, err := ui.SelectAsset(assets)
	if err != nil {
		return err
	}

	filePath, err := downloader.DownloadAsset(ctx, asset)
	if err != nil {
		return err
	}
	fmt.Printf("下载目录位置: %s\n", filePath)

	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}

	return installer.Install(ctx, filePath, asset.Name)
}

func filterMihomoAssets(assets []github.Asset) []github.Asset {
	result := make([]github.Asset, 0, len(assets))
	for _, asset := range assets {
		if asset.BrowserDownloadURL == "" {
			continue
		}
		if isCurrentPlatformMihomoAsset(asset.Name) && isSupportedArchive(asset.Name) {
			result = append(result, asset)
		}
	}
	return result
}

func isCurrentPlatformMihomoAsset(name string) bool {
	parts := strings.Split(strings.ToLower(name), "-")
	if len(parts) < 3 || parts[0] != "mihomo" {
		return false
	}

	return parts[1] == runtime.GOOS && matchesCurrentArch(parts[2])
}

func matchesCurrentArch(assetArch string) bool {
	if runtime.GOARCH == "arm" {
		return assetArch == "arm" || strings.HasPrefix(assetArch, "armv")
	}
	return assetArch == runtime.GOARCH
}

func isSupportedArchive(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".gz") ||
		strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz")
}
