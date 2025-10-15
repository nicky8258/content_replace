package watcher

import (
	"content_replace/config"
	"content_replace/logger"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher 文件监听器
type Watcher struct {
	watcher   *fsnotify.Watcher
	rules     []config.Rule
	rulesPath []string
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	callback  func([]config.Rule) error
}

// NewWatcher 创建新的文件监听器
func NewWatcher(rulesPath []string, callback func([]config.Rule) error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监听器失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		watcher:  watcher,
		ctx:      ctx,
		cancel:   cancel,
		callback: callback,
	}

	// 转换并存储绝对路径
	absPaths := make([]string, 0, len(rulesPath))
	for _, path := range rulesPath {
		abs, err := filepath.Abs(path)
		if err != nil {
			logger.Error("获取文件绝对路径失败 %s: %v", path, err)
			continue
		}
		absPaths = append(absPaths, abs)
	}
	w.rulesPath = absPaths

	// 初始加载规则
	if err := w.loadRules(); err != nil {
		logger.Error("初始加载规则失败: %v", err)
	}

	// 添加监听路径（目录）
	dirs := make(map[string]bool)
	for _, path := range w.rulesPath {
		dir := filepath.Dir(path)
		if !dirs[dir] {
			if err := w.watcher.Add(dir); err != nil {
				logger.Error("添加监听路径失败 %s: %v", dir, err)
			} else {
				logger.Debug("添加监听路径: %s", dir)
				dirs[dir] = true
			}
		}
	}

	return w, nil
}

// Start 开始监听文件变化
func (w *Watcher) Start() {
	logger.Info("开始监听规则文件变化")

	go w.watchLoop()
}

// Stop 停止监听
func (w *Watcher) Stop() {
	logger.Info("停止文件监听")
	w.cancel()
	w.watcher.Close()
}

// watchLoop 监听循环
func (w *Watcher) watchLoop() {
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C // 立即停止定时器
	}

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			w.handleEvent(event, debounceTimer)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Error("文件监听错误: %v", err)

		case <-w.ctx.Done():
			return
		}
	}
}

// handleEvent 处理文件事件
func (w *Watcher) handleEvent(event fsnotify.Event, debounceTimer *time.Timer) {
	// 检查是否是我们监听的规则文件
	if !w.isWatchedFile(event.Name) {
		return
	}

	logger.Debugf("文件事件: %s %s", event.Op.String(), event.Name)

	// 只处理写入和创建事件
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		// 重置防抖定时器
		debounceTimer.Stop()
		debounceTimer.Reset(500 * time.Millisecond) // 500ms防抖

		go func() {
			<-debounceTimer.C
			w.reloadRules()
		}()
	}
}

// isWatchedFile 检查是否是我们监听的文件
func (w *Watcher) isWatchedFile(filename string) bool {
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		logger.Error("获取事件文件绝对路径失败 %s: %v", filename, err)
		return false
	}

	for _, path := range w.rulesPath {
		if absFilename == path {
			return true
		}
	}
	return false
}

// reloadRules 重新加载规则
func (w *Watcher) reloadRules() {
	logger.Info("检测到规则文件变化，重新加载规则...")

	if err := w.loadRules(); err != nil {
		logger.Error("重新加载规则失败: %v", err)
		return
	}

	// 调用回调函数
	if w.callback != nil {
		if err := w.callback(w.getRules()); err != nil {
			logger.Error("规则重载回调失败: %v", err)
		} else {
			logger.Info("规则重载成功")
		}
	}
}

// loadRules 加载规则
func (w *Watcher) loadRules() error {
	// 注意：这里我们使用绝对路径来加载规则
	allRules, err := config.LoadRulesFromPaths(w.rulesPath)
	if err != nil {
		return fmt.Errorf("加载规则失败: %v", err)
	}

	w.mutex.Lock()
	w.rules = allRules
	w.mutex.Unlock()

	logger.Info("成功加载 %d 条规则", len(allRules))
	return nil
}

// GetRules 获取当前规则
func (w *Watcher) GetRules() []config.Rule {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	rules := make([]config.Rule, len(w.rules))
	copy(rules, w.rules)
	return rules
}

// getRules 内部获取规则（不加锁）
func (w *Watcher) getRules() []config.Rule {
	return w.rules
}

// GetStats 获取监听器统计信息
func (w *Watcher) GetStats() map[string]interface{} {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return map[string]interface{}{
		"watching_files": w.rulesPath,
		"rules_count":    len(w.rules),
		"watcher_active": true,
	}
}