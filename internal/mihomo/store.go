package mihomo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"snailproxy/internal/progress"
)

const (
	subscriptionMetadataFile = "subscriptions.json"
	templatesDir             = "templates"
	baseConfigFile           = "mihomo.yaml"
	finalConfigFile          = "config.yaml"
	rulesFile                = "rules.yaml"
	proxyGroupsFile          = "proxygroups.yaml"
	customProxiesFile        = "proxies.yaml"
)

type Store struct {
	BaseDir string
}

type Subscription struct {
	Name string `json:"name"`
	File string `json:"file"`
	URL  string `json:"url"`
}

type EnsureResult struct {
	Created     []string
	Overwritten []string
	Skipped     []string
}

type ApplyOptions struct {
	Subscription        Subscription
	GenerateProxyGroups bool
}

type ApplyResult struct {
	FinalConfigPath  string
	ProxyCount       int
	CustomProxyCount int
	GroupCount       int
	Warnings         []string
}

type groupSpec struct {
	Name        string
	DirectFirst bool
}

func NewStore() Store {
	return Store{BaseDir: DefaultBaseDir()}
}

func DefaultBaseDir() string {
	switch runtime.GOOS {
	case "linux":
		return "/opt/mihomo"
	case "windows":
		if programData := os.Getenv("ProgramData"); programData != "" {
			return filepath.Join(programData, "mihomo")
		}
	}
	return filepath.Join(os.TempDir(), "mihomo")
}

func (s Store) ProfilesDir() string {
	return filepath.Join(s.BaseDir, "profiles")
}

func (s Store) TemplatesDir() string {
	return filepath.Join(s.BaseDir, templatesDir)
}

func (s Store) MetadataPath() string {
	return filepath.Join(s.ProfilesDir(), subscriptionMetadataFile)
}

func (s Store) BaseConfigPath() string {
	return filepath.Join(s.TemplatesDir(), baseConfigFile)
}

func (s Store) FinalConfigPath() string {
	return filepath.Join(s.BaseDir, finalConfigFile)
}

func (s Store) RulesPath() string {
	return filepath.Join(s.TemplatesDir(), rulesFile)
}

func (s Store) ProxyGroupsPath() string {
	return filepath.Join(s.TemplatesDir(), proxyGroupsFile)
}

func (s Store) CustomProxiesPath() string {
	return filepath.Join(s.ProfilesDir(), customProxiesFile)
}

func (s Store) ProfilePath(file string) string {
	return filepath.Join(s.ProfilesDir(), filepath.Base(file))
}

func (s Store) EnsureDirs() error {
	for _, dir := range []string{s.BaseDir, s.ProfilesDir(), s.TemplatesDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}
	return nil
}

func (s Store) LoadSubscriptions() ([]Subscription, error) {
	data, err := os.ReadFile(s.MetadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取订阅信息失败: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}

	var subscriptions []Subscription
	if err := json.Unmarshal(data, &subscriptions); err != nil {
		return nil, fmt.Errorf("解析订阅信息失败: %w", err)
	}
	for i := range subscriptions {
		subscriptions[i].File = filepath.Base(subscriptions[i].File)
	}
	return subscriptions, nil
}

