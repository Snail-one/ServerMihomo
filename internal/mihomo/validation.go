package mihomo

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ValidateSubscriptionData(data []byte) error {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("订阅 YAML 无效: %w", err)
	}

	proxiesValue, ok := root["proxies"]
	if !ok {
		return fmt.Errorf("订阅 YAML 缺少 proxies 字段")
	}
	proxies, ok := proxiesValue.([]any)
	if !ok {
		return fmt.Errorf("订阅 YAML 的 proxies 必须是列表")
	}
	if len(proxies) == 0 {
		return fmt.Errorf("订阅 YAML 的 proxies 不能为空")
	}

	for _, proxy := range proxies {
		fields, ok := proxy.(map[string]any)
		if !ok {
			continue
		}
		name, ok := fields["name"].(string)
		if ok && strings.TrimSpace(name) != "" {
			return nil
		}
	}
	return fmt.Errorf("订阅 YAML 的 proxies 中没有可用 name")
}
