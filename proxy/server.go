package proxy

import (
	"content_replace/config"
	"content_replace/logger"
	"content_replace/replacer"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Server HTTP代理服务器
type Server struct {
	config    *config.Config
	server    *http.Server
	engine    *replacer.Engine
	forwarder *Forwarder
	handler   *Handler
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewServer 创建新的代理服务器
func NewServer(cfg *config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建替换引擎
	var engine *replacer.Engine
	if len(cfg.Rules.Files) > 0 {
		// 使用多个规则文件
		engine = replacer.NewEngineFromPaths(cfg.Rules.Files)
	} else {
		// 使用单个规则文件（兼容原有逻辑）
		engine = replacer.NewEngine(cfg.Rules.File)
	}
	
	// 创建转发器
	forwarder, err := NewForwarder(cfg)
	if err != nil {
		logger.Error("创建转发器失败: %v", err)
		cancel()
		return nil
	}
	
	// 创建处理器
	handler := NewHandler(cfg, engine, forwarder)

	server := &Server{
		config:    cfg,
		engine:    engine,
		forwarder: forwarder,
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
	}

	return server
}

// Start 启动服务器
func (s *Server) Start() error {
	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:         s.config.GetAddress(),
		Handler:      s.handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Infof("启动HTTP代理服务器在 %s", s.config.GetAddress())

	// 启动服务器
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("服务器启动失败: %v", err)
	}

	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	logger.Info("正在停止HTTP代理服务器...")

	// 取消上下文
	s.cancel()

	// 停止替换引擎
	s.engine.Stop()

	// 关闭HTTP服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			logger.Error("服务器关闭失败: %v", err)
			return err
		}
	}

	// 等待所有goroutine完成
	s.wg.Wait()

	logger.Info("HTTP代理服务器已停止")
	return nil
}

// UpdateRules 更新规则
func (s *Server) UpdateRules(rules []config.Rule) {
	s.engine.UpdateRules(rules)
}

// GetEngine 获取替换引擎
func (s *Server) GetEngine() *replacer.Engine {
	return s.engine
}

// GetForwarder 获取转发器
func (s *Server) GetForwarder() *Forwarder {
	return s.forwarder
}

// GetConfig 获取配置
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// IsRunning 检查服务器是否正在运行
func (s *Server) IsRunning() bool {
	return s.server != nil
}

// ReloadConfig 重新加载配置
func (s *Server) ReloadConfig() error {
	logger.Info("重新加载配置...")
	
	// 重新加载主配置
	if err := s.config.Reload("configs/config.yaml"); err != nil {
		return fmt.Errorf("重新加载配置失败: %v", err)
	}
	
	// 重新加载规则
	if err := s.engine.ReloadRules(); err != nil {
		return fmt.Errorf("重新加载规则失败: %v", err)
	}
	
	logger.Info("配置重新加载成功")
	return nil
}

// GetStats 获取服务器统计信息
func (s *Server) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"server": map[string]interface{}{
			"address":    s.config.GetAddress(),
			"running":    s.IsRunning(),
			"debug_mode": s.config.Debug.Enabled,
		},
		"engine":    s.engine.GetStats(),
		"forwarder": s.forwarder.GetStats(),
	}
	
	return stats
}

// HealthCheck 健康检查
func (s *Server) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"server":    "running",
	}
	
	// 检查目标服务器健康状态
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	targetHealthy := s.forwarder.IsHealthy(ctx)
	health["target_server"] = map[string]interface{}{
		"status": map[bool]string{true: "healthy", false: "unhealthy"}[targetHealthy],
		"url":    s.forwarder.GetTargetURL().String(),
	}
	
	// 如果目标服务器不健康，整体状态为不健康
	if !targetHealthy {
		health["status"] = "degraded"
	}
	
	// 检查引擎状态
	engineStats := s.engine.GetStats()
	if engineStats["enabled_rules"].(int) == 0 {
		health["engine"] = "no_rules_enabled"
	} else {
		health["engine"] = "healthy"
	}
	
	return health
}

// SetLogLevel 设置日志级别
func (s *Server) SetLogLevel(level string) error {
	logger.Infof("设置日志级别为: %s", level)
	// 这里可以扩展日志级别的动态设置
	return nil
}

// EnableDebugMode 启用调试模式
func (s *Server) EnableDebugMode() {
	s.config.Debug.Enabled = true
	logger.Info("调试模式已启用")
}

// DisableDebugMode 禁用调试模式
func (s *Server) DisableDebugMode() {
	s.config.Debug.Enabled = false
	logger.Info("调试模式已禁用")
}

// GetRequestInfo 获取请求信息
func (s *Server) GetRequestInfo() map[string]interface{} {
	return map[string]interface{}{
		"total_rules":       len(s.engine.GetRules()),
		"enabled_rules":     len(s.engine.GetEnabledRules()),
		"target_url":        s.forwarder.GetTargetURL().String(),
		"server_address":    s.config.GetAddress(),
		"auto_reload_rules": s.config.Rules.AutoReload,
		"debug_mode":        s.config.Debug.Enabled,
	}
}