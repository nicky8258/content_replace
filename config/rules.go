package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// RuleMode 匹配模式枚举
type RuleMode string

const (
	ModePrefix   RuleMode = "prefix"   // 前缀匹配
	ModeSuffix   RuleMode = "suffix"   // 后缀匹配
	ModeContains RuleMode = "contains" // 包含匹配
	ModeRegex    RuleMode = "regex"    // 正则匹配
)

// RuleAction 规则动作枚举
type RuleAction string

const (
	ActionReplace         RuleAction = "replace"           // 替换
	ActionDelete          RuleAction = "delete"            // 删除
	ActionDeleteJsonField RuleAction = "delete_json_field" // 删除JSON字段
)

// Rule 替换规则结构
type Rule struct {
	Name    string     `yaml:"name"`     // 规则名称
	Enabled bool       `yaml:"enabled"`  // 是否启用
	Mode    RuleMode   `yaml:"mode"`     // 匹配模式
	Pattern string     `yaml:"pattern"`  // 匹配模式
	Action  RuleAction `yaml:"action"`   // 动作类型
	Value   string     `yaml:"value"`    // 替换值（仅用于替换动作）
}

// RulesConfig 规则配置文件结构
type RulesFile struct {
	Rules []Rule `yaml:"rules"`
	Easy  *EasyRules `yaml:"easy,omitempty"`
}

// EasyRules 简单规则配置
type EasyRules struct {
	Delete  *EasyDeleteRules  `yaml:"delete,omitempty"`
	Replace *EasyReplaceRules `yaml:"replace,omitempty"`
	Prefix  *EasyPrefixRules  `yaml:"prefix,omitempty"`
	Suffix  *EasySuffixRules  `yaml:"suffix,omitempty"`
	Regex   *EasyRegexRules   `yaml:"regex,omitempty"`
}

// EasyDeleteRules 简单删除规则
type EasyDeleteRules struct {
	Contains []string `yaml:"contains,omitempty"`
	Prefix   []string `yaml:"prefix,omitempty"`
	Suffix   []string `yaml:"suffix,omitempty"`
	Regex    []string `yaml:"regex,omitempty"`
}

// EasyReplaceRules 简单替换规则
type EasyReplaceRules struct {
	Contains map[string]string `yaml:"contains,omitempty"`
	Prefix   map[string]string `yaml:"prefix,omitempty"`
	Suffix   map[string]string `yaml:"suffix,omitempty"`
	Regex    map[string]string `yaml:"regex,omitempty"`
}

// EasyPrefixRules 简单前缀规则
type EasyPrefixRules struct {
	Delete  []string          `yaml:"delete,omitempty"`
	Replace map[string]string `yaml:"replace,omitempty"`
}

// EasySuffixRules 简单后缀规则
type EasySuffixRules struct {
	Delete  []string          `yaml:"delete,omitempty"`
	Replace map[string]string `yaml:"replace,omitempty"`
}

// EasyRegexRules 简单正则规则
type EasyRegexRules struct {
	Delete  []string          `yaml:"delete,omitempty"`
	Replace map[string]string `yaml:"replace,omitempty"`
}

// LoadRules 加载规则文件
func LoadRules(rulesPath string) ([]Rule, error) {
	return LoadRulesFromPaths([]string{rulesPath})
}

// LoadRulesFromPaths 从多个路径加载规则文件
func LoadRulesFromPaths(rulesPaths []string) ([]Rule, error) {
	var allRules []Rule
	
	for _, rulesPath := range rulesPaths {
		rules, err := loadSingleRulesFile(rulesPath)
		if err != nil {
			return nil, fmt.Errorf("加载规则文件失败 %s: %v", rulesPath, err)
		}
		allRules = append(allRules, rules...)
	}

	// 验证所有规则
	if err := validateRules(allRules); err != nil {
		return nil, fmt.Errorf("规则验证失败: %v", err)
	}

	return allRules, nil
}

