package logger

import (
	"io"
	"log"
	"os"
	"time"
)

var (
	debugMode = false
	infoLogger  *log.Logger
	debugLogger *log.Logger
	errorLogger *log.Logger
)

// Init 初始化日志系统
func Init(debug bool) {
	debugMode = debug
	
	// 创建日志目录
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("无法创建日志目录: %v", err)
	}
	
	// 初始化不同级别的日志器
	infoLogger = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	
	// 如果不是debug模式，禁用debug日志输出
	if !debugMode {
		debugLogger = log.New(io.Discard, "[DEBUG] ", log.LstdFlags)
	}
}

// Info 记录信息日志
func Info(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Infof 格式化信息日志
func Infof(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Debug 记录调试日志
func Debug(format string, v ...interface{}) {
	debugLogger.Printf(format, v...)
}

// Debugf 格式化调试日志
func Debugf(format string, v ...interface{}) {
	if debugMode {
		debugLogger.Printf(format, v...)
	}
}

// Error 记录错误日志
func Error(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}

// LogOriginalRequest 记录原始请求内容
func LogOriginalRequest(method, path string, headers map[string][]string, body string) {
	if !debugMode {
		return
	}
	
	Debugf("=== 原始请求 ===")
	Debugf("方法: %s", method)
	Debugf("路径: %s", path)
	Debugf("Headers:")
	for k, v := range headers {
		Debugf("  %s: %v", k, v)
	}
	Debugf("Body内容:")
	Debugf("%s", body)
	Debugf("================")
}

// LogModifiedRequest 记录修改后的请求内容
func LogModifiedRequest(method, path string, headers map[string][]string, body string) {
	if !debugMode {
		return
	}
	
	Debugf("=== 修改后请求 ===")
	Debugf("方法: %s", method)
	Debugf("路径: %s", path)
	Debugf("Body内容:")
	Debugf("%s", body)
	Debugf("==================")
}

// LogRuleMatch 记录规则匹配情况
func LogRuleMatch(ruleName, mode, pattern, action, value string, matched bool) {
	if !debugMode {
		return
	}
	
	status := "未匹配"
	if matched {
		status = "✓ 匹配"
	}
	
	Debugf("规则匹配: %s | 模式: %s | 匹配内容: %s | 动作: %s | 替换值: %s | 状态: %s", 
		ruleName, mode, pattern, action, value, status)
}

// LogRuleApplied 记录规则应用结果
func LogRuleApplied(ruleName, originalContent, modifiedContent string) {
	if !debugMode {
		return
	}
	
	Debugf("规则应用: %s", ruleName)
	Debugf("原始内容片段: %s", originalContent)
	Debugf("修改后内容片段: %s", modifiedContent)
}

// LogRequestStart 记录请求开始处理
func LogRequestStart(requestID, method, path string) {
	Debugf("开始处理请求 [%s] %s %s", requestID, method, path)
}

// LogRequestEnd 记录请求处理完成
func LogRequestEnd(requestID string, statusCode int, duration time.Duration) {
	Debugf("请求处理完成 [%s] 状态码: %d 耗时: %v", requestID, statusCode, duration)
}

// IsDebugEnabled 检查是否启用debug模式
func IsDebugEnabled() bool {
	return debugMode
}