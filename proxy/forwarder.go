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
	
	"github.com/fatih/color"
)

// 颜色定义
var (
	greenF   = color.New(color.FgGreen).SprintFunc()
	redF     = color.New(color.FgRed).SprintFunc()
	yellowF  = color.New(color.FgYellow).SprintFunc()
	blueF    = color.New(color.FgBlue).SprintFunc()
	cyanF    = color.New(color.FgCyan).SprintFunc()
	magentaF = color.New(color.FgMagenta).SprintFunc()
	
	boldGreenF  = color.New(color.FgGreen, color.Bold).SprintFunc()
	boldCyanF   = color.New(color.FgCyan, color.Bold).SprintFunc()
	boldYellowF = color.New(color.FgYellow, color.Bold).SprintFunc()
)

// Forwarder HTTP请求转发器
type Forwarder struct {
	targetURL      *url.URL           // 单目标模式使用
	loadBalancer   *LoadBalancer      // 多目标模式使用
	httpClient     *http.Client
	config         *config.Config
	isMultiTarget  bool
}

// NewForwarder 创建新的转发器
func NewForwarder(cfg *config.Config) (*Forwarder, error) {
	// 获取目标URL
	targetURLs := cfg.Target.GetTargetURLs()
	if len(targetURLs) == 0 {
		return nil, fmt.Errorf("没有配置目标服务器URL")
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
	
	forwarder := &Forwarder{
		httpClient: httpClient,
		config:     cfg,
	}
	
	// 判断是单目标还是多目标模式
	if len(targetURLs) > 1 {
		// 多目标模式：创建负载均衡器
		lb, err := NewLoadBalancer(targetURLs)
		if err != nil {
			return nil, fmt.Errorf("创建负载均衡器失败: %v", err)
		}
		forwarder.loadBalancer = lb
		forwarder.isMultiTarget = true
		logger.Infof("转发器初始化: 多目标负载均衡模式，服务器数量 = %d，策略 = %s",
			len(targetURLs), cfg.Target.Strategy)
		for i, urlStr := range targetURLs {
			logger.Infof("  目标服务器[%d]: %s", i+1, urlStr)
		}
	} else {
		// 单目标模式：直接使用单个URL
		targetURLStr := targetURLs[0]
		if targetURLStr == "" {
			return nil, fmt.Errorf("目标服务器URL为空")
		}
		
		targetURL, err := url.Parse(targetURLStr)
		if err != nil {
			return nil, fmt.Errorf("解析目标URL失败: %v", err)
		}
		
		if targetURL.Scheme == "" {
			return nil, fmt.Errorf("目标URL缺少协议(http/https): %s", targetURLStr)
		}
		
		if targetURL.Host == "" {
			return nil, fmt.Errorf("目标URL缺少主机地址: %s", targetURLStr)
		}
		
		forwarder.targetURL = targetURL
		forwarder.isMultiTarget = false
		logger.Infof("转发器初始化: %s，目标URL = %s", boldGreenF("单目标模式"), blueF(targetURL.String()))
	}

	return forwarder, nil
}

// ForwardRequest 转发HTTP请求
func (f *Forwarder) ForwardRequest(req *http.Request, modifiedBody []byte) (*http.Response, error) {
	startTime := time.Now()
	
	// 获取目标URL（支持负载均衡）
	var baseURL *url.URL
	if f.isMultiTarget {
		// 多目标模式：从负载均衡器获取下一个目标
		baseURL = f.loadBalancer.GetNext()
		logger.Debugf("%s 选择目标服务器: %s", boldYellowF("[负载均衡]"), cyanF(baseURL.String()))
	} else {
		// 单目标模式：使用固定目标
		baseURL = f.targetURL
	}
	
	// 构建完整的目标URL
	targetURL := f.buildTargetURL(baseURL, req.URL.Path, req.URL.RawQuery)
	
	logger.Debugf("转发请求到: %s %s", boldCyanF(req.Method), blueF(targetURL.String()))
	
	// 创建新请求
	targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL.String(), bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, fmt.Errorf("创建目标请求失败: %v", err)
	}

	// 复制请求头
	f.copyHeaders(req.Header, targetReq.Header)
	
	// 移除可能引起问题的头
	f.removeProblematicHeaders(targetReq.Header)
	
	logger.Debugf("转发请求头数量: %s", cyanF(fmt.Sprintf("%d", len(targetReq.Header))))
	
	// 发送请求
	resp, err := f.httpClient.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("转发请求失败: %v", err)
	}
	
	duration := time.Since(startTime)
	var coloredStatus string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		coloredStatus = greenF(fmt.Sprintf("%d", resp.StatusCode))
	} else if resp.StatusCode >= 400 {
		coloredStatus = redF(fmt.Sprintf("%d", resp.StatusCode))
	} else {
		coloredStatus = yellowF(fmt.Sprintf("%d", resp.StatusCode))
	}
	logger.Debugf("转发请求完成，状态码: %s，耗时: %s", coloredStatus, blueF(duration.String()))
	
	return resp, nil
}