// loadSingleRulesFile 加载单个规则文件
func loadSingleRulesFile(rulesPath string) ([]Rule, error) {
	// 检查规则文件是否存在
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("规则文件不存在: %s", rulesPath)
	}

	// 读取规则文件
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("读取规则文件失败: %v", err)
	}

	// 首先尝试作为简单规则文件解析
	var easyRules EasyRules
	if err := yaml.Unmarshal(data, &easyRules); err == nil {
		// 检查是否包含简单规则结构
		if hasEasyRules(&easyRules) {
			rules := convertEasyRules(&easyRules)
			return rules, nil
		}
	}

	// 如果不是简单规则文件，尝试作为标准规则文件解析
	var rulesFile RulesFile
	if err := yaml.Unmarshal(data, &rulesFile); err != nil {
		return nil, fmt.Errorf("解析规则文件失败: %v", err)
	}

	// 合并规则
	allRules := rulesFile.Rules
	
	// 如果有简单规则，转换为标准规则
	if rulesFile.Easy != nil {
		easyRules := convertEasyRules(rulesFile.Easy)
		allRules = append(allRules, easyRules...)
	}
	
	// 检查是否有外部文件引用的规则
	if rulesFile.Easy != nil {
		// 检查删除规则中的外部文件引用
		if rulesFile.Easy.Delete != nil && len(rulesFile.Easy.Delete.Contains) > 0 {
			for _, path := range rulesFile.Easy.Delete.Contains {
				if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
					// 加载外部简单规则文件
					externalRules, err := LoadEasyRules(path)
					if err != nil {
						return nil, fmt.Errorf("加载外部简单规则文件失败 %s: %v", path, err)
					}
					allRules = append(allRules, externalRules...)
				} else {
					// 作为普通删除规则处理
					allRules = append(allRules, Rule{
						Name:    fmt.Sprintf("删除-%s", path[:min(20, len(path))]),
						Enabled: true,
						Mode:    ModeContains,
						Pattern: path,
						Action:  ActionDelete,
					})
				}
			}
		}
	}

	return allRules, nil
}

// hasEasyRules 检查是否包含简单规则
func hasEasyRules(easyRules *EasyRules) bool {
	if easyRules == nil {
		return false
	}
	
	// 检查删除规则
	if easyRules.Delete != nil {
		if len(easyRules.Delete.Contains) > 0 ||
		   len(easyRules.Delete.Prefix) > 0 ||
		   len(easyRules.Delete.Suffix) > 0 ||
		   len(easyRules.Delete.Regex) > 0 {
			return true
		}
	}
	
	// 检查替换规则
	if easyRules.Replace != nil {
		if len(easyRules.Replace.Contains) > 0 ||
		   len(easyRules.Replace.Prefix) > 0 ||
		   len(easyRules.Replace.Suffix) > 0 ||
		   len(easyRules.Replace.Regex) > 0 {
			return true
		}
	}
	
	// 检查前缀规则
	if easyRules.Prefix != nil {
		if len(easyRules.Prefix.Delete) > 0 ||
		   len(easyRules.Prefix.Replace) > 0 {
			return true
		}
	}
	
	// 检查后缀规则
	if easyRules.Suffix != nil {
		if len(easyRules.Suffix.Delete) > 0 ||
		   len(easyRules.Suffix.Replace) > 0 {
			return true
		}
	}
	
	// 检查正则规则
	if easyRules.Regex != nil {
		if len(easyRules.Regex.Delete) > 0 ||
		   len(easyRules.Regex.Replace) > 0 {
			return true
		}
	}
	
	return false
}

// validateRules 验证规则
func validateRules(rules []Rule) error {
	for i, rule := range rules {
		if err := validateRule(&rule, i); err != nil {
			return err
		}
	}
	return nil
}

// validateRule 验证单个规则
func validateRule(rule *Rule, index int) error {
	if rule.Name == "" {
		return fmt.Errorf("规则 #%d: 规则名称不能为空", index+1)
	}

	if rule.Mode == "" {
		return fmt.Errorf("规则 #%d (%s): 匹配模式不能为空", index+1, rule.Name)
	}

	// 验证匹配模式
	validModes := map[RuleMode]bool{
		ModePrefix:   true,
		ModeSuffix:   true,
		ModeContains: true,
		ModeRegex:    true,
	}
	if !validModes[rule.Mode] {
		return fmt.Errorf("规则 #%d (%s): 无效的匹配模式 '%s'，支持的模式: prefix, suffix, contains, regex", 
			index+1, rule.Name, rule.Mode)
	}

	if rule.Pattern == "" {
		return fmt.Errorf("规则 #%d (%s): 匹配内容不能为空", index+1, rule.Name)
	}

	if rule.Action == "" {
		return fmt.Errorf("规则 #%d (%s): 动作类型不能为空", index+1, rule.Name)
	}

	// 验证动作类型
	validActions := map[RuleAction]bool{
		ActionReplace:         true,
		ActionDelete:          true,
		ActionDeleteJsonField: true,
	}
	if !validActions[rule.Action] {
		return fmt.Errorf("规则 #%d (%s): 无效的动作类型 '%s'，支持的动作: replace, delete, delete_json_field", 
			index+1, rule.Name, rule.Action)
	}

	// 如果是替换动作，替换值不能为空
	if rule.Action == ActionReplace && rule.Value == "" {
		return fmt.Errorf("规则 #%d (%s): 替换动作的替换值不能为空", index+1, rule.Name)
	}

	// 如果是正则模式，验证正则表达式
	if rule.Mode == ModeRegex {
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("规则 #%d (%s): 正则表达式无效: %v", index+1, rule.Name, err)
		}
	}

	return nil
}

