package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"snailproxy/internal/archive"
	"snailproxy/internal/downloader"
	"snailproxy/internal/github"
	"snailproxy/internal/mihomo"
	"snailproxy/internal/platform"
	"snailproxy/internal/ui/installmenu"
	"snailproxy/resources"
)

const latestReleaseURL = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
const linuxInstalledBinary = "/opt/mihomo/mihomo"

type missingBundledMihomoPackageError struct {
	BaseDir string
	Pattern string
}

func (e missingBundledMihomoPackageError) Error() string {
	expectedPath := filepath.Join(e.BaseDir, "packages", e.Pattern)
	return fmt.Sprintf(
		"本地安装资源中没有当前 Linux 架构的 mihomo 安装包: linux/%s（期望匹配: %s）；请先运行 go generate ./resources 并重新构建 snailproxy，或在主菜单选择“安装与更新 -> 在线下载并安装 mihomo”",
		runtime.GOARCH,
		expectedPath,
	)
}

func Run(ctx context.Context) error {
	action, err := installmenu.Select()
	if err != nil {
		return err
	}

	switch action {
	case installmenu.ActionReturn:
		fmt.Println("已返回。")
		return nil
	case installmenu.ActionLocal:
		return localInstall()
	case installmenu.ActionOnline:
		return downloadAndPrepareMihomo(ctx)
	case installmenu.ActionService:
		return installMihomoService(ctx)
	default:
		return fmt.Errorf("未知安装操作: %d", action)
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

	asset, err := installmenu.SelectAsset(assets)
	if err != nil {
		return err
	}

	filePath := downloader.AssetPath(asset)
	if fileExists(filePath) {
		overwrite, err := installmenu.ConfirmOverwrite(filePath)
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
	if !fileExists(linuxInstalledBinary) {
		return true, false, nil
	}

	overwrite, err := installmenu.ConfirmOverwriteInstall(linuxInstalledBinary)
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
	manager, err := platform.NewManager()
	if err != nil {
		return err
	}

	return manager.PrepareBinary(ctx, filePath, assetName, overwrite)
}

func installMihomoService(ctx context.Context) error {
	manager, err := platform.NewManager()
	if err != nil {
		return err
	}

	return manager.InstallService(ctx)
}

func localInstall() error {
	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return err
	}

	overwrite, err := installmenu.ConfirmOverwriteLocalInstall(store.BaseDir)
	if err != nil {
		return err
	}

	result, err := resources.ReleaseMihomoBundle(resources.ReleaseOptions{
		TargetDir: store.BaseDir,
		Overwrite: overwrite,
	})
	if err != nil {
		return err
	}

	if len(result.Released) == 0 && len(result.Skipped) == 0 {
		fmt.Printf("本地安装资源包没有可释放文件: resources/mihomo\n")
	} else {
		fmt.Printf("本地安装资源已释放到: %s\n", result.TargetDir)
		for _, path := range result.Released {
			fmt.Printf("已释放: %s\n", path)
		}
		for _, path := range result.Skipped {
			fmt.Printf("已跳过已有文件: %s\n", path)
		}
	}

	binaryPath, err := installBundledMihomoBinary(store.BaseDir, overwrite)
	if err != nil {
		return err
	}
	if binaryPath != "" {
		fmt.Printf("mihomo 程序文件已安装到: %s\n", binaryPath)
	}
	return nil
}

func installBundledMihomoBinary(baseDir string, overwrite bool) (string, error) {
	targetPath := filepath.Join(baseDir, mihomoBinaryName())
	if fileExists(targetPath) && !overwrite {
		fmt.Printf("已保留现有 mihomo 程序文件: %s\n", targetPath)
		return "", nil
	}

	packagePath, err := bundledMihomoPackagePath(baseDir)
	if err != nil {
		var missingPackage missingBundledMihomoPackageError
		if errors.As(err, &missingPackage) && fileExists(targetPath) {
			fmt.Printf("缺少当前 Linux 架构的内置 mihomo 安装包，无法覆盖现有程序文件；已保留: %s\n", targetPath)
			fmt.Printf("提示: %v\n", missingPackage)
			return "", nil
		}
		return "", err
	}
	binaryPath, err := archive.ExtractMihomoBinary(packagePath, filepath.Base(packagePath), baseDir)
	if err != nil {
		return "", fmt.Errorf("解压内置 mihomo 安装包失败: %w", err)
	}
	if err := os.Chmod(binaryPath, 0o770); err != nil {
		return "", fmt.Errorf("设置 mihomo 程序权限失败: %w", err)
	}
	return binaryPath, nil
}

func bundledMihomoPackagePath(baseDir string) (string, error) {
	pattern := filepath.Join(baseDir, "packages", bundledMihomoPackagePattern())
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", missingBundledMihomoPackageError{
			BaseDir: baseDir,
			Pattern: bundledMihomoPackagePattern(),
		}
	}
	return matches[0], nil
}

func bundledMihomoPackagePattern() string {
	switch runtime.GOARCH {
	case "amd64":
		return "mihomo-linux-amd64-v3-v*.gz"
	case "arm64":
		return "mihomo-linux-arm64-v*.gz"
	default:
		return fmt.Sprintf("mihomo-linux-%s-*", runtime.GOARCH)
	}
}

func mihomoBinaryName() string {
	return "mihomo"
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
		return nil, fmt.Errorf("release %s 没有找到适用于当前 Linux 架构 linux/%s 的 mihomo 下载包", release.TagName, runtime.GOARCH)
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

	return parts[1] == "linux" && matchesCurrentArch(parts[2])
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
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
