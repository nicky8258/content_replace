package main

import (
	"content_replace/config"
	"content_replace/logger"
	"content_replace/proxy"
	"content_replace/watcher"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 解析命令行参数
	var configFile = flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化日志系统（使用配置文件中的debug设置）
	logger.Init(cfg.Debug.Enabled)
	logger.Info("启动HTTP内容替换代理服务器")
	logger.Info("配置加载成功")
	logger.Debugf("服务器配置: %+v", cfg.Server)
	logger.Debugf("目标地址: %s", cfg.Target.BaseURL)
	logger.Debugf("调试模式: %v", cfg.Debug.Enabled)

	// 启动代理服务器
	server := proxy.NewServer(cfg)
	go func() {
		logger.Infof("代理服务器启动在 %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.Start(); err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 如果启用了自动重载，则启动文件监听器
	var fileWatcher *watcher.Watcher
	if cfg.Rules.AutoReload {
		// 收集所有规则文件路径
		rulesPaths := cfg.Rules.Files
		if cfg.Rules.File != "" {
			rulesPaths = append(rulesPaths, cfg.Rules.File)
		}

		if len(rulesPaths) > 0 {
			fileWatcher, err = watcher.NewWatcher(rulesPaths, func(rules []config.Rule) error {
				server.UpdateRules(rules)
				return nil
			})
			if err != nil {
				logger.Error("创建文件监听器失败: %v", err)
			} else {
				fileWatcher.Start()
			}
		}
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 关闭文件监听器
	if fileWatcher != nil {
		fileWatcher.Stop()
	}

	server.Stop()
	logger.Info("服务器已关闭")
}