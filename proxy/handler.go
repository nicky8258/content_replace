package proxy

import (
	"content_replace/config"
	"content_replace/logger"
	"content_replace/replacer"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Handler HTTP请求处理器
type Handler struct {
	config   *config.Config
	engine   *replacer.Engine
	forwarder *Forwarder
}

// NewHandler 创建新的处理器
func NewHandler(cfg *config.Config, engine *replacer.Engine, forwarder *Forwarder) *Handler {
	return &Handler{
		config:    cfg,
		engine:    engine,
		forwarder: forwarder,
	}
}

// ServeHTTP 处理HTTP请求
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	requestID := h.generateRequestID()
	
	logger.LogRequestStart(requestID, req.Method, req.URL.Path)
	
	// 记录原始请求
	if h.config.ShouldShowOriginal() {
		headers := make(map[string][]string)
		for k, v := range req.Header {
			headers[k] = v
		}
		
		body, err := h.readRequestBody(req)
		if err != nil {
			logger.Error("读取请求体失败: %v", err)
			http.Error(w, "读取请求体失败", http.StatusInternalServerError)
			return
		}
		
		logger.LogOriginalRequest(req.Method, req.URL.Path, headers, body)
		
		// 重新设置请求体
		req.Body = io.NopCloser(strings.NewReader(body))
	}
	
	// 读取请求体
	body, err := h.readRequestBody(req)
	if err != nil {
		logger.Error("读取请求体失败: %v", err)
		http.Error(w, "读取请求体失败", http.StatusInternalServerError)
		return
	}
	
	// 如果有内容，进行替换处理
	var modifiedBody []byte
	if len(body) > 0 {
		modifiedBodyStr, err := h.engine.Process(body)
		if err != nil {
			logger.Error("内容替换失败: %v", err)
			http.Error(w, "内容替换失败", http.StatusInternalServerError)
			return
		}
		modifiedBody = []byte(modifiedBodyStr)
		
		// 记录修改后的内容
		if h.config.ShouldShowModified() {
			headers := make(map[string][]string)
			for k, v := range req.Header {
				headers[k] = v
			}
			logger.LogModifiedRequest(req.Method, req.URL.Path, headers, modifiedBodyStr)
		}
	} else {
		modifiedBody = []byte(body)
	}
	
	// 转发请求
	resp, err := h.forwarder.ForwardRequest(req, modifiedBody)
	if err != nil {
		logger.Error("转发失败: %v", err)
		http.Error(w, "转发失败", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	// 复制响应
	if err := h.forwarder.CopyResponse(w, resp); err != nil {
		logger.Error("复制响应失败: %v", err)
		return
	}
	
	duration := time.Since(startTime)
	logger.LogRequestEnd(requestID, resp.StatusCode, duration)
}

// readRequestBody 读取请求体
func (h *Handler) readRequestBody(req *http.Request) (string, error) {
	if req.Body == nil {
		return "", nil
	}
	
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", fmt.Errorf("读取请求体失败: %v", err)
	}
	
	// 重新设置请求体，以便后续可以再次读取
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	
	return string(body), nil
}

// generateRequestID 生成请求ID
func (h *Handler) generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// logRequestDetails 记录请求详情
func (h *Handler) logRequestDetails(req *http.Request, body string) {
	logger.Debugf("=== 请求详情 ===")
	logger.Debugf("请求ID: %s", h.generateRequestID())
	logger.Debugf("方法: %s", req.Method)
	logger.Debugf("URL: %s", req.URL.String())
	logger.Debugf("协议: %s", req.Proto)
	logger.Debugf("主机: %s", req.Host)
	logger.Debugf("远程地址: %s", req.RemoteAddr)
	
	logger.Debugf("请求头:")
	for name, values := range req.Header {
		for _, value := range values {
			logger.Debugf("  %s: %s", name, value)
		}
	}
	
	if len(body) > 0 {
		logger.Debugf("请求体长度: %d", len(body))
		if logger.IsDebugEnabled() {
			// 限制调试模式下显示的请求体长度
			maxDisplayLength := 1000
			if len(body) > maxDisplayLength {
				logger.Debugf("请求体内容 (前%d字符): %s...", maxDisplayLength, body[:maxDisplayLength])
			} else {
				logger.Debugf("请求体内容: %s", body)
			}
		}
	}
	
	logger.Debugf("================")
}

// logResponseDetails 记录响应详情
func (h *Handler) logResponseDetails(resp *http.Response) {
	logger.Debugf("=== 响应详情 ===")
	logger.Debugf("状态码: %d", resp.StatusCode)
	logger.Debugf("协议: %s", resp.Proto)
	
	logger.Debugf("响应头:")
	for name, values := range resp.Header {
		for _, value := range values {
			logger.Debugf("  %s: %s", name, value)
		}
	}
	
	logger.Debugf("================")
}

// shouldProcessBody 检查是否应该处理请求体
func (h *Handler) shouldProcessBody(req *http.Request) bool {
	// 只对有请求体的方法进行处理
	contentLength := req.ContentLength
	if contentLength <= 0 {
		return false
	}
	
	// 检查内容类型
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		// 如果没有指定内容类型，但有内容长度，尝试处理
		return contentLength > 0
	}
	
	// 只处理文本类型的内容
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/x-www-form-urlencoded",
		"application/javascript",
	}
	
	for _, textType := range textTypes {
		if strings.HasPrefix(strings.ToLower(contentType), textType) {
			return true
		}
	}
	
	// 对于其他类型，如果开启了debug模式，也尝试处理
	if logger.IsDebugEnabled() {
		logger.Debugf("内容类型 %s 可能不是文本类型，但debug模式已开启，尝试处理", contentType)
		return true
	}
	
	return false
}

// getContentType 获取内容类型
func (h *Handler) getContentType(req *http.Request) string {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

// isMultipartRequest 检查是否为multipart请求
func (h *Handler) isMultipartRequest(req *http.Request) bool {
	contentType := req.Header.Get("Content-Type")
	return strings.Contains(strings.ToLower(contentType), "multipart/form-data")
}

// shouldSkipProcessing 检查是否应该跳过处理
func (h *Handler) shouldSkipProcessing(req *http.Request) bool {
	// 跳过健康检查请求
	if req.URL.Path == "/health" {
		return true
	}
	
	// 跳过静态资源请求
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf"}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(req.URL.Path), ext) {
			return true
		}
	}
	
	return false
}