func (s Store) SaveSubscriptions(subscriptions []Subscription) error {
	if err := os.MkdirAll(s.ProfilesDir(), 0o755); err != nil {
		return fmt.Errorf("创建 profiles 目录失败: %w", err)
	}

	normalized := make([]Subscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		subscription.Name = strings.TrimSpace(subscription.Name)
		subscription.File = filepath.Base(strings.TrimSpace(subscription.File))
		subscription.URL = strings.TrimSpace(subscription.URL)
		if subscription.Name == "" || subscription.File == "" || subscription.URL == "" {
			continue
		}
		normalized = append(normalized, subscription)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("编码订阅信息失败: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.MetadataPath(), data, 0o644); err != nil {
		return fmt.Errorf("保存订阅信息失败: %w", err)
	}
	return nil
}

func (s Store) NewSubscriptionFileName() (string, error) {
	if err := os.MkdirAll(s.ProfilesDir(), 0o755); err != nil {
		return "", fmt.Errorf("创建 profiles 目录失败: %w", err)
	}

	for i := 0; i < 20; i++ {
		token, err := randomHex(12)
		if err != nil {
			return "", err
		}
		file := "subscription-" + token + ".yaml"
		if _, err := os.Stat(s.ProfilePath(file)); os.IsNotExist(err) {
			return file, nil
		}
	}
	return "", fmt.Errorf("生成唯一订阅文件名失败")
}

func (s Store) EnsureDefaultFiles(overwriteBase bool) (EnsureResult, error) {
	var result EnsureResult
	if err := s.EnsureDirs(); err != nil {
		return result, err
	}

	basePath := s.BaseConfigPath()
	if fileExists(basePath) {
		if overwriteBase {
			if err := writeDefaultMihomoConfig(basePath); err != nil {
				return result, fmt.Errorf("覆盖默认配置失败: %w", err)
			}
			result.Overwritten = append(result.Overwritten, basePath)
		} else {
			result.Skipped = append(result.Skipped, basePath)
		}
	} else {
		if err := writeDefaultMihomoConfig(basePath); err != nil {
			return result, fmt.Errorf("创建默认配置失败: %w", err)
		}
		result.Created = append(result.Created, basePath)
	}

	defaults := map[string]string{
		s.RulesPath():       defaultRules,
		s.ProxyGroupsPath(): defaultProxyGroups,
	}
	for targetPath, content := range defaults {
		if fileExists(targetPath) {
			result.Skipped = append(result.Skipped, targetPath)
			continue
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return result, fmt.Errorf("创建默认文件 %s 失败: %w", targetPath, err)
		}
		result.Created = append(result.Created, targetPath)
	}

	return result, nil
}

func (s Store) ApplySubscription(options ApplyOptions) (ApplyResult, error) {
	var result ApplyResult
	result.FinalConfigPath = s.FinalConfigPath()

	baseData, err := os.ReadFile(s.BaseConfigPath())
	if err != nil {
		return result, fmt.Errorf("读取基础配置失败: %w", err)
	}

	subscriptionFile := filepath.Base(options.Subscription.File)
	subscriptionPath := s.ProfilePath(subscriptionFile)
	subscriptionData, err := os.ReadFile(subscriptionPath)
	if err != nil {
		return result, fmt.Errorf("读取订阅文件 %s 失败: %w", subscriptionPath, err)
	}

	subscriptionProxyLines, err := extractTopLevelListBlock(string(subscriptionData), "proxies")
	if err != nil {
		return result, fmt.Errorf("解析订阅 proxies 失败: %w", err)
	}
	if len(nonEmptyLines(subscriptionProxyLines)) == 0 {
		return result, fmt.Errorf("订阅文件 %s 没有 proxies 内容", subscriptionPath)
	}

	customProxyLines, customProxyNames, err := s.loadCustomProxies()
	if err != nil {
		return result, err
	}

	subscriptionProxyNames := proxyNamesFromBlockLines(subscriptionProxyLines)
	if len(subscriptionProxyNames) == 0 {
		return result, fmt.Errorf("订阅文件 %s 的 proxies 中没有可用 name", subscriptionPath)
	}

	rulesData, err := os.ReadFile(s.RulesPath())
	if err != nil {
		return result, fmt.Errorf("读取规则文件失败: %w", err)
	}
	if _, err := extractTopLevelListBlock(string(rulesData), "rules"); err != nil {
		return result, fmt.Errorf("解析规则文件失败: %w", err)
	}

	var proxyGroupContent string
	var groupNames []string
	if options.GenerateProxyGroups {
		specs, err := s.loadProxyGroupSpecs()
		if err != nil {
			return result, err
		}
		proxyGroupContent, groupNames = renderProxyGroups(subscriptionProxyNames, customProxyNames, specs)
	}

	warnings := missingRuleGroupWarnings(string(rulesData), groupNames)

	var builder strings.Builder
	builder.WriteString(strings.TrimRight(string(baseData), "\r\n"))
	builder.WriteString("\n\n")
	builder.WriteString("proxies:\n")
	builder.WriteString(renderIndentedBlock(customProxyLines))
	builder.WriteString(renderIndentedBlock(subscriptionProxyLines))
	builder.WriteByte('\n')
	if options.GenerateProxyGroups {
		builder.WriteString(proxyGroupContent)
		builder.WriteByte('\n')
	}
	builder.WriteString(strings.TrimRight(string(rulesData), "\r\n"))
	builder.WriteByte('\n')

	if err := os.WriteFile(s.FinalConfigPath(), []byte(builder.String()), 0o644); err != nil {
		return result, fmt.Errorf("写入最终配置失败: %w", err)
	}

	result.ProxyCount = len(subscriptionProxyNames)
	result.CustomProxyCount = len(customProxyNames)
	result.GroupCount = len(groupNames)
	result.Warnings = warnings
	return result, nil
}

func DownloadSubscription(ctx context.Context, rawURL string, targetPath string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("订阅链接无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("订阅链接只支持 http/https: %s", parsedURL.Scheme)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建订阅目录失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "snailproxy-subscription")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载订阅失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载订阅返回状态码: %s", resp.Status)
	}

	downloadPath := targetPath + ".download"
	out, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建订阅下载文件失败: %w", err)
	}
	progressWriter := progress.NewWriter(os.Stdout, "订阅下载进度", resp.ContentLength)
	written, copyErr := io.Copy(io.MultiWriter(out, progressWriter), resp.Body)
	progressWriter.Finish()
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("写入订阅文件失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("关闭订阅文件失败: %w", closeErr)
	}
	if written == 0 {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("订阅下载内容为空")
	}

	if err := os.Rename(downloadPath, targetPath); err != nil {
		_ = os.Remove(downloadPath)
		return fmt.Errorf("保存订阅文件失败: %w", err)
	}

	fmt.Printf("订阅下载完成: %s (%d bytes)\n", targetPath, written)
	return nil
}

