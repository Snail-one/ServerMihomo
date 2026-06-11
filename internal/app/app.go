package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"snailproxy/internal/downloader"
	"snailproxy/internal/github"
	"snailproxy/internal/platform"
	"snailproxy/internal/ui"
)

const latestReleaseURL = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
const linuxInstalledBinary = "/opt/mihomo/mihomo"

func Run(ctx context.Context, args []string) error {
	fmt.Printf("当前系统: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	if err := platform.RequireSudo(); err != nil {
		return err
	}

	for {
		action, err := selectPlatformAction()
		if err != nil {
			return err
		}
		if action == ui.ActionExit {
			fmt.Println("已退出。")
			return nil
		}

		if err := runAction(ctx, action); err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println("操作完成。")
		}
		fmt.Println()
	}
}

func runAction(ctx context.Context, action ui.Action) error {
	switch action {
	case ui.ActionDownload:
		return downloadAndPrepareMihomo(ctx)
	case ui.ActionInstallService:
		return installMihomoService(ctx)
	case ui.ActionUninstall:
		return platform.Uninstall(ctx)
	default:
		return fmt.Errorf("未知操作: %d", action)
	}
}

func selectPlatformAction() (ui.Action, error) {
	switch runtime.GOOS {
	case "linux":
		return ui.SelectLinuxAction()
	case "windows":
		return ui.SelectWindowsAction()
	default:
		return ui.ActionExit, fmt.Errorf("暂不支持当前系统: %s", runtime.GOOS)
	}
}

func downloadAndPrepareMihomo(ctx context.Context) error {
	printLocalMihomoVersion(ctx)

	assets, err := fetchMihomoAssets(ctx)
	if err != nil {
		return err
	}

	prepareBinary, overwriteBinary, err := checkInstalledBinary()
	if err != nil {
		return err
	}
	if !prepareBinary {
		return nil
	}

	asset, err := ui.SelectAsset(assets)
	if err != nil {
		return err
	}

	filePath := downloader.AssetPath(asset)
	if fileExists(filePath) {
		overwrite, err := ui.ConfirmOverwrite(filePath)
		if err != nil {
			return err
		}
		if !overwrite {
			if err := downloader.VerifyAssetFile(filePath, asset); err != nil {
				return err
			}
			fmt.Printf("跳过下载，使用本地文件: %s\n", filePath)
			return prepareMihomoBinaryIfNeeded(ctx, filePath, asset.Name, overwriteBinary)
		}
	}

	filePath, err = downloader.DownloadAsset(ctx, asset)
	if err != nil {
		return err
	}
	fmt.Printf("下载目录位置: %s\n", filePath)
	return prepareMihomoBinaryIfNeeded(ctx, filePath, asset.Name, overwriteBinary)
}

func printLocalMihomoVersion(ctx context.Context) {
	if runtime.GOOS != "linux" {
		return
	}
	if !fileExists(linuxInstalledBinary) {
		fmt.Printf("当前本地版本: 未安装（%s 不存在）\n", linuxInstalledBinary)
		return
	}

	fmt.Println("当前本地版本:")
	cmd := exec.CommandContext(ctx, linuxInstalledBinary, "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("读取本地 mihomo 版本失败: %v\n", err)
	}
}

func checkInstalledBinary() (bool, bool, error) {
	if runtime.GOOS != "linux" || !fileExists(linuxInstalledBinary) {
		return true, false, nil
	}

	overwrite, err := ui.ConfirmOverwriteInstall(linuxInstalledBinary)
	if err != nil {
		return false, false, err
	}
	if !overwrite {
		fmt.Printf("跳过选项 1，保留现有程序文件: %s\n", linuxInstalledBinary)
		return false, false, nil
	}

	return true, true, nil
}

func prepareMihomoBinaryIfNeeded(ctx context.Context, filePath string, assetName string, overwrite bool) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}

	return installer.PrepareBinary(ctx, filePath, assetName, overwrite)
}

func installMihomoService(ctx context.Context) error {
	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}

	return installer.InstallService(ctx)
}

func fetchMihomoAssets(ctx context.Context) ([]github.Asset, error) {
	fmt.Printf("正在连接 GitHub API: %s\n", latestReleaseURL)

	release, err := github.FetchLatestRelease(ctx, latestReleaseURL)
	if err != nil {
		return nil, err
	}

	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("release %s 没有可下载的 assets", release.TagName)
	}

	fmt.Printf("最新版本: %s\n", release.TagName)
	assets := filterMihomoAssets(release.Assets)
	if len(assets) == 0 {
		return nil, fmt.Errorf("release %s 没有找到适用于当前系统 %s/%s 的 mihomo 下载包", release.TagName, runtime.GOOS, runtime.GOARCH)
	}

	return assets, nil
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
