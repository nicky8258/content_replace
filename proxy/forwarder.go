package proxy

import (
	"bytes"
	"content_replace/config"
	"content_replace/logger"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Forwarder HTTP请求转发器
type Forwarder struct {
	targetURL   *url.URL
	httpClient  *http.Client
	config      *config.Config
}

// NewForwarder 创建新的转发器
func NewForwarder(cfg *config.Config) (*Forwarder, error) {
	// 解析目标URL
	targetURL, err := url.Parse(cfg.Target.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("解析目标URL失败: %v", err)
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: cfg.Target.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &Forwarder{
		targetURL:  targetURL,
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// ForwardRequest 转发HTTP请求
func (f *Forwarder) ForwardRequest(req *http.Request, modifiedBody []byte) (*http.Response, error) {
	startTime := time.Now()
	
	// 构建目标URL
	targetURL := f.buildTargetURL(req.URL.Path, req.URL.RawQuery)
	
	logger.Debugf("转发请求到: %s %s", req.Method, targetURL.String())
	
	// 创建新请求
	targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL.String(), bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, fmt.Errorf("创建目标请求失败: %v", err)
	}

	// 复制请求头
	f.copyHeaders(req.Header, targetReq.Header)
	
	// 移除可能引起问题的头
	f.removeProblematicHeaders(targetReq.Header)
	
	logger.Debugf("转发请求头数量: %d", len(targetReq.Header))
	
	// 发送请求
	resp, err := f.httpClient.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("转发请求失败: %v", err)
	}
	
	duration := time.Since(startTime)
	logger.Debugf("转发请求完成，状态码: %d，耗时: %v", resp.StatusCode, duration)
	
	return resp, nil
}

// buildTargetURL 构建目标URL
func (f *Forwarder) buildTargetURL(path, query string) *url.URL {
	// 确保路径以/开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// 构建完整URL
	targetURL := *f.targetURL // 复制目标URL
	targetURL.Path = path
	targetURL.RawQuery = query
	
	return &targetURL
}

// copyHeaders 复制请求头
func (f *Forwarder) copyHeaders(src, dst http.Header) {
	for key, values := range src {
		// 跳过一些不应该转发的头
		if f.shouldSkipHeader(key) {
			logger.Debugf("跳过转发头: %s", key)
			continue
		}
		
		// 复制头值
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// shouldSkipHeader 检查是否应该跳过某个头
func (f *Forwarder) shouldSkipHeader(key string) bool {
	skipHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":           true,
		"Transfer-Encoding":  true,
		"Upgrade":            true,
		"Proxy-Connection":   true,
	}
	
	// 转换为小写进行比较
	lowerKey := strings.ToLower(key)
	return skipHeaders[lowerKey] || strings.HasPrefix(lowerKey, "proxy-")
}

// removeProblematicHeaders 移除可能有问题的头
func (f *Forwarder) removeProblematicHeaders(headers http.Header) {
	problematicHeaders := []string{
		"Content-Length",
	}
	
	for _, header := range problematicHeaders {
		headers.Del(header)
	}
}

// CopyResponse 复制响应
func (f *Forwarder) CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	// 复制状态码
	w.WriteHeader(resp.StatusCode)
	
	// 复制响应头
	f.copyResponseHeaders(resp.Header, w.Header())
	
	// 复制响应体
	if resp.Body != nil {
		defer resp.Body.Close()
		
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			return fmt.Errorf("复制响应体失败: %v", err)
		}
	}
	
	logger.Debugf("响应复制完成，状态码: %d", resp.StatusCode)
	return nil
}

// copyResponseHeaders 复制响应头
func (f *Forwarder) copyResponseHeaders(src, dst http.Header) {
	for key, values := range src {
		// 跳过一些不应该转发的响应头
		if f.shouldSkipResponseHeader(key) {
			logger.Debugf("跳过转发响应头: %s", key)
			continue
		}
		
		// 复制头值
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// shouldSkipResponseHeader 检查是否应该跳过某个响应头
func (f *Forwarder) shouldSkipResponseHeader(key string) bool {
	skipHeaders := map[string]bool{
		"Connection":        true,
		"Keep-Alive":        true,
		"Proxy-Authenticate": true,
		"Proxy-Authorization": true,
		"Te":               true,
		"Trailers":         true,
		"Transfer-Encoding": true,
		"Upgrade":          true,
	}
	
	// 转换为小写进行比较
	lowerKey := strings.ToLower(key)
	return skipHeaders[lowerKey] || strings.HasPrefix(lowerKey, "proxy-")
}

// GetTargetURL 获取目标URL
func (f *Forwarder) GetTargetURL() *url.URL {
	return f.targetURL
}

// GetClient 获取HTTP客户端
func (f *Forwarder) GetClient() *http.Client {
	return f.httpClient
}

// SetTimeout 设置超时时间
func (f *Forwarder) SetTimeout(timeout time.Duration) {
	f.httpClient.Timeout = timeout
}

// IsHealthy 检查目标服务器健康状态
func (f *Forwarder) IsHealthy(ctx context.Context) bool {
	healthURL := *f.targetURL
	healthURL.Path = "/health"
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL.String(), nil)
	if err != nil {
		logger.Debugf("创建健康检查请求失败: %v", err)
		return false
	}
	
	resp, err := f.httpClient.Do(req)
	if err != nil {
		logger.Debugf("健康检查请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()
	
	isHealthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	logger.Debugf("健康检查结果: %s (状态码: %d)", map[bool]string{true: "健康", false: "不健康"}[isHealthy], resp.StatusCode)
	
	return isHealthy
}

// GetStats 获取转发器统计信息
func (f *Forwarder) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"target_url": f.targetURL.String(),
		"timeout":    f.httpClient.Timeout.String(),
	}
}