func DefaultSubscriptionName(rawURL string, fileName string) string {
	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		if base := path.Base(parsedURL.Path); base != "." && base != "/" && base != "" {
			return strings.TrimSuffix(base, path.Ext(base))
		}
		if parsedURL.Hostname() != "" {
			return parsedURL.Hostname()
		}
	}
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func (s Store) loadCustomProxies() ([]string, []string, error) {
	customPath := s.CustomProxiesPath()
	data, err := os.ReadFile(customPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("读取自定义代理文件失败: %w", err)
	}

	lines, err := extractTopLevelListBlock(string(data), "proxies")
	if err != nil {
		return nil, nil, fmt.Errorf("解析自定义代理文件 %s 失败: %w", customPath, err)
	}
	names := proxyNamesFromBlockLines(lines)
	return lines, names, nil
}

func (s Store) loadProxyGroupSpecs() ([]groupSpec, error) {
	data, err := os.ReadFile(s.ProxyGroupsPath())
	if err != nil {
		return nil, fmt.Errorf("读取代理组模板失败: %w", err)
	}

	lines, err := extractTopLevelListBlock(string(data), "proxy-groups")
	if err != nil {
		return nil, fmt.Errorf("解析代理组模板失败: %w", err)
	}

	var specs []groupSpec
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		if strings.HasPrefix(value, "- ") || strings.HasPrefix(value, "-\t") {
			value = strings.TrimSpace(value[1:])
		}
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}

		directFirst := false
		switch value[0] {
		case '-':
			directFirst = true
			value = strings.TrimSpace(value[1:])
		case '+':
			value = strings.TrimSpace(value[1:])
		}
		value = unquoteYAMLScalar(value)
		if value == "" || value == "Snail" || value == "Proxies" {
			continue
		}
		specs = append(specs, groupSpec{Name: value, DirectFirst: directFirst})
	}
	return specs, nil
}