// Match 检查规则是否匹配内容
func (r *Rule) Match(content string) bool {
	if !r.Enabled {
		return false
	}

	return r.matchString(content)
}

// Apply 应用规则到内容
func (r *Rule) Apply(content string) string {
	if r.Action != ActionDeleteJsonField && !r.Match(content) {
		return content
	}

	switch r.Action {
	case ActionDelete:
		return r.deleteContent(content)
	case ActionReplace:
		return r.replaceContent(content)
	case ActionDeleteJsonField:
		return r.deleteJsonField(content)
	default:
		return content
	}
}

// deleteContent 删除匹配的内容
func (r *Rule) deleteContent(content string) string {
	switch r.Mode {
	case ModePrefix:
		if strings.HasPrefix(content, r.Pattern) {
			return content[len(r.Pattern):]
		}
	case ModeSuffix:
		if strings.HasSuffix(content, r.Pattern) {
			return content[:len(content)-len(r.Pattern)]
		}
	case ModeContains:
		return strings.ReplaceAll(content, r.Pattern, "")
	case ModeRegex:
		re := regexp.MustCompile(r.Pattern)
		return re.ReplaceAllString(content, "")
	}
	return content
}

// replaceContent 替换匹配的内容
func (r *Rule) replaceContent(content string) string {
	switch r.Mode {
	case ModePrefix:
		if strings.HasPrefix(content, r.Pattern) {
			return r.Value + content[len(r.Pattern):]
		}
	case ModeSuffix:
		if strings.HasSuffix(content, r.Pattern) {
			return content[:len(content)-len(r.Pattern)] + r.Value
		}
	case ModeContains:
		return strings.ReplaceAll(content, r.Pattern, r.Value)
	case ModeRegex:
		re := regexp.MustCompile(r.Pattern)
		return re.ReplaceAllString(content, r.Value)
	}
	return content
}

// deleteJsonField 删除包含匹配内容的整个JSON对象
func (r *Rule) deleteJsonField(content string) string {
	var data interface{}
	if !(strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")) && !(strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]")) {
		return content
	}

	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return content
	}

	modifiedData, _ := r.recursiveDelete(data)

	modifiedBytes, err := json.MarshalIndent(modifiedData, "", "  ")
	if err != nil {
		return content
	}

	return string(modifiedBytes)
}

// recursiveDelete 递归遍历并删除节点
func (r *Rule) recursiveDelete(node interface{}) (interface{}, bool) {
	switch n := node.(type) {
	case map[string]interface{}:
		for _, v := range n {
			if s, ok := v.(string); ok && r.matchString(s) {
				return nil, true
			}
		}

		modifiedMap := make(map[string]interface{})
		for k, v := range n {
			modifiedValue, deleteChild := r.recursiveDelete(v)
			if !deleteChild {
				modifiedMap[k] = modifiedValue
			}
		}
		return modifiedMap, false

	case []interface{}:
		var modifiedSlice []interface{}
		for _, item := range n {
			modifiedItem, deleteItem := r.recursiveDelete(item)
			if !deleteItem {
				modifiedSlice = append(modifiedSlice, modifiedItem)
			}
		}
		return modifiedSlice, false

	default:
		return n, false
	}
}

// matchString 根据规则模式检查字符串
func (r *Rule) matchString(s string) bool {
	switch r.Mode {
	case ModeContains:
		return strings.Contains(s, r.Pattern)
	case ModePrefix:
		return strings.HasPrefix(s, r.Pattern)
	case ModeSuffix:
		return strings.HasSuffix(s, r.Pattern)
	case ModeRegex:
		matched, _ := regexp.MatchString(r.Pattern, s)
		return matched
	default:
		return false
	}
}


// GetDescription 获取规则描述
func (r *Rule) GetDescription() string {
	var actionDesc string
	switch r.Action {
	case ActionDelete:
		actionDesc = "删除"
	case ActionDeleteJsonField:
		actionDesc = "删除JSON字段"
	case ActionReplace:
		actionDesc = fmt.Sprintf("替换为 '%s'", r.Value)
	}
	
	return fmt.Sprintf("%s: %s匹配 '%s' -> %s", 
		r.Name, r.Mode, r.Pattern, actionDesc)
}

// IsEnabled 检查规则是否启用
func (r *Rule) IsEnabled() bool {
	return r.Enabled
}

// SetEnabled 设置规则启用状态
func (r *Rule) SetEnabled(enabled bool) {
	r.Enabled = enabled
}

