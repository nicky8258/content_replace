package replacer

import (
	"content_replace/config"
	"content_replace/logger"
	"context"
	"fmt"
	"sync"
)

// Engine 内容替换引擎
type Engine struct {
	rules      []config.Rule
	rulesPath  string
	rulesPaths []string // 支持多个规则文件路径
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewEngine 创建新的替换引擎
func NewEngine(rulesPath string) *Engine {
	return NewEngineFromPaths([]string{rulesPath})
}

// NewEngineFromPaths 从多个路径创建替换引擎
func NewEngineFromPaths(rulesPaths []string) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		rulesPath:  rulesPaths[0], // 保存主路径用于兼容性
		ctx:        ctx,
		cancel:     cancel,
		rulesPaths: rulesPaths, // 保存所有路径
	}

	// 初始加载规则
	if err := engine.LoadRules(); err != nil {
		logger.Error("加载规则失败: %v", err)
	}

	return engine
}

// LoadRules 加载规则文件
func (e *Engine) LoadRules() error {
	var rules []config.Rule
	var err error

	if len(e.rulesPaths) > 1 {
		// 从多个文件加载规则
		rules, err = config.LoadRulesFromPaths(e.rulesPaths)
	} else {
		// 从单个文件加载规则（兼容原有逻辑）
		rules, err = config.LoadRules(e.rulesPath)
	}

	if err != nil {
		return fmt.Errorf("加载规则失败: %v", err)
	}

	e.UpdateRules(rules)

	logger.Info("成功加载 %d 条替换规则", len(rules))
	for _, rule := range rules {
		logger.Debugf("规则: %s", rule.GetDescription())
	}

	return nil
}

// ReloadRules 重新加载规则
func (e *Engine) ReloadRules() error {
	logger.Info("重新加载替换规则...")
	return e.LoadRules()
}

// UpdateRules 更新规则
func (e *Engine) UpdateRules(rules []config.Rule) {
	e.mutex.Lock()
	e.rules = rules
	e.mutex.Unlock()
	logger.Info("替换引擎已更新 %d 条规则", len(rules))
}

// Process 处理内容替换
func (e *Engine) Process(content string) (string, error) {
	e.mutex.RLock()
	rules := make([]config.Rule, len(e.rules))
	copy(rules, e.rules)
	e.mutex.RUnlock()

	if len(rules) == 0 {
		logger.Debug("没有可用的替换规则")
		return content, nil
	}

	originalContent := content
	modifiedContent := content

	logger.Debugf("开始处理内容替换，共 %d 条规则", len(rules))

	for _, rule := range rules {
		if !rule.IsEnabled() {
			logger.Debugf("规则 %s 已禁用，跳过", rule.Name)
			continue
		}

		// 检查是否匹配
		matched := rule.Match(modifiedContent)
		logger.LogRuleMatch(rule.Name, string(rule.Mode), rule.Pattern, string(rule.Action), rule.Value, matched)

		if matched {
			// 应用规则
			beforeApply := modifiedContent
			modifiedContent = rule.Apply(modifiedContent)

			logger.LogRuleApplied(rule.Name, beforeApply, modifiedContent)

			// 如果内容有变化，记录日志
			if beforeApply != modifiedContent {
				logger.Debugf("规则 %s 应用成功，内容已修改", rule.Name)
			}
		}

		// 检查上下文是否被取消
		select {
		case <-e.ctx.Done():
			return "", fmt.Errorf("替换引擎已停止")
		default:
		}
	}

	// 如果内容有变化，记录最终结果
	if originalContent != modifiedContent {
		logger.Debugf("内容替换完成，原始内容长度: %d，修改后长度: %d",
			len(originalContent), len(modifiedContent))
	} else {
		logger.Debugf("内容替换完成，内容未发生改变")
	}

	return modifiedContent, nil
}

// GetRules 获取当前规则列表
func (e *Engine) GetRules() []config.Rule {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	rules := make([]config.Rule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// GetEnabledRules 获取启用的规则列表
func (e *Engine) GetEnabledRules() []config.Rule {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var enabledRules []config.Rule
	for _, rule := range e.rules {
		if rule.IsEnabled() {
			enabledRules = append(enabledRules, rule)
		}
	}
	return enabledRules
}

// GetRuleByName 根据名称获取规则
func (e *Engine) GetRuleByName(name string) (*config.Rule, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	for _, rule := range e.rules {
		if rule.Name == name {
			return &rule, true
		}
	}
	return nil, false
}

// EnableRule 启用规则
func (e *Engine) EnableRule(name string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i := range e.rules {
		if e.rules[i].Name == name {
			e.rules[i].SetEnabled(true)
			logger.Info("规则 %s 已启用", name)
			return nil
		}
	}
	return fmt.Errorf("规则 %s 不存在", name)
}

// DisableRule 禁用规则
func (e *Engine) DisableRule(name string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i := range e.rules {
		if e.rules[i].Name == name {
			e.rules[i].SetEnabled(false)
			logger.Info("规则 %s 已禁用", name)
			return nil
		}
	}
	return fmt.Errorf("规则 %s 不存在", name)
}

// GetStats 获取引擎统计信息
func (e *Engine) GetStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_rules":    len(e.rules),
		"enabled_rules":  0,
		"disabled_rules": 0,
	}

	for _, rule := range e.rules {
		if rule.IsEnabled() {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		} else {
			stats["disabled_rules"] = stats["disabled_rules"].(int) + 1
		}
	}

	return stats
}

// Stop 停止替换引擎
func (e *Engine) Stop() {
	e.cancel()
	logger.Info("替换引擎已停止")
}

// ValidateContent 验证内容
func (e *Engine) ValidateContent(content string) error {
	if content == "" {
		return fmt.Errorf("内容不能为空")
	}
	return nil
}