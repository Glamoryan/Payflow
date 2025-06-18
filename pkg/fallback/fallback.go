package fallback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"payflow/pkg/circuitbreaker"
	"payflow/pkg/logger"
)

type Strategy int

const (
	StrategyCache Strategy = iota
	StrategyDefault
	StrategyRetry
	StrategyCircuitBreaker
	StrategyQueue
	StrategyDegraded
)

type FallbackManager struct {
	strategies map[string]*FallbackConfig
	cache      FallbackCache
	logger     logger.Logger
	retryQueue *RetryQueue
	mutex      sync.RWMutex
}

type FallbackConfig struct {
	Name           string
	PrimaryFunc    func(ctx context.Context) (interface{}, error)
	FallbackFunc   func(ctx context.Context, err error) (interface{}, error)
	Strategy       Strategy
	MaxRetries     int
	RetryInterval  time.Duration
	Timeout        time.Duration
	CircuitBreaker *circuitbreaker.CircuitBreaker
	CacheKey       string
	CacheTTL       time.Duration
	DefaultValue   interface{}
	DegradedMode   bool
}

type FallbackCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
}

type SimpleCache struct {
	data  map[string]*cacheItem
	mutex sync.RWMutex
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

type RetryQueue struct {
	items   chan *RetryItem
	workers int
	logger  logger.Logger
}

type RetryItem struct {
	ID         string
	Function   func() error
	MaxRetries int
	Interval   time.Duration
	Attempt    int
}

func NewFallbackManager(logger logger.Logger) *FallbackManager {
	return &FallbackManager{
		strategies: make(map[string]*FallbackConfig),
		cache:      NewSimpleCache(),
		logger:     logger,
		retryQueue: NewRetryQueue(5, logger),
	}
}

func (fm *FallbackManager) RegisterFallback(config *FallbackConfig) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	fm.strategies[config.Name] = config
	fm.logger.InfoContext(context.Background(), "Fallback registered", map[string]interface{}{
		"name":     config.Name,
		"strategy": config.Strategy,
	})
}

func (fm *FallbackManager) Execute(ctx context.Context, name string) (interface{}, error) {
	fm.mutex.RLock()
	config, exists := fm.strategies[name]
	fm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("fallback configuration not found: %s", name)
	}

	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	result, err := fm.executePrimary(ctx, config)
	if err == nil {
		if config.Strategy == StrategyCache && config.CacheKey != "" {
			fm.cache.Set(config.CacheKey, result, config.CacheTTL)
		}
		return result, nil
	}

	return fm.executeFallback(ctx, config, err)
}

func (fm *FallbackManager) executePrimary(ctx context.Context, config *FallbackConfig) (interface{}, error) {
	if config.CircuitBreaker != nil {
		return config.CircuitBreaker.Execute(func() (interface{}, error) {
			return config.PrimaryFunc(ctx)
		})
	}

	return config.PrimaryFunc(ctx)
}

func (fm *FallbackManager) executeFallback(ctx context.Context, config *FallbackConfig, primaryErr error) (interface{}, error) {
	switch config.Strategy {
	case StrategyCache:
		return fm.fallbackCache(config, primaryErr)
	case StrategyDefault:
		return fm.fallbackDefault(config, primaryErr)
	case StrategyRetry:
		return fm.fallbackRetry(ctx, config, primaryErr)
	case StrategyDegraded:
		return fm.fallbackDegraded(ctx, config, primaryErr)
	default:
		if config.FallbackFunc != nil {
			return config.FallbackFunc(ctx, primaryErr)
		}
		return nil, primaryErr
	}
}

func (fm *FallbackManager) fallbackCache(config *FallbackConfig, primaryErr error) (interface{}, error) {
	if config.CacheKey == "" {
		return nil, fmt.Errorf("cache key not specified for cache fallback")
	}

	if value, found := fm.cache.Get(config.CacheKey); found {
		fm.logger.InfoContext(context.Background(), "Fallback cache hit", map[string]interface{}{
			"name":      config.Name,
			"cache_key": config.CacheKey,
		})
		return value, nil
	}

	if config.DefaultValue != nil {
		fm.logger.InfoContext(context.Background(), "Fallback to default value", map[string]interface{}{
			"name": config.Name,
		})
		return config.DefaultValue, nil
	}

	return nil, fmt.Errorf("cache fallback failed, no cached value or default: %w", primaryErr)
}

