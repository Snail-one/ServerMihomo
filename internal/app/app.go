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
	"snailproxy/internal/mihomo"
	"snailproxy/internal/platform"
	"snailproxy/internal/ui"
	"snailproxy/resources"
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
	case ui.ActionDownloadSubscription:
		return downloadOrUpdateSubscription(ctx)
	case ui.ActionSelectSubscription:
		return selectAndApplySubscription(ctx)
	case ui.ActionReleaseResources:
		return releaseLocalResourcePackage()
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
	_ = ctx

	store := mihomo.NewStore()
	if err := store.EnsureDirs(); err != nil {
		return err
	}

	subscriptions, err := store.LoadSubscriptions()
	if err != nil {
		return err
	}
	if len(subscriptions) == 0 {
		return fmt.Errorf("还没有订阅，请先选择“下载/更新 Clash 订阅”")
	}

	index, err := ui.SelectSubscription(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}
	if index < 0 {
		fmt.Println("已返回。")
		return nil
	}

	overwriteBase := false
	if fileExists(store.BaseConfigPath()) {
		overwriteBase, err = ui.ConfirmOverwriteDefaultConfig(store.BaseConfigPath())
		if err != nil {
			return err
		}
	}

	ensureResult, err := store.EnsureDefaultFiles(overwriteBase)
	if err != nil {
		return err
	}
	printEnsureResult(ensureResult)

	result, err := store.ApplySubscription(mihomo.ApplyOptions{
		Subscription:        subscriptions[index],
		GenerateProxyGroups: true,
	})
	if err != nil {
		return err
	}

	fmt.Printf("最终配置已生成: %s\n", result.FinalConfigPath)
	fmt.Printf("订阅代理数量: %d，自定义代理数量: %d，代理组数量: %d\n", result.ProxyCount, result.CustomProxyCount, result.GroupCount)
	for _, warning := range result.Warnings {
		fmt.Printf("提示: %s\n", warning)
	}
	return nil
}

func releaseLocalResourcePackage() error {
	store := mihomo.NewStore()
	overwrite, err := ui.ConfirmOverwriteResourcePackage(store.BaseDir)
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
		fmt.Printf("本地资源包没有可释放文件: resources/mihomo\n")
		return nil
	}
	fmt.Printf("本地资源包已释放到: %s\n", result.TargetDir)
	for _, path := range result.Released {
		fmt.Printf("已释放: %s\n", path)
	}
	for _, path := range result.Skipped {
		fmt.Printf("已跳过已有文件: %s\n", path)
	}
	return nil
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

func printEnsureResult(result mihomo.EnsureResult) {
	for _, path := range result.Created {
		fmt.Printf("已创建默认文件: %s\n", path)
	}
	for _, path := range result.Overwritten {
		fmt.Printf("已覆盖默认文件: %s\n", path)
	}
	for _, path := range result.Skipped {
		fmt.Printf("已保留现有文件: %s\n", path)
	}
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
