package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
	
	"github.com/fatih/color"
)

var (
	debugMode = false
	infoLogger  *log.Logger
	debugLogger *log.Logger
	errorLogger *log.Logger
)

// 颜色定义
var (
	// 规则匹配状态颜色
	green        = color.New(color.FgGreen).SprintFunc()     // 绿色：成功/匹配
	red          = color.New(color.FgRed).SprintFunc()       // 红色：错误/未匹配
	yellow       = color.New(color.FgYellow).SprintFunc()    // 黄色：警告
	blue         = color.New(color.FgBlue).SprintFunc()      // 蓝色：信息
	cyan         = color.New(color.FgCyan).SprintFunc()      // 青色：请求信息
	magenta      = color.New(color.FgMagenta).SprintFunc()   // 洋红：特殊标记
	
	// 加粗版本
	boldGreen    = color.New(color.FgGreen, color.Bold).SprintFunc()
	boldRed      = color.New(color.FgRed, color.Bold).SprintFunc()
	boldYellow   = color.New(color.FgYellow, color.Bold).SprintFunc()
	boldBlue     = color.New(color.FgBlue, color.Bold).SprintFunc()
	boldCyan     = color.New(color.FgCyan, color.Bold).SprintFunc()
	
	// 状态码颜色
	successColor = color.New(color.FgGreen, color.Bold).SprintFunc()
	errorColor   = color.New(color.FgRed, color.Bold).SprintFunc()
	warningColor = color.New(color.FgYellow, color.Bold).SprintFunc()
	infoColor    = color.New(color.FgBlue, color.Bold).SprintFunc()
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
	
	Debugf("%s", boldYellow("=== 原始请求 ==="))
	Debugf("方法: %s", boldCyan(method))
	Debugf("路径: %s", cyan(path))
	Debugf("%s", blue("Headers:"))
	for k, v := range headers {
		Debugf("  %s: %v", green(k), yellow(v))
	}
	Debugf("%s", blue("Body内容:"))
	if len(body) > 0 {
		compressedBody := compressJSONContent(body)
		Debugf("%s", magenta(compressedBody))
	} else {
		Debugf("%s", red("(空)"))
	}
	Debugf("================")
}

// LogModifiedRequest 记录修改后的请求内容
func LogModifiedRequest(method, path string, headers map[string][]string, body string) {
	if !debugMode {
		return
	}
	
	Debugf("%s", boldGreen("=== 修改后请求 ==="))
	Debugf("方法: %s", boldCyan(method))
	Debugf("路径: %s", cyan(path))
//	Debugf("%s", blue("Headers:"))
//	for k, v := range headers {
//		Debugf("  %s: %v", green(k), yellow(v))
//	}
	Debugf("%s", blue("Body内容:"))
	if len(body) > 0 {
		compressedBody := compressJSONContent(body)
		Debugf("%s", green(compressedBody))
	} else {
		Debugf("%s", red("(空)"))
	}
	Debugf("==================")
}

// LogRuleMatch 记录规则匹配情况
func LogRuleMatch(ruleName, mode, pattern, action, value string, matched bool) {
	if !debugMode {
		return
	}
	
	var status string
	var coloredRuleName, coloredStatus string
	
	if matched {
		status = "✓ 匹配"
		coloredRuleName = green(ruleName)
		coloredStatus = boldGreen(status)
	} else {
		status = "✗ 未匹配"
		coloredRuleName = red(ruleName)
		coloredStatus = red(status)
	}
	
	// 截断 pattern 和 value，最大显示30个字符
	truncatedPattern := truncateString(pattern, 30)
	truncatedValue := truncateString(value, 30)
	
	coloredMode := blue(mode)
	coloredPattern := cyan(truncatedPattern)
	coloredValue := magenta(truncatedValue)
	
	Debugf("规则匹配: %s | 模式: %s | 匹配内容: %s | 替换值: %s | 状态: %s",
		coloredRuleName, coloredMode, coloredPattern, coloredValue, coloredStatus)
}

// LogRuleApplied 记录规则应用结果
func LogRuleApplied(ruleName, originalContent, modifiedContent string) {
	if !debugMode {
		return
	}
	
	Debugf("规则应用: %s", boldGreen(ruleName))
	
	// 对原始内容片段应用JSON压缩
	compressedOriginal := compressJSONContent(originalContent)
	Debugf("原始内容片段: %s", red(compressedOriginal))
	
	// 对修改后内容片段应用JSON压缩
	//compressedModified := compressJSONContent(modifiedContent)
	//Debugf("修改后内容片段: %s", green(compressedModified))
}

// LogRequestStart 记录请求开始处理
func LogRequestStart(requestID, method, path string) {
	Debugf("开始处理请求 [%s] %s %s", requestID, method, path)
}

// LogRequestEnd 记录请求处理完成
func LogRequestEnd(requestID string, statusCode int, duration time.Duration) {
	var coloredStatusCode string
	
	switch {
	case statusCode >= 200 && statusCode < 300:
		// 2xx 成功状态码 - 绿色
		coloredStatusCode = successColor(fmt.Sprintf("%d", statusCode))
	case statusCode >= 300 && statusCode < 400:
		// 3xx 重定向状态码 - 黄色
		coloredStatusCode = warningColor(fmt.Sprintf("%d", statusCode))
	case statusCode >= 400 && statusCode < 500:
		// 4xx 客户端错误 - 红色
		coloredStatusCode = errorColor(fmt.Sprintf("%d", statusCode))
	case statusCode >= 500:
		// 5xx 服务器错误 - 红色加粗
		coloredStatusCode = errorColor(fmt.Sprintf("%d", statusCode))
	default:
		// 其他状态码 - 蓝色
		coloredStatusCode = infoColor(fmt.Sprintf("%d", statusCode))
	}
	
	coloredRequestID := cyan(requestID)
	coloredDuration := blue(duration.String())
	
	Debugf("请求处理完成 [%s] 状态码: %s 耗时: %s",
		coloredRequestID, coloredStatusCode, coloredDuration)
}

// truncateString 截断字符串到指定长度，超出部分用省略号表示
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	
	if maxLength <= 30 {
		return s[:maxLength] // 如果最大长度小于等于3，直接截断不添加省略号
	}
	
	return s[:maxLength-30] + "..."
}
// compressJSONContent 压缩 JSON 格式化内容
func compressJSONContent(content string) string {
	// 检查是否包含换行符（JSON 格式化的特征）
	if !strings.Contains(content, "\n") {
		return content // 如果没有换行符，可能已经是压缩格式
	}
	
	// 尝试解析为 JSON 来验证格式
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		// 看起来像 JSON，尝试压缩
		var jsonInterface interface{}
		if err := json.Unmarshal([]byte(content), &jsonInterface); err == nil {
			// 如果能成功解析为 JSON，重新序列化为压缩格式
			if compressed, err := json.Marshal(jsonInterface); err == nil {
				return string(compressed)
			}
		}
	}
	
	// 如果不是有效的 JSON，只移除换行符，保留原有内容
	result := strings.ReplaceAll(content, "\n", " ")
	result = strings.ReplaceAll(result, "\r", " ")
	return result
}

// IsDebugEnabled 检查是否启用debug模式
func IsDebugEnabled() bool {
	return debugMode
}