func renderProxyGroups(subscriptionProxyNames []string, customProxyNames []string, specs []groupSpec) (string, []string) {
	subscriptionProxyNames = dedupeStrings(subscriptionProxyNames)
	customProxyNames = dedupeStrings(customProxyNames)

	var builder strings.Builder
	var groupNames []string
	builder.WriteString("proxy-groups:\n")

	writeProxyGroup(&builder, "Snail", append([]string{"DIRECT"}, customProxyNames...))
	groupNames = append(groupNames, "Snail")

	proxiesGroup := make([]string, 0, len(subscriptionProxyNames)+2)
	proxiesGroup = append(proxiesGroup, "Snail")
	proxiesGroup = append(proxiesGroup, subscriptionProxyNames...)
	proxiesGroup = append(proxiesGroup, "DIRECT")
	writeProxyGroup(&builder, "Proxies", proxiesGroup)
	groupNames = append(groupNames, "Proxies")

	seen := map[string]bool{"Snail": true, "Proxies": true}
	for _, spec := range specs {
		if seen[spec.Name] {
			continue
		}
		seen[spec.Name] = true
		groupNames = append(groupNames, spec.Name)

		proxies := make([]string, 0, len(subscriptionProxyNames)+3)
		if spec.DirectFirst {
			proxies = append(proxies, "DIRECT", "Snail", "Proxies")
		} else {
			proxies = append(proxies, "Proxies", "Snail", "DIRECT")
		}
		proxies = append(proxies, subscriptionProxyNames...)
		writeProxyGroup(&builder, spec.Name, proxies)
	}

	return strings.TrimRight(builder.String(), "\n"), groupNames
}

func writeProxyGroup(builder *strings.Builder, name string, proxies []string) {
	builder.WriteString("- name: ")
	builder.WriteString(renderYAMLScalar(name))
	builder.WriteByte('\n')
	builder.WriteString("  type: select\n")
	builder.WriteString("  proxies:\n")
	for _, proxy := range dedupeStrings(proxies) {
		builder.WriteString("  - ")
		builder.WriteString(renderYAMLScalar(proxy))
		builder.WriteByte('\n')
	}
}

func missingRuleGroupWarnings(rulesContent string, groupNames []string) []string {
	ruleLines, err := extractTopLevelListBlock(rulesContent, "rules")
	if err != nil {
		return []string{fmt.Sprintf("检查规则代理组失败: %v", err)}
	}

	knownGroups := map[string]bool{
		"DIRECT":      true,
		"REJECT":      true,
		"REJECT-DROP": true,
		"PASS":        true,
	}
	for _, name := range groupNames {
		knownGroups[name] = true
	}

	missing := map[string]bool{}
	for _, ruleLine := range ruleLines {
		rule := strings.TrimSpace(ruleLine)
		if strings.HasPrefix(rule, "-") {
			rule = strings.TrimSpace(rule[1:])
		}
		if rule == "" || strings.HasPrefix(rule, "#") {
			continue
		}

		target := ruleTarget(rule)
		if target == "" || knownGroups[target] {
			continue
		}
		missing[target] = true
	}

	if len(missing) == 0 {
		return nil
	}

	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)

	warnings := make([]string, 0, len(names))
	for _, name := range names {
		warnings = append(warnings, fmt.Sprintf("规则缺少对应的代理组: %s", name))
	}
	return warnings
}

func ruleTarget(rule string) string {
	fields := splitCSVLike(rule)
	if len(fields) == 0 {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(fields[0])) {
	case "MATCH", "FINAL":
		if len(fields) >= 2 {
			return unquoteYAMLScalar(strings.TrimSpace(fields[1]))
		}
	default:
		if len(fields) >= 3 {
			return unquoteYAMLScalar(strings.TrimSpace(fields[2]))
		}
	}
	return ""
}

func splitCSVLike(value string) []string {
	var fields []string
	var builder strings.Builder
	var quote rune
	for _, r := range value {
		switch {
		case quote != 0:
			builder.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
			builder.WriteRune(r)
		case r == ',':
			fields = append(fields, strings.TrimSpace(builder.String()))
			builder.Reset()
		default:
			builder.WriteRune(r)
		}
	}
	fields = append(fields, strings.TrimSpace(builder.String()))
	return fields
}

