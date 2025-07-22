package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
)

// LifecycleManager 生命周期管理器
type LifecycleManager struct {
	logger   kratoslog.Logger
	hooks    []Hook
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once
}

// Hook 生命周期钩子
type Hook struct {
	Name     string                      // 钩子名称
	OnStart  func(context.Context) error // 启动时执行的函数
	OnStop   func(context.Context) error // 停止时执行的函数
	Priority int                         // 优先级，数字越小优先级越高
	// Priority分级:
	// 0-99:    基础设施层（数据库、Redis、Kafka连接）
	// 100-199: 服务器层（HTTP、gRPC、WebSocket服务器）
	// 200-299: 客户端层（gRPC客户端、外部连接）
	// 300+:    业务逻辑层
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(logger kratoslog.Logger) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &LifecycleManager{
		logger: logger,
		hooks:  make([]Hook, 0),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// AddHook 添加生命周期钩子
func (lm *LifecycleManager) AddHook(hook Hook) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.hooks = append(lm.hooks, hook)
	lm.sortHooks()
}

// sortHooks 按优先级排序钩子
func (lm *LifecycleManager) sortHooks() {
	// 冒泡，临时，按优先级从小到大排序
	for i := 0; i < len(lm.hooks)-1; i++ {
		for j := 0; j < len(lm.hooks)-1-i; j++ {
			if lm.hooks[j].Priority > lm.hooks[j+1].Priority {
				lm.hooks[j], lm.hooks[j+1] = lm.hooks[j+1], lm.hooks[j]
			}
		}
	}
}

// Start 启动所有钩子
func (lm *LifecycleManager) Start() error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lm.logger.Log(kratoslog.LevelInfo, "msg", "Starting lifecycle hooks")

	for _, hook := range lm.hooks {
		if hook.OnStart != nil {
			lm.logger.Log(kratoslog.LevelInfo, "msg", "Starting hook", "name", hook.Name)

			if err := hook.OnStart(lm.ctx); err != nil {
				lm.logger.Log(kratoslog.LevelError, "msg", "Hook start failed", "name", hook.Name, "error", err)
				return err
			}

			lm.logger.Log(kratoslog.LevelInfo, "msg", "Hook started successfully", "name", hook.Name)
		}
	}

	lm.logger.Log(kratoslog.LevelInfo, "msg", "All lifecycle hooks started")
	return nil
}

// Stop 停止所有钩子
func (lm *LifecycleManager) Stop() error {
	var stopErr error

	lm.stopOnce.Do(func() {
		lm.mu.RLock()
		defer lm.mu.RUnlock()

		lm.logger.Log(kratoslog.LevelInfo, "msg", "Stopping lifecycle hooks")

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 反向停止钩子（后启动的先停止）
		for i := len(lm.hooks) - 1; i >= 0; i-- {
			hook := lm.hooks[i]
			if hook.OnStop != nil {
				lm.logger.Log(kratoslog.LevelInfo, "msg", "Stopping hook", "name", hook.Name)

				if err := hook.OnStop(ctx); err != nil {
					lm.logger.Log(kratoslog.LevelError, "msg", "Hook stop failed", "name", hook.Name, "error", err)
					if stopErr == nil {
						stopErr = err
					}
				} else {
					lm.logger.Log(kratoslog.LevelInfo, "msg", "Hook stopped successfully", "name", hook.Name)
				}
			}
		}

		// 取消上下文
		lm.cancel()
		close(lm.done)

		lm.logger.Log(kratoslog.LevelInfo, "msg", "All lifecycle hooks stopped")
	})

	return stopErr
}

// Wait 等待停止信号
func (lm *LifecycleManager) Wait() {
	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	case sig := <-sigChan:
		lm.logger.Log(kratoslog.LevelInfo, "msg", "Received signal", "signal", sig.String())
		lm.Stop()
	case <-lm.done:
		// 已经停止
	}
}

// Context 获取生命周期上下文
func (lm *LifecycleManager) Context() context.Context {
	return lm.ctx
}

// Done 获取完成通道
func (lm *LifecycleManager) Done() <-chan struct{} {
	return lm.done
}

// IsRunning 检查是否正在运行
func (lm *LifecycleManager) IsRunning() bool {
	select {
	case <-lm.done:
		return false
	default:
		return true
	}
}