func (fm *FallbackManager) fallbackDefault(config *FallbackConfig, primaryErr error) (interface{}, error) {
	if config.DefaultValue != nil {
		fm.logger.InfoContext(context.Background(), "Fallback to default value", map[string]interface{}{
			"name": config.Name,
		})
		return config.DefaultValue, nil
	}

	return nil, fmt.Errorf("no default value specified: %w", primaryErr)
}

func (fm *FallbackManager) fallbackRetry(ctx context.Context, config *FallbackConfig, primaryErr error) (interface{}, error) {
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	interval := config.RetryInterval
	if interval <= 0 {
		interval = time.Second
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			result, err := fm.executePrimary(ctx, config)
			if err == nil {
				fm.logger.InfoContext(context.Background(), "Retry successful", map[string]interface{}{
					"name":    config.Name,
					"attempt": attempt,
				})
				return result, nil
			}

			fm.logger.Error("Retry attempt failed", map[string]interface{}{
				"name":    config.Name,
				"attempt": attempt,
				"error":   err.Error(),
			})

			interval *= 2
		}
	}

	if config.FallbackFunc != nil {
		return config.FallbackFunc(ctx, primaryErr)
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", primaryErr)
}

func (fm *FallbackManager) fallbackDegraded(ctx context.Context, config *FallbackConfig, primaryErr error) (interface{}, error) {
	if config.FallbackFunc != nil {
		fm.logger.InfoContext(context.Background(), "Executing degraded mode", map[string]interface{}{
			"name": config.Name,
		})
		return config.FallbackFunc(ctx, primaryErr)
	}

	return nil, fmt.Errorf("no degraded mode function specified: %w", primaryErr)
}

func (fm *FallbackManager) QueueRetry(item *RetryItem) {
	fm.retryQueue.Add(item)
}

func (fm *FallbackManager) GetStats() map[string]interface{} {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	stats := map[string]interface{}{
		"registered_fallbacks": len(fm.strategies),
		"retry_queue_size":     len(fm.retryQueue.items),
	}

	fallbackStats := make(map[string]interface{})
	for name, config := range fm.strategies {
		stat := map[string]interface{}{
			"strategy": config.Strategy,
		}
		if config.CircuitBreaker != nil {
			stat["circuit_breaker_state"] = config.CircuitBreaker.State().String()
			stat["circuit_breaker_counts"] = config.CircuitBreaker.Counts()
		}
		fallbackStats[name] = stat
	}
	stats["fallbacks"] = fallbackStats

	return stats
}

func NewSimpleCache() *SimpleCache {
	cache := &SimpleCache{
		data: make(map[string]*cacheItem),
	}

	go cache.cleanup()

	return cache
}

func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		return nil, false
	}

	return item.value, true
}

func (c *SimpleCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *SimpleCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

func (c *SimpleCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expiresAt) {
				delete(c.data, key)
			}
		}
		c.mutex.Unlock()
	}
}

func NewRetryQueue(workers int, logger logger.Logger) *RetryQueue {
	rq := &RetryQueue{
		items:   make(chan *RetryItem, 1000),
		workers: workers,
		logger:  logger,
	}

	for i := 0; i < workers; i++ {
		go rq.worker()
	}

	return rq
}

func (rq *RetryQueue) Add(item *RetryItem) {
	select {
	case rq.items <- item:
	default:
		rq.logger.Error("Retry queue is full, dropping item", map[string]interface{}{
			"item_id": item.ID,
		})
	}
}

func (rq *RetryQueue) worker() {
	for item := range rq.items {
		rq.processRetryItem(item)
	}
}

func (rq *RetryQueue) processRetryItem(item *RetryItem) {
	item.Attempt++

	err := item.Function()
	if err == nil {
		rq.logger.InfoContext(context.Background(), "Retry operation successful", map[string]interface{}{
			"item_id": item.ID,
			"attempt": item.Attempt,
		})
		return
	}

	if item.Attempt < item.MaxRetries {
		go func() {
			time.Sleep(item.Interval)
			rq.Add(item)
		}()

		rq.logger.Error("Retry operation failed, scheduling retry", map[string]interface{}{
			"item_id": item.ID,
			"attempt": item.Attempt,
			"error":   err.Error(),
		})
	} else {
		rq.logger.Error("Retry operation failed permanently", map[string]interface{}{
			"item_id":     item.ID,
			"attempt":     item.Attempt,
			"max_retries": item.MaxRetries,
			"error":       err.Error(),
		})
	}
}
