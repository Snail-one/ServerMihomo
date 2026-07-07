package subscription

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"snailproxy/internal/domain/mihomo"
	"snailproxy/internal/feature"
)

type Feature struct{}

func (Feature) ID() string {
	return "subscription"
}

func (Feature) Label() string {
	return "订阅管理"
}

func (Feature) Order() int {
	return 20
}

func (Feature) Run(ctx context.Context, runtime feature.Runtime) error {
	store, subscriptions, err := loadSubscriptionStore(runtime)
	if err != nil {
		return err
	}

	action, err := Select(subscriptionLabels(subscriptions))
	if err != nil {
		return err
	}

	switch action {
	case ActionReturn:
		fmt.Println("已返回。")
		return nil
	case ActionCreate:
		return downloadNewSubscription(ctx, store, subscriptions)
	case ActionUpdate:
		index, ok, err := selectSubscriptionTarget("更新", subscriptions)
		if err != nil || !ok {
			return err
		}
		return updateExistingSubscription(ctx, store, subscriptions, index)
	case ActionModify:
		index, ok, err := selectSubscriptionTarget("修改", subscriptions)
		if err != nil || !ok {
			return err
		}
		return modifyExistingSubscription(ctx, store, subscriptions, index)
	case ActionDelete:
		index, ok, err := selectSubscriptionTarget("删除", subscriptions)
		if err != nil || !ok {
			return err
		}
		return deleteExistingSubscription(store, subscriptions, index)
	case ActionApply:
		index, ok, err := selectSubscriptionTarget("应用", subscriptions)
		if err != nil || !ok {
			return err
		}
		return applyExistingSubscription(ctx, runtime, store, subscriptions, index)
	default:
		return fmt.Errorf("未知订阅操作: %d", action)
	}
}

func loadSubscriptionStore(runtime feature.Runtime) (mihomo.Store, []mihomo.Subscription, error) {
	store := runtime.NewMihomoStore()
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

	index, err := SelectSubscription(subscriptionLabels(subscriptions))
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
	subscriptionURL, err := PromptSubscriptionURL()
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

	name, err := PromptSubscriptionName(mihomo.DefaultSubscriptionName(subscriptionURL, fileName))
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

	name, err := PromptSubscriptionName(subscription.Name)
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
	subscriptionURL, err := PromptSubscriptionURLDefault(subscription.URL)
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
	name, err := PromptSubscriptionName(defaultName)
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
	confirmed, err := ConfirmDeleteSubscription(label)
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

func applyExistingSubscription(ctx context.Context, runtime feature.Runtime, store mihomo.Store, subscriptions []mihomo.Subscription, index int) error {
	if index < 0 || index >= len(subscriptions) {
		return fmt.Errorf("订阅选择无效")
	}

	finalConfigPath, err := store.CopySubscriptionToFinalConfig(subscriptions[index])
	if err != nil {
		return err
	}

	fmt.Printf("订阅配置已原样应用到: %s\n", finalConfigPath)

	manager, err := runtime.NewPlatformManager()
	if err != nil {
		return err
	}
	fmt.Println("正在重启 mihomo 服务以应用配置...")
	return manager.RestartService(ctx)
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
