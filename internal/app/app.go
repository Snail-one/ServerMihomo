package app

import (
	"context"
	"encoding/json"
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
	"snailproxy/internal/ui"
	"snailproxy/internal/version"
	"snailproxy/resources"
)

const latestReleaseURL = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
const linuxInstalledBinary = "/opt/mihomo/mihomo"
const mihomoPackageManifestFile = "manifest.json"

type mihomoPackageManifest struct {
	Source      string                       `json:"source"`
	Method      string                       `json:"method"`
	Version     string                       `json:"version"`
	APIURL      string                       `json:"api_url"`
	GeneratedAt string                       `json:"generated_at"`
	Assets      []mihomoPackageManifestAsset `json:"assets"`
}

type mihomoPackageManifestAsset struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type missingBundledMihomoPackageError struct {
	BaseDir string
	Pattern string
}

func (e missingBundledMihomoPackageError) Error() string {
	expectedPath := filepath.Join(e.BaseDir, "packages", e.Pattern)
	return fmt.Sprintf(
		"本地安装资源中没有当前平台的 mihomo 安装包: %s/%s（期望匹配: %s）；请先运行 go generate ./resources 并重新构建 snailproxy，或在主菜单选择“安装 -> 在线安装”",
		runtime.GOOS,
		runtime.GOARCH,
		expectedPath,
	)
}

func Run(ctx context.Context, args []string) error {
	if handleVersionArg(args) {
		fmt.Println(version.Info())
		return nil
	}

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

func handleVersionArg(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch strings.TrimSpace(args[0]) {
	case "-v", "--version", "version":
		return true
	default:
		return false
	}
}

func runAction(ctx context.Context, action ui.Action) error {
	switch action {
	case ui.ActionInstall:
		return installMihomo(ctx)
	case ui.ActionDownload:
		return downloadAndPrepareMihomo(ctx)
	case ui.ActionInstallService:
		return installMihomoService(ctx)
	case ui.ActionDownloadSubscription:
		return downloadOrUpdateSubscription(ctx)
	case ui.ActionApplySubscription:
		return selectAndApplySubscription(ctx)
	case ui.ActionLocalInstall:
		return localInstall()
	case ui.ActionVerifyLocalMihomo:
		return verifyLocalMihomo(ctx)
	case ui.ActionManageMihomoService:
		return manageMihomoService(ctx)
	case ui.ActionUninstall:
		return platform.Uninstall(ctx)
	default:
		return fmt.Errorf("未知操作: %d", action)
	}
}

func installMihomo(ctx context.Context) error {
	action, err := ui.SelectInstallAction()
	if err != nil {
		return err
	}

	switch action {
	case ui.InstallReturn:
		fmt.Println("已返回。")
		return nil
	case ui.InstallLocal:
		return localInstall()
	case ui.InstallOnline:
		return downloadAndPrepareMihomo(ctx)
	case ui.InstallService:
		return installMihomoService(ctx)
	default:
		return fmt.Errorf("未知安装操作: %d", action)
	}
}

func selectPlatformAction() (ui.Action, error) {
	if runtime.GOOS != "linux" {
		return ui.ActionExit, fmt.Errorf("暂不支持当前系统: %s", runtime.GOOS)
	}
	return ui.SelectLinuxAction()
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

func manageMihomoService(ctx context.Context) error {
	action, err := ui.SelectMihomoServiceAction()
	if err != nil {
		return err
	}
	if action == ui.MihomoServiceReturn {
		fmt.Println("已返回。")
		return nil
	}

	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}

	switch action {
	case ui.MihomoServiceStart:
		return installer.StartService(ctx)
	case ui.MihomoServiceRestart:
		return installer.RestartService(ctx)
	case ui.MihomoServiceStop:
		return installer.StopService(ctx)
	case ui.MihomoServiceWriteProxyEnv:
		return installer.WriteProxyEnvironment(ctx)
	case ui.MihomoServiceClearProxyEnv:
		return installer.ClearProxyEnvironment(ctx)
	default:
		return fmt.Errorf("未知 mihomo 服务操作: %d", action)
	}
}

func downloadOrUpdateSubscription(ctx context.Context) error {
	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return err
	}

	subscriptions, err := store.LoadSubscriptions()
	if err != nil {
		return err
	}

	index, action, err := ui.SelectSubscriptionDownloadTarget(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}

	switch action {
	case ui.SubscriptionDownloadReturn:
		fmt.Println("已返回。")
		return nil
	case ui.SubscriptionDownloadCreate:
		return downloadNewSubscription(ctx, store, subscriptions)
	case ui.SubscriptionDownloadDelete:
		return deleteExistingSubscription(store, subscriptions)
	case ui.SubscriptionDownloadModify:
		return modifyExistingSubscription(ctx, store, subscriptions)
	case ui.SubscriptionDownloadUpdate:
		return updateExistingSubscription(ctx, store, subscriptions, index)
	default:
		return fmt.Errorf("未知订阅操作: %d", action)
	}
}