func extractTopLevelListBlock(content string, key string) ([]string, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	inBlock := false
	var block []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if !inBlock {
			if isTopLevelKeyLine(line, key) {
				rest := strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
				rest = stripYAMLComment(rest)
				if rest == "" {
					inBlock = true
					continue
				}
				if rest == "[]" {
					return nil, nil
				}
				return nil, fmt.Errorf("%s 暂不支持行内列表写法", key)
			}
			continue
		}

		if trimmed != "" && !startsWithWhitespace(line) && looksLikeAnyTopLevelKey(line) {
			break
		}
		block = append(block, line)
	}

	if !inBlock {
		return nil, fmt.Errorf("缺少 %s 字段", key)
	}
	return trimEmptyEdges(block), nil
}

func isTopLevelKeyLine(line string, key string) bool {
	if startsWithWhitespace(line) {
		return false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
	return strings.HasPrefix(trimmed, key+":")
}

func stripYAMLComment(value string) string {
	for i, r := range value {
		if r == '#' && (i == 0 || value[i-1] == ' ' || value[i-1] == '\t') {
			return strings.TrimSpace(value[:i])
		}
	}
	return strings.TrimSpace(value)
}

func looksLikeAnyTopLevelKey(line string) bool {
	if startsWithWhitespace(line) {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	colon := strings.IndexByte(trimmed, ':')
	return colon > 0
}

func startsWithWhitespace(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func renderIndentedBlock(lines []string) string {
	lines = trimEmptyEdges(lines)
	if len(lines) == 0 {
		return ""
	}

	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingSpaces(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent < 0 {
		return ""
	}

	var builder strings.Builder
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			builder.WriteByte('\n')
			continue
		}
		if leadingSpaces(line) >= minIndent {
			line = line[minIndent:]
		}
		builder.WriteString("  ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func proxyNamesFromBlockLines(lines []string) []string {
	normalized := normalizeBlockForParsing(lines)
	var names []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		if name := extractProxyName(current); name != "" {
			names = append(names, name)
		}
		current = nil
	}

	for _, line := range normalized {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") {
			flush()
		}
		current = append(current, line)
	}
	flush()

	return dedupeStrings(names)
}

func normalizeBlockForParsing(lines []string) []string {
	lines = trimEmptyEdges(lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingSpaces(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return lines
	}

	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		if leadingSpaces(line) >= minIndent {
			line = line[minIndent:]
		}
		normalized = append(normalized, line)
	}
	return normalized
}

func extractProxyName(lines []string) string {
	item := strings.Join(lines, "\n")
	offset := 0
	for {
		idx := strings.Index(item[offset:], "name:")
		if idx < 0 {
			return ""
		}
		idx += offset
		if !isYAMLKeyBoundary(item, idx) {
			offset = idx + len("name:")
			continue
		}
		return readYAMLScalar(item[idx+len("name:"):])
	}
}

func isYAMLKeyBoundary(value string, idx int) bool {
	if idx == 0 {
		return true
	}
	switch value[idx-1] {
	case ' ', '\t', '\n', '\r', '{', ',', '-':
		return true
	default:
		return false
	}
}

func readYAMLScalar(value string) string {
	value = strings.TrimLeft(value, " \t")
	if value == "" {
		return ""
	}

	switch value[0] {
	case '"':
		for i := 1; i < len(value); i++ {
			if value[i] == '"' && value[i-1] != '\\' {
				quoted := value[:i+1]
				unquoted, err := strconv.Unquote(quoted)
				if err == nil {
					return strings.TrimSpace(unquoted)
				}
				return strings.Trim(quoted, `"`)
			}
		}
	case '\'':
		for i := 1; i < len(value); i++ {
			if value[i] == '\'' {
				return strings.TrimSpace(strings.ReplaceAll(value[1:i], "''", "'"))
			}
		}
	}

	end := len(value)
	for i, r := range value {
		if r == ',' || r == '}' || r == '\n' || r == '\r' {
			end = i
			break
		}
		if r == '#' && i > 0 && (value[i-1] == ' ' || value[i-1] == '\t') {
			end = i
			break
		}
	}
	return strings.TrimSpace(value[:end])
}

func unquoteYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if value[0] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	if value[0] == '\'' && strings.HasSuffix(value, "'") {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return value
}

var plainScalarPattern = regexp.MustCompile(`^[A-Za-z0-9_.@:/+-]+$`)

func renderYAMLScalar(value string) string {
	if plainScalarPattern.MatchString(value) && !isReservedYAMLScalar(value) {
		return value
	}
	return strconv.Quote(value)
}

func isReservedYAMLScalar(value string) bool {
	switch strings.ToLower(value) {
	case "true", "false", "null", "~", "yes", "no", "on", "off":
		return true
	default:
		return false
	}
}

func nonEmptyLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

func trimEmptyEdges(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

func leadingSpaces(value string) int {
	count := 0
	for count < len(value) && value[count] == ' ' {
		count++
	}
	return count
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func randomHex(byteCount int) (string, error) {
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机值失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func randomString(alphabet string, length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	if alphabet == "" {
		return "", fmt.Errorf("随机字符集不能为空")
	}

	max := big.NewInt(int64(len(alphabet)))
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("生成随机值失败: %w", err)
		}
		builder.WriteByte(alphabet[int(index.Int64())])
	}
	return builder.String(), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func writeDefaultMihomoConfig(targetPath string) error {
	content, err := defaultMihomoConfig()
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, []byte(content), 0o644)
}

func defaultMihomoConfig() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	username, err := randomString(letters, 10)
	if err != nil {
		return "", err
	}
	password, err := randomString(alnum, 18)
	if err != nil {
		return "", err
	}
	secret, err := randomString(alnum, 100)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(defaultMihomoConfigTemplate, secret, username, password), nil
}

const defaultMihomoConfigTemplate = `mode: rule
mixed-port: 52013
allow-lan: false
log-level: error
ipv6: true
external-controller: 127.0.0.1:52014
secret: %s
external-ui: metacubexd
external-ui-url: "https://github.com/MetaCubeX/metacubexd/archive/refs/heads/gh-pages.zip"
unified-delay: true
authentication:
- %s:%s
lan-allowed-ips:
- 127.0.0.1/32
- 192.168.31.0/24
skip-auth-prefixes:
- 127.0.0.1/32
- 192.168.31.0/24
external-controller-cors:
  allow-private-network: true
  allow-origins:
  - tauri://localhost
  - http://tauri.localhost
  - https://yacd.metacubex.one
  - https://metacubex.github.io
  - https://board.zash.run.place
tun:
  auto-detect-interface: true
  auto-route: true
  device: Mihomo
  dns-hijack:
  - any:53
  mtu: 1500
  route-exclude-address: []
  stack: gvisor
  strict-route: true
  enable: false
profile:
  store-selected: true
dns:
  default-nameserver:
  - system
  direct-nameserver:
  - system
  direct-nameserver-follow-policy: false
  enable: true
  enhanced-mode: fake-ip
  fake-ip-filter:
  - '*.lan'
  - '*.local'
  - '*.arpa'
  - time.*.com
  - ntp.*.com
  - time.*.com
  - +.market.xiaomi.com
  - localhost.ptlogin2.qq.com
  - '*.msftncsi.com'
  - www.msftconnecttest.com
  fake-ip-filter-mode: blacklist
  fake-ip-range: 198.18.0.1/16
  fallback: []
  fallback-filter:
    domain:
    - +.google.com
    - +.facebook.com
    - +.youtube.com
    geoip: true
    geoip-code: CN
    ipcidr:
    - 240.0.0.0/4
    - 0.0.0.0/32
  ipv6: true
  listen: :53
  nameserver:
  - https://cloudflare-dns.com/dns-query
  - https://dns.google/dns-query
  prefer-h3: false
  proxy-server-nameserver:
  - https://doh.pub/dns-query
  - https://dns.alidns.com/dns-query
  - tls://223.5.5.5
  respect-rules: true
  use-hosts: false
  use-system-hosts: false
`

const defaultProxyGroups = `proxy-groups:
- -Steam
- -Epic
- -Battle
- -OneDrive
- -Bilibili
- -Kaspersky
- -Ollama
- -Apple
- +Microsoft
- +Spotify
- +YouTube
- +TikTok
- +Netflix
- +Disney
- +GitHub
- +OpenAI
- +Google
- +Grok
- +Anthropic
- +Huggingface
- +X
- +Telegram
- +Pixiv
- +PayPal
- +Final
`

const defaultRules = `rules:
- GEOSITE,private,DIRECT
- GEOIP,private,DIRECT,no-resolve
- GEOIP,lan,DIRECT,no-resolve
- DOMAIN-SUFFIX,cn,DIRECT
- DOMAIN-SUFFIX,dantapi.top,DIRECT
- DOMAIN-SUFFIX,tikdownloader.io,DIRECT
- DOMAIN-SUFFIX,cobalt.tools,DIRECT
- DOMAIN-SUFFIX,cdn-tos-cn.bytedance.net,DIRECT
- GEOSITE,steam@cn,DIRECT
- GEOSITE,category-games@cn,DIRECT
- GEOSITE,douyin,DIRECT
- PROCESS-NAME,nvcontainer.exe,DIRECT
- PROCESS-NAME,OneDrive.exe,OneDrive
- PROCESS-NAME,OneDrive.Sync.Service.exe,OneDrive
- PROCESS-NAME,FileCoAuth.exe,OneDrive
- PROCESS-NAME,Spotify.exe,Spotify
- PROCESS-NAME,steam.exe,Steam
- PROCESS-NAME,steamwebhelper.exe,Steam
- PROCESS-NAME,Battle.net.exe,Battle
- PROCESS-NAME,Agent.exe,Battle
- PROCESS-NAME,Telegram.exe,Telegram
- GEOSITE,epicgames,Epic
- GEOSITE,blizzard,Battle
- GEOSITE,steam,Steam
- GEOSITE,spotify,Spotify
- GEOSITE,kaspersky,Kaspersky
- GEOSITE,huggingface,Huggingface
- GEOSITE,openai,OpenAI
- GEOSITE,anthropic,Anthropic
- GEOSITE,twitter,X
- GEOSITE,x,X
- GEOSITE,telegram,Telegram
- GEOSITE,youtube,YouTube
- GEOSITE,tiktok,TikTok
- GEOSITE,netflix,Netflix
- GEOSITE,google,Google
- GEOSITE,github,GitHub
- GEOSITE,microsoft,Microsoft
- GEOSITE,onedrive,OneDrive
- GEOSITE,apple,Apple
- GEOSITE,pixiv,Pixiv
- GEOSITE,bilibili,Bilibili
- GEOSITE,biliintl,Bilibili
- GEOSITE,paypal,PayPal
- GEOSITE,category-scholar-!cn,Proxies
- GEOSITE,geolocation-!cn,Proxies
- GEOSITE,cn,DIRECT
- DOMAIN-SUFFIX,local,DIRECT
- GEOIP,telegram,Telegram,no-resolve
- GEOIP,google,Google,no-resolve
- GEOIP,netflix,Netflix,no-resolve
- IP-CIDR,172.110.32.0/21,YouTube,no-resolve
- IP-CIDR,216.73.80.0/20,YouTube,no-resolve
- IP-CIDR6,2620:120:e000::/40,YouTube,no-resolve
- IP-CIDR,106.75.74.76/32,Bilibili,no-resolve
- IP-CIDR,111.206.25.147/32,Bilibili,no-resolve
- IP-CIDR,119.3.238.64/32,Bilibili,no-resolve
- IP-CIDR,120.92.108.182/32,Bilibili,no-resolve
- IP-CIDR,120.92.113.99/32,Bilibili,no-resolve
- IP-CIDR,120.92.153.217/32,Bilibili,no-resolve
- IP-CIDR,134.175.207.130/32,Bilibili,no-resolve
- IP-CIDR,203.107.1.0/24,Bilibili,no-resolve
- IP-CIDR,116.211.202.206/32,Bilibili,no-resolve
- IP-CIDR,116.211.202.216/32,Bilibili,no-resolve
- IP-CIDR,104.154.127.126/32,Spotify,no-resolve
- IP-CIDR,35.186.224.47/32,Spotify,no-resolve
- IP-CIDR,24.199.123.28/32,OpenAI,no-resolve
- IP-CIDR,64.23.132.171/32,OpenAI,no-resolve
- IP-CIDR,17.0.0.0/8,Apple,no-resolve
- IP-CIDR,63.92.224.0/19,Apple,no-resolve
- IP-CIDR,65.199.22.0/23,Apple,no-resolve
- IP-CIDR,139.178.128.0/18,Apple,no-resolve
- IP-CIDR,144.178.0.0/19,Apple,no-resolve
- IP-CIDR,144.178.36.0/22,Apple,no-resolve
- IP-CIDR,144.178.48.0/20,Apple,no-resolve
- IP-CIDR,192.35.50.0/24,Apple,no-resolve
- IP-CIDR,198.183.17.0/24,Apple,no-resolve
- IP-CIDR,205.180.175.0/24,Apple,no-resolve
- IP-CIDR6,2403:300::/32,Apple,no-resolve
- IP-CIDR6,2620:149::/32,Apple,no-resolve
- IP-CIDR6,2a01:b740::/32,Apple,no-resolve
- IP-CIDR,109.239.140.0/24,Telegram,no-resolve
- IP-CIDR,139.59.210.98/32,Telegram,no-resolve
- IP-CIDR,196.55.216.167/32,Telegram,no-resolve
- IP-CIDR,5.28.192.0/18,Telegram,no-resolve
- IP-CIDR6,2001:b28:f23c::/47,Telegram,no-resolve
- IP-CIDR6,2001:b28:f23d::/48,Telegram,no-resolve
- IP-CIDR6,2001:b28:f23f::/48,Telegram,no-resolve
- IP-CIDR6,2001:67c:4e8::/48,Telegram,no-resolve
- IP-CIDR6,2a0a:f280::/29,Telegram,no-resolve
- IP-CIDR,91.108.56.0/22,Telegram,no-resolve
- IP-CIDR,91.108.4.0/22,Telegram,no-resolve
- IP-CIDR,91.108.8.0/22,Telegram,no-resolve
- IP-CIDR,91.108.16.0/22,Telegram,no-resolve
- IP-CIDR,91.108.12.0/22,Telegram,no-resolve
- IP-CIDR,149.154.160.0/20,Telegram,no-resolve
- IP-CIDR,91.105.192.0/23,Telegram,no-resolve
- IP-CIDR,91.108.20.0/22,Telegram,no-resolve
- IP-CIDR,91.108.0.0/16,Telegram,no-resolve
- IP-CIDR,185.76.151.0/24,Telegram,no-resolve
- IP-CIDR6,2001:b28:f23c::/48,Telegram,no-resolve
- IP-CIDR6,2a0a:f280::/32,Telegram,no-resolve
- IP-CIDR,13.32.0.0/15,Proxies,no-resolve
- IP-CIDR,13.35.0.0/17,Proxies,no-resolve
- IP-CIDR,18.184.0.0/15,Proxies,no-resolve
- IP-CIDR,18.194.0.0/15,Proxies,no-resolve
- IP-CIDR,18.208.0.0/13,Proxies,no-resolve
- IP-CIDR,18.232.0.0/14,Proxies,no-resolve
- IP-CIDR,52.200.0.0/13,Proxies,no-resolve
- IP-CIDR,52.58.0.0/15,Proxies,no-resolve
- IP-CIDR,52.74.0.0/16,Proxies,no-resolve
- IP-CIDR,52.77.0.0/16,Proxies,no-resolve
- IP-CIDR,52.84.0.0/15,Proxies,no-resolve
- IP-CIDR,54.156.0.0/14,Proxies,no-resolve
- IP-CIDR,54.226.0.0/15,Proxies,no-resolve
- IP-CIDR,54.230.156.0/22,Proxies,no-resolve
- IP-CIDR,54.93.0.0/16,Proxies,no-resolve
- IP-CIDR,103.2.30.0/23,Proxies,no-resolve
- IP-CIDR,125.209.208.0/20,Proxies,no-resolve
- IP-CIDR,147.92.128.0/17,Proxies,no-resolve
- IP-CIDR,203.104.144.0/21,Proxies,no-resolve
- GEOIP,CN,DIRECT,no-resolve
- GEOIP,CN,DIRECT
- MATCH,Final
`