// buildTargetURL 构建目标URL
func (f *Forwarder) buildTargetURL(baseURL *url.URL, path, query string) *url.URL {
	// 确保路径以/开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// 构建完整URL
	targetURL := *baseURL // 复制基础URL
	targetURL.Path = path
	targetURL.RawQuery = query
	
	return &targetURL
}

// copyHeaders 复制请求头
func (f *Forwarder) copyHeaders(src, dst http.Header) {
	for key, values := range src {
		// 跳过一些不应该转发的头
		if f.shouldSkipHeader(key) {
			logger.Debugf("跳过转发头: %s", yellowF(key))
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
	// 复制响应头
	f.copyResponseHeaders(resp.Header, w.Header())
	
	// 复制状态码
	w.WriteHeader(resp.StatusCode)
	
	// 复制响应体
	if resp.Body != nil {
		defer resp.Body.Close()
		
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			return fmt.Errorf("复制响应体失败: %v", err)
		}
	}
	
	var coloredStatus string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		coloredStatus = greenF(fmt.Sprintf("%d", resp.StatusCode))
	} else if resp.StatusCode >= 400 {
		coloredStatus = redF(fmt.Sprintf("%d", resp.StatusCode))
	} else {
		coloredStatus = yellowF(fmt.Sprintf("%d", resp.StatusCode))
	}
	logger.Debugf("响应复制完成，状态码: %s", coloredStatus)
	return nil
}

// copyResponseHeaders 复制响应头
func (f *Forwarder) copyResponseHeaders(src, dst http.Header) {
	for key, values := range src {
		// 跳过一些不应该转发的响应头
		if f.shouldSkipResponseHeader(key) {
			logger.Debugf("跳过转发响应头: %s", yellowF(key))
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

// GetTargetURL 获取目标URL（单目标模式）或第一个目标（多目标模式）
func (f *Forwarder) GetTargetURL() *url.URL {
	if f.isMultiTarget && f.loadBalancer != nil {
		// 多目标模式：返回第一个目标
		return f.loadBalancer.targets[0]
	}
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
	var targetURL *url.URL
	
	if f.isMultiTarget && f.loadBalancer != nil {
		// 多目标模式：检查第一个目标
		targetURL = f.loadBalancer.targets[0]
	} else {
		targetURL = f.targetURL
	}
	
	if targetURL == nil {
		return false
	}
	
	healthURL := *targetURL
	healthURL.Path = "/health"
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL.String(), nil)
	if err != nil {
		logger.Debugf("创建健康检查请求失败: %s", redF(err.Error()))
		return false
	}
	
	resp, err := f.httpClient.Do(req)
	if err != nil {
		logger.Debugf("健康检查请求失败: %s", redF(err.Error()))
		return false
	}
	defer resp.Body.Close()
	
	isHealthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	var statusText, statusColor string
	if isHealthy {
		statusText = "健康"
		statusColor = greenF(statusText)
	} else {
		statusText = "不健康"
		statusColor = redF(statusText)
	}
	var coloredStatusCode string
	if isHealthy {
		coloredStatusCode = greenF(fmt.Sprintf("%d", resp.StatusCode))
	} else {
		coloredStatusCode = redF(fmt.Sprintf("%d", resp.StatusCode))
	}
	logger.Debugf("健康检查结果: %s (状态码: %s)", statusColor, coloredStatusCode)
	
	return isHealthy
}

// GetStats 获取转发器统计信息
func (f *Forwarder) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"timeout":         f.httpClient.Timeout.String(),
		"is_multi_target": f.isMultiTarget,
	}
	
	if f.isMultiTarget && f.loadBalancer != nil {
		stats["mode"] = "load_balancing"
		stats["target_count"] = f.loadBalancer.GetTargetCount()
		stats["strategy"] = f.config.Target.Strategy
	} else if f.targetURL != nil {
		stats["mode"] = "single_target"
		stats["target_url"] = f.targetURL.String()
	}
	
	return stats
}