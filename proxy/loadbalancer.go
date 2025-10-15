package proxy

import (
	"net/url"
	"sync"
)

// LoadBalancer 简单的负载均衡器
type LoadBalancer struct {
	targets      []*url.URL
	currentIndex int
	mutex        sync.Mutex
}

// NewLoadBalancer 创建新的负载均衡器
func NewLoadBalancer(targetURLs []string) (*LoadBalancer, error) {
	if len(targetURLs) == 0 {
		return nil, nil // 如果没有URL，返回nil（单目标模式）
	}
	
	// 解析所有URL
	targets := make([]*url.URL, 0, len(targetURLs))
	for _, urlStr := range targetURLs {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		targets = append(targets, parsedURL)
	}
	
	return &LoadBalancer{
		targets:      targets,
		currentIndex: 0,
	}, nil
}

// GetNext 获取下一个目标服务器（轮询）
func (lb *LoadBalancer) GetNext() *url.URL {
	if lb == nil || len(lb.targets) == 0 {
		return nil
	}
	
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	// 获取当前索引的目标
	target := lb.targets[lb.currentIndex]
	
	// 更新索引，轮询到下一个
	lb.currentIndex = (lb.currentIndex + 1) % len(lb.targets)
	
	return target
}

// GetTargetCount 获取目标服务器数量
func (lb *LoadBalancer) GetTargetCount() int {
	if lb == nil {
		return 0
	}
	return len(lb.targets)
}