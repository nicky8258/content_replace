package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 主配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Target   TargetConfig   `yaml:"target"`
	Logging  LoggingConfig  `yaml:"logging"`
	Rules    RulesConfig    `yaml:"rules"`
	Debug    DebugConfig    `yaml:"debug"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host" default:"0.0.0.0"`
	Port int    `yaml:"port" default:"8080"`
}

// TargetConfig 目标服务器配置（支持多服务器）
type TargetConfig struct {
	// 向后兼容：单个目标服务器
	BaseURL string `yaml:"base_url"`
	
	// 新增：多个目标服务器URL数组
	URLs []string `yaml:"urls,omitempty"`
	
	// 通用配置
	Timeout time.Duration `yaml:"timeout" default:"30s"`
	
	// 负载均衡配置
	Strategy string `yaml:"strategy" default:"round_robin"`
	
	// 健康检查配置
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled bool          `yaml:"enabled" default:"true"`
	Interval time.Duration `yaml:"interval" default:"30s"`
	Path    string        `yaml:"path" default:"/health"`
	Timeout time.Duration `yaml:"timeout" default:"5s"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level" default:"info"`
	File  string `yaml:"file" default:"logs/proxy.log"`
}

// RulesConfig 规则配置
type RulesConfig struct {
	File       string   `yaml:"file"`         // 主规则文件路径
	Files      []string `yaml:"files,omitempty"` // 多个规则文件路径
	AutoReload bool     `yaml:"auto_reload" default:"true"`
}

// DebugConfig 调试配置
type DebugConfig struct {
	Enabled         bool `yaml:"enabled" default:"false"`
	ShowOriginal    bool `yaml:"show_original" default:"true"`
	ShowModified    bool `yaml:"show_modified" default:"true"`
	ShowRuleMatches bool `yaml:"show_rule_matches" default:"true"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置默认值
	setDefaults(&config)

	// 验证配置
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	// 确保日志目录存在
	logDir := filepath.Dir(config.Logging.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	// 目标服务器配置默认值
	if config.Target.Timeout == 0 {
		config.Target.Timeout = 30 * time.Second
	}
	if config.Target.Strategy == "" {
		config.Target.Strategy = "round_robin"
	}
	
	// 健康检查默认值
	if config.Target.HealthCheck.Interval == 0 {
		config.Target.HealthCheck.Interval = 30 * time.Second
	}
	if config.Target.HealthCheck.Path == "" {
		config.Target.HealthCheck.Path = "/health"
	}
	if config.Target.HealthCheck.Timeout == 0 {
		config.Target.HealthCheck.Timeout = 5 * time.Second
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.File == "" {
		config.Logging.File = "logs/proxy.log"
	}
}

// validate 验证配置
func validate(config *Config) error {
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("服务器端口必须在1-65535之间")
	}

	// 验证目标服务器配置
	if err := validateTargetConfig(&config.Target); err != nil {
		return err
	}

	// 至少需要一个规则文件路径（file或files）
	if config.Rules.File == "" && len(config.Rules.Files) == 0 {
		return fmt.Errorf("规则文件路径不能为空，请设置file或files字段")
	}

	// 检查规则文件是否存在
	if config.Rules.File != "" {
		if _, err := os.Stat(config.Rules.File); os.IsNotExist(err) {
			return fmt.Errorf("规则文件不存在: %s", config.Rules.File)
		}
	}
	
	// 检查多个规则文件是否存在
	if len(config.Rules.Files) > 0 {
		for _, file := range config.Rules.Files {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				return fmt.Errorf("规则文件不存在: %s", file)
			}
		}
	}

	return nil
}

// GetAddress 获取服务器监听地址
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDebugEnabled 检查是否启用调试模式
func (c *Config) IsDebugEnabled() bool {
	return c.Debug.Enabled
}

// ShouldShowOriginal 检查是否显示原始内容
func (c *Config) ShouldShowOriginal() bool {
	return c.Debug.Enabled && c.Debug.ShowOriginal
}

// ShouldShowModified 检查是否显示修改后内容
func (c *Config) ShouldShowModified() bool {
	return c.Debug.Enabled && c.Debug.ShowModified
}

// ShouldShowRuleMatches 检查是否显示规则匹配情况
func (c *Config) ShouldShowRuleMatches() bool {
	return c.Debug.Enabled && c.Debug.ShowRuleMatches
}

// Reload 重新加载配置
func (c *Config) Reload(configPath string) error {
	newConfig, err := Load(configPath)
	if err != nil {
		return err
	}

	*c = *newConfig
	return nil
}
// validateTargetConfig 验证目标服务器配置
func validateTargetConfig(target *TargetConfig) error {
	// 检查是否配置了目标服务器
	if target.BaseURL == "" && len(target.URLs) == 0 {
		return fmt.Errorf("目标服务器地址不能为空，请设置base_url或urls字段")
	}
	
	// 如果同时配置了base_url和urls，优先使用urls
	if target.BaseURL != "" && len(target.URLs) > 0 {
		fmt.Printf("警告: 同时配置了base_url和urls，将优先使用urls数组\n")
	}
	
	// 验证单个URL格式
	if target.BaseURL != "" {
		if _, err := url.Parse(target.BaseURL); err != nil {
			return fmt.Errorf("无效的目标URL: %v", err)
		}
	}
	
	// 验证多个URL格式
	for i, urlStr := range target.URLs {
		if _, err := url.Parse(urlStr); err != nil {
			return fmt.Errorf("无效的目标URL[%d]: %v", i, err)
		}
	}
	
	// 验证负载均衡策略
	validStrategies := map[string]bool{
		"round_robin": true,
		// 为未来扩展预留
		"weighted_round_robin": true,
		"least_connections": true,
	}
	
	if !validStrategies[target.Strategy] {
		return fmt.Errorf("不支持的负载均衡策略: %s", target.Strategy)
	}
	
	return nil
}

// GetTargetURLs 获取所有目标服务器URL
func (t *TargetConfig) GetTargetURLs() []string {
	if len(t.URLs) > 0 {
		return t.URLs
	}
	
	// 向后兼容：如果urls为空，使用base_url
	if t.BaseURL != "" {
		return []string{t.BaseURL}
	}
	
	return []string{}
}

// IsMultiTarget 检查是否配置了多个目标服务器
func (t *TargetConfig) IsMultiTarget() bool {
	return len(t.URLs) > 1
}

// GetStrategy 获取负载均衡策略
func (t *TargetConfig) GetStrategy() string {
	if t.Strategy == "" {
		return "round_robin"
	}
	return t.Strategy
}