func downloadNewSubscription(ctx context.Context, store mihomo.Store, subscriptions []mihomo.Subscription) error {
	subscriptionURL, err := ui.PromptSubscriptionURL()
	if err != nil {
		return err
	}

	fileName, err := store.NewSubscriptionFileName()
	if err != nil {
		return err
	}
	if err := mihomo.DownloadSubscription(ctx, subscriptionURL, store.ProfilePath(fileName)); err != nil {
		return err
	}

	name, err := ui.PromptSubscriptionName(mihomo.DefaultSubscriptionName(subscriptionURL, fileName))
	if err != nil {
		return err
	}

	subscriptions = append(subscriptions, mihomo.Subscription{
		Name: name,
		File: fileName,
		URL:  subscriptionURL,
	})
	if err := store.SaveSubscriptions(subscriptions); err != nil {
		return err
	}

	fmt.Printf("订阅已保存: %s（%s）\n", name, fileName)
	fmt.Printf("订阅信息: %s\n", store.MetadataPath())
	return nil
}

func updateExistingSubscription(ctx context.Context, store mihomo.Store, subscriptions []mihomo.Subscription, index int) error {
	if index < 0 || index >= len(subscriptions) {
		return fmt.Errorf("订阅选择无效")
	}

	subscription := subscriptions[index]
	if strings.TrimSpace(subscription.URL) == "" {
		return fmt.Errorf("订阅 %s 缺少下载链接", subscription.Name)
	}
	if strings.TrimSpace(subscription.File) == "" {
		return fmt.Errorf("订阅 %s 缺少实际文件名", subscription.Name)
	}

	fmt.Printf("正在更新订阅: %s（%s）\n", subscription.Name, subscription.File)
	if err := mihomo.DownloadSubscription(ctx, subscription.URL, store.ProfilePath(subscription.File)); err != nil {
		return err
	}

	name, err := ui.PromptSubscriptionName(subscription.Name)
	if err != nil {
		return err
	}
	subscription.Name = name
	subscriptions[index] = subscription

	if err := store.SaveSubscriptions(subscriptions); err != nil {
		return err
	}

	fmt.Printf("订阅已更新: %s（%s）\n", subscription.Name, subscription.File)
	return nil
}

func modifyExistingSubscription(ctx context.Context, store mihomo.Store, subscriptions []mihomo.Subscription) error {
	if len(subscriptions) == 0 {
		fmt.Println("没有可修改的订阅。")
		return nil
	}

	index, err := ui.SelectSubscription(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}
	if index < 0 {
		fmt.Println("已返回。")
		return nil
	}

	subscription := subscriptions[index]
	subscriptionURL, err := ui.PromptSubscriptionURLDefault(subscription.URL)
	if err != nil {
		return err
	}

	fileName := strings.TrimSpace(subscription.File)
	if fileName == "" {
		fileName, err = store.NewSubscriptionFileName()
		if err != nil {
			return err
		}
	}

	defaultName := strings.TrimSpace(subscription.Name)
	if defaultName == "" {
		defaultName = mihomo.DefaultSubscriptionName(subscriptionURL, fileName)
	}
	name, err := ui.PromptSubscriptionName(defaultName)
	if err != nil {
		return err
	}

	fmt.Printf("正在使用修改后的链接下载订阅: %s（%s）\n", name, filepath.Base(fileName))
	if err := mihomo.DownloadSubscription(ctx, subscriptionURL, store.ProfilePath(fileName)); err != nil {
		return err
	}

	subscription.Name = name
	subscription.File = filepath.Base(fileName)
	subscription.URL = subscriptionURL
	subscriptions[index] = subscription

	if err := store.SaveSubscriptions(subscriptions); err != nil {
		return err
	}

	fmt.Printf("订阅已修改: %s（%s）\n", subscription.Name, subscription.File)
	return nil
}