// convertEasyRules 将简单规则转换为标准规则
func convertEasyRules(easy *EasyRules) []Rule {
	var rules []Rule
	
	if easy.Delete != nil {
		rules = append(rules, convertDeleteRules(easy.Delete)...)
	}
	
	if easy.Replace != nil {
		rules = append(rules, convertReplaceRules(easy.Replace)...)
	}
	
	if easy.Prefix != nil {
		rules = append(rules, convertPrefixRules(easy.Prefix)...)
	}
	
	if easy.Suffix != nil {
		rules = append(rules, convertSuffixRules(easy.Suffix)...)
	}
	
	if easy.Regex != nil {
		rules = append(rules, convertRegexRules(easy.Regex)...)
	}
	
	return rules
}

// convertDeleteRules 转换删除规则
func convertDeleteRules(delete *EasyDeleteRules) []Rule {
	var rules []Rule
	
	// 包含删除
	for i, pattern := range delete.Contains {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量删除-contains-%d", i+1),
			Enabled: true,
			Mode:    ModeContains,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 前缀删除
	for i, pattern := range delete.Prefix {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量删除-prefix-%d", i+1),
			Enabled: true,
			Mode:    ModePrefix,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 后缀删除
	for i, pattern := range delete.Suffix {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量删除-suffix-%d", i+1),
			Enabled: true,
			Mode:    ModeSuffix,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 正则删除
	for i, pattern := range delete.Regex {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量删除-regex-%d", i+1),
			Enabled: true,
			Mode:    ModeRegex,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	return rules
}

// convertReplaceRules 转换替换规则
func convertReplaceRules(replace *EasyReplaceRules) []Rule {
	var rules []Rule
	
	// 包含替换
	for pattern, value := range replace.Contains {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量替换-contains-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModeContains,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	// 前缀替换
	for pattern, value := range replace.Prefix {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量替换-prefix-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModePrefix,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	// 后缀替换
	for pattern, value := range replace.Suffix {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量替换-suffix-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModeSuffix,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	// 正则替换
	for pattern, value := range replace.Regex {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("批量替换-regex-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModeRegex,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	return rules
}

// convertPrefixRules 转换前缀规则
func convertPrefixRules(prefix *EasyPrefixRules) []Rule {
	var rules []Rule
	
	// 前缀删除
	for i, pattern := range prefix.Delete {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("前缀删除-%d", i+1),
			Enabled: true,
			Mode:    ModePrefix,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 前缀替换
	for pattern, value := range prefix.Replace {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("前缀替换-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModePrefix,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	return rules
}

// convertSuffixRules 转换后缀规则
func convertSuffixRules(suffix *EasySuffixRules) []Rule {
	var rules []Rule
	
	// 后缀删除
	for i, pattern := range suffix.Delete {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("后缀删除-%d", i+1),
			Enabled: true,
			Mode:    ModeSuffix,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 后缀替换
	for pattern, value := range suffix.Replace {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("后缀替换-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModeSuffix,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	return rules
}

// convertRegexRules 转换正则规则
func convertRegexRules(regex *EasyRegexRules) []Rule {
	var rules []Rule
	
	// 正则删除
	for i, pattern := range regex.Delete {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("正则删除-%d", i+1),
			Enabled: true,
			Mode:    ModeRegex,
			Pattern: pattern,
			Action:  ActionDelete,
		})
	}
	
	// 正则替换
	for pattern, value := range regex.Replace {
		rules = append(rules, Rule{
			Name:    fmt.Sprintf("正则替换-%s", pattern[:min(10, len(pattern))]),
			Enabled: true,
			Mode:    ModeRegex,
			Pattern: pattern,
			Action:  ActionReplace,
			Value:   value,
		})
	}
	
	return rules
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LoadEasyRules 加载简单规则文件
func LoadEasyRules(easyRulesPath string) ([]Rule, error) {
	// 检查规则文件是否存在
	if _, err := os.Stat(easyRulesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("简单规则文件不存在: %s", easyRulesPath)
	}

	// 读取规则文件
	data, err := os.ReadFile(easyRulesPath)
	if err != nil {
		return nil, fmt.Errorf("读取简单规则文件失败: %v", err)
	}

	// 解析简单规则
	var easyRules EasyRules
	if err := yaml.Unmarshal(data, &easyRules); err != nil {
		return nil, fmt.Errorf("解析简单规则文件失败: %v", err)
	}

	// 转换为标准规则
	rules := convertEasyRules(&easyRules)
	
	// 验证规则
	if err := validateRules(rules); err != nil {
		return nil, fmt.Errorf("简单规则验证失败: %v", err)
	}

	return rules, nil
}
