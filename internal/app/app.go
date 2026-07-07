package app

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
	"snailproxy/internal/ui"
	"snailproxy/internal/version"
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

func Run(ctx context.Context, args []string) error {
	if handleVersionArg(args) {
		fmt.Println(version.Info())
		return nil
	}

	fmt.Printf("当前 Linux 系统: linux/%s\n", runtime.GOARCH)

	if err := platform.RequireSudo(); err != nil {
		return err
	}

	for {
		action, err := ui.SelectMainAction()
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
		return handleInstall(ctx)
	case ui.ActionManageSubscription:
		return handleSubscription(ctx)
	case ui.ActionManageMihomoService:
		return handleService(ctx)
	case ui.ActionUninstall:
		return platform.Uninstall(ctx)
	default:
		return fmt.Errorf("未知操作: %d", action)
	}
}

func handleInstall(ctx context.Context) error {
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

func handleService(ctx context.Context) error {
	action, err := ui.SelectServiceAction()
	if err != nil {
		return err
	}
	if action == ui.ServiceReturn {
		fmt.Println("已返回。")
		return nil
	}

	installer, err := platform.NewInstaller()
	if err != nil {
		return err
	}

	switch action {
	case ui.ServiceStart:
		return installer.StartService(ctx)
	case ui.ServiceRestart:
		return installer.RestartService(ctx)
	case ui.ServiceStop:
		return installer.StopService(ctx)
	case ui.ServiceWriteProxyEnv:
		return installer.WriteProxyEnvironment(ctx)
	case ui.ServiceClearProxyEnv:
		return installer.ClearProxyEnvironment(ctx)
	default:
		return fmt.Errorf("未知 mihomo 服务操作: %d", action)
	}
}

func handleSubscription(ctx context.Context) error {
	store, subscriptions, err := loadSubscriptionStore()
	if err != nil {
		return err
	}

	action, err := ui.SelectSubscriptionAction(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}

	switch action {
	case ui.SubscriptionReturn:
		fmt.Println("已返回。")
		return nil
	case ui.SubscriptionCreate:
		return downloadNewSubscription(ctx, store, subscriptions)
	case ui.SubscriptionUpdate:
		index, ok, err := selectSubscriptionTarget("更新", subscriptions)
		if err != nil || !ok {
			return err
		}
		return updateExistingSubscription(ctx, store, subscriptions, index)
	case ui.SubscriptionModify:
		index, ok, err := selectSubscriptionTarget("修改", subscriptions)
		if err != nil || !ok {
			return err
		}
		return modifyExistingSubscription(ctx, store, subscriptions, index)
	case ui.SubscriptionDelete:
		index, ok, err := selectSubscriptionTarget("删除", subscriptions)
		if err != nil || !ok {
			return err
		}
		return deleteExistingSubscription(store, subscriptions, index)
	case ui.SubscriptionApply:
		index, ok, err := selectSubscriptionTarget("应用", subscriptions)
		if err != nil || !ok {
			return err
		}
		return applyExistingSubscription(ctx, store, subscriptions, index)
	default:
		return fmt.Errorf("未知订阅操作: %d", action)
	}
}

func loadSubscriptionStore() (mihomo.Store, []mihomo.Subscription, error) {
	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return store, nil, err
	}

	subscriptions, err := store.LoadSubscriptions()
	if err != nil {
		return store, nil, err
	}
	return store, subscriptions, nil
}

func selectSubscriptionTarget(actionName string, subscriptions []mihomo.Subscription) (int, bool, error) {
	if len(subscriptions) == 0 {
		fmt.Printf("没有可%s的订阅。\n", actionName)
		return -1, false, nil
	}

	index, err := ui.SelectSubscription(subscriptionLabels(subscriptions))
	if err != nil {
		return -1, false, err
	}
	if index < 0 {
		fmt.Println("已返回。")
		return -1, false, nil
	}
	return index, true, nil
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

func modifyExistingSubscription(ctx context.Context, store mihomo.Store, subscriptions []mihomo.Subscription, index int) error {
	if index < 0 || index >= len(subscriptions) {
		return fmt.Errorf("订阅选择无效")
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

func deleteExistingSubscription(store mihomo.Store, subscriptions []mihomo.Subscription, index int) error {
	if index < 0 || index >= len(subscriptions) {
		return fmt.Errorf("订阅选择无效")
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

func applyExistingSubscription(ctx context.Context, store mihomo.Store, subscriptions []mihomo.Subscription, index int) error {
	if index < 0 || index >= len(subscriptions) {
		return fmt.Errorf("订阅选择无效")
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