func deleteExistingSubscription(store mihomo.Store, subscriptions []mihomo.Subscription) error {
	if len(subscriptions) == 0 {
		fmt.Println("没有可删除的订阅。")
		return nil
	}

	index, err := ui.SelectSubscription(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}
	if index < 0 {
		fmt.Println("已返回。")
		return nil
	}

	subscription := subscriptions[index]
	label := subscriptionLabel(subscription)
	confirmed, err := ui.ConfirmDeleteSubscription(label)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("已取消删除。")
		return nil
	}

	file := strings.TrimSpace(subscription.File)
	if file != "" {
		targetPath := store.ProfilePath(file)
		if err := os.Remove(targetPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("删除订阅文件失败: %w", err)
			}
			fmt.Printf("订阅文件不存在，已跳过: %s\n", targetPath)
		} else {
			fmt.Printf("已删除订阅文件: %s\n", targetPath)
		}
	}

	subscriptions = append(subscriptions[:index], subscriptions[index+1:]...)
	if err := store.SaveSubscriptions(subscriptions); err != nil {
		return err
	}

	fmt.Printf("订阅已删除: %s\n", label)
	return nil
}

func selectAndApplySubscription(ctx context.Context) error {
	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return err
	}

	subscriptions, err := store.LoadSubscriptions()
	if err != nil {
		return err
	}
	if len(subscriptions) == 0 {
		return fmt.Errorf("还没有订阅，请先选择“下载/更新/修改/删除 Clash 订阅”")
	}

	index, err := ui.SelectSubscription(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}
	if index < 0 {
		fmt.Println("已返回。")
		return nil
	}

	finalConfigPath, err := store.CopySubscriptionToFinalConfig(subscriptions[index])
	if err != nil {
		return err
	}

	fmt.Printf("订阅配置已原样应用到: %s\n", finalConfigPath)

	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}
	fmt.Println("正在重启 mihomo 服务以应用配置...")
	return installer.RestartService(ctx)
}

func localInstall() error {
	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return err
	}

	overwrite, err := ui.ConfirmOverwriteLocalInstall(store.BaseDir)
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
			fmt.Printf("缺少当前平台的内置 mihomo 安装包，无法覆盖现有程序文件；已保留: %s\n", targetPath)
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
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "mihomo-linux-amd64-v3-v*.gz"
	case "linux/arm64":
		return "mihomo-linux-arm64-v*.gz"
	default:
		return fmt.Sprintf("mihomo-%s-%s-*", runtime.GOOS, runtime.GOARCH)
	}
}

func mihomoBinaryName() string {
	return "mihomo"
}

func verifyLocalMihomo(ctx context.Context) error {
	store := mihomo.NewStore()
	binaryPath := filepath.Join(store.BaseDir, mihomoBinaryName())
	if !fileExists(binaryPath) {
		return fmt.Errorf("本地 mihomo 程序文件不存在: %s", binaryPath)
	}

	packagePath, err := bundledMihomoPackagePath(store.BaseDir)
	if err != nil {
		return fmt.Errorf("缺少本地安装包，无法验证当前 mihomo: %w", err)
	}
	assetName := filepath.Base(packagePath)

	manifest, err := loadMihomoPackageManifest(store.BaseDir)
	if err != nil {
		return err
	}
	asset, ok := findManifestAssetByName(manifest.Assets, assetName)
	if !ok {
		return fmt.Errorf("本地安装包 manifest 中没有当前平台安装包: %s", assetName)
	}

	fmt.Printf("本地安装包版本: %s（%s，生成于 %s）\n", manifest.Version, manifest.Source, manifest.GeneratedAt)
	if asset.SHA256 == "" {
		return fmt.Errorf("本地安装包 manifest 缺少 sha256，无法验证安装包: %s", assetName)
	}
	actualPackageHash, err := downloader.FileSHA256(packagePath)
	if err != nil {
		return err
	}
	if actualPackageHash != strings.ToLower(strings.TrimSpace(asset.SHA256)) {
		return fmt.Errorf("本地安装包 sha256 验证失败: 期望 %s，实际 %s", asset.SHA256, actualPackageHash)
	}
	fmt.Printf("本地安装包 sha256 验证通过: %s\n", actualPackageHash)

	tempDir, err := os.MkdirTemp("", "snailproxy-mihomo-verify-")
	if err != nil {
		return fmt.Errorf("创建验证临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	expectedBinaryPath, err := archive.ExtractMihomoBinary(packagePath, assetName, tempDir)
	if err != nil {
		return fmt.Errorf("解压已验证安装包失败: %w", err)
	}

	expectedHash, err := downloader.FileSHA256(expectedBinaryPath)
	if err != nil {
		return err
	}
	actualHash, err := downloader.FileSHA256(binaryPath)
	if err != nil {
		return err
	}
	if actualHash != expectedHash {
		return fmt.Errorf("本地 mihomo 文件验证失败: 期望 %s，实际 %s", expectedHash, actualHash)
	}

	fmt.Printf("本地 mihomo 文件验证通过: %s\n", binaryPath)
	fmt.Printf("mihomo sha256: %s\n", actualHash)
	printMihomoFreshnessHint(ctx, manifest)
	return nil
}

func loadMihomoPackageManifest(baseDir string) (mihomoPackageManifest, error) {
	targetPath := filepath.Join(baseDir, "packages", mihomoPackageManifestFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return mihomoPackageManifest{}, fmt.Errorf("读取本地安装包 manifest 失败: %w", err)
	}
	var manifest mihomoPackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return mihomoPackageManifest{}, fmt.Errorf("解析本地安装包 manifest 失败: %w", err)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return mihomoPackageManifest{}, fmt.Errorf("本地安装包 manifest 缺少版本号")
	}
	return manifest, nil
}

func findManifestAssetByName(assets []mihomoPackageManifestAsset, name string) (mihomoPackageManifestAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return mihomoPackageManifestAsset{}, false
}

func printMihomoFreshnessHint(ctx context.Context, manifest mihomoPackageManifest) {
	if strings.TrimSpace(manifest.APIURL) == "" {
		return
	}
	release, err := github.FetchLatestRelease(ctx, manifest.APIURL)
	if err != nil {
		fmt.Printf("提示: 无法检查当前最新版本: %v\n", err)
		return
	}
	if strings.TrimSpace(release.TagName) == strings.TrimSpace(manifest.Version) {
		fmt.Printf("版本状态: 本地安装包版本就是当前最新版本 %s。\n", manifest.Version)
		return
	}
	fmt.Printf("版本状态: 本地安装包版本是 %s，当前最新版本是 %s；本地文件验证通过，但离线包可能已过时。\n", manifest.Version, release.TagName)
}

func subscriptionLabels(subscriptions []mihomo.Subscription) []string {
	labels := make([]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		labels = append(labels, subscriptionLabel(subscription))
	}
	return labels
}

func subscriptionLabel(subscription mihomo.Subscription) string {
	name := strings.TrimSpace(subscription.Name)
	if name == "" {
		name = "未命名订阅"
	}
	file := strings.TrimSpace(subscription.File)
	if file == "" {
		file = "未知文件"
	}
	return fmt.Sprintf("%s（%s）", name, file)
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
