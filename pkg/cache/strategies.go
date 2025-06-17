package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"payflow/pkg/logger"
	"time"
)

// Cache key constants
const (
	// User cache keys
	UserPrefix        = "user"
	UserByIDKey       = "user:id:%d"
	UserByUsernameKey = "user:username:%s"
	UserByEmailKey    = "user:email:%s"

	// Balance cache keys
	BalancePrefix     = "balance"
	BalanceByUserKey  = "balance:user:%d"
	BalanceHistoryKey = "balance:history:user:%d"

	// Transaction cache keys
	TransactionPrefix    = "transaction"
	TransactionByIDKey   = "transaction:id:%d"
	TransactionByUserKey = "transaction:user:%d"
	TransactionStatsKey  = "transaction:stats:user:%d"

	// Event cache keys
	EventPrefix         = "event"
	EventByAggregateKey = "event:aggregate:%s:%s"
	EventByTypeKey      = "event:type:%s"

	// Aggregated data cache keys
	DashboardStatsKey     = "dashboard:stats"
	TopUsersKey           = "dashboard:top_users"
	RecentTransactionsKey = "dashboard:recent_transactions"
)

// Cache expiration times
const (
	ShortExpiration    = 5 * time.Minute  // Frequently changing data
	MediumExpiration   = 30 * time.Minute // Moderately changing data
	LongExpiration     = 2 * time.Hour    // Rarely changing data
	VeryLongExpiration = 24 * time.Hour   // Static or rarely updated data
)

// CacheStrategy defines different caching patterns
type CacheStrategy interface {
	// Read-through: Check cache first, if miss then fetch from source and cache it
	ReadThrough(ctx context.Context, key string, dest interface{}, fetchFunc func() (interface{}, error), expiration time.Duration) error

	// Write-through: Write to cache and source simultaneously
	WriteThrough(ctx context.Context, key string, value interface{}, writeFunc func(value interface{}) error, expiration time.Duration) error

	// Write-behind: Write to cache immediately, write to source asynchronously
	WriteBehind(ctx context.Context, key string, value interface{}, writeFunc func(value interface{}) error, expiration time.Duration) error

	// Cache-aside: Manual cache management
	CacheAside(ctx context.Context, key string, dest interface{}, fetchFunc func() (interface{}, error), expiration time.Duration) error
}

// CacheManager implements various caching strategies
type CacheManager struct {
	cache  Cache
	logger logger.Logger
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cache Cache, logger logger.Logger) CacheStrategy {
	return &CacheManager{
		cache:  cache,
		logger: logger,
	}
}

// ReadThrough implements read-through caching pattern
func (cm *CacheManager) ReadThrough(ctx context.Context, key string, dest interface{}, fetchFunc func() (interface{}, error), expiration time.Duration) error {
	// Try to get from cache first
	err := cm.cache.Get(ctx, key, dest)
	if err == nil {
		// Cache hit
		cm.logger.Debug("Cache hit for read-through", map[string]interface{}{"key": key})
		return nil
	}

	if err != ErrCacheMiss {
		// Real error, not just cache miss
		cm.logger.Error("Cache error in read-through", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		// Continue to fetch from source despite cache error
	}

	// Cache miss or error, fetch from source
	cm.logger.Debug("Cache miss, fetching from source", map[string]interface{}{"key": key})
	data, err := fetchFunc()
	if err != nil {
		cm.logger.Error("Source fetch error in read-through", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	// Store in cache for next time
	if err := cm.cache.Set(ctx, key, data, expiration); err != nil {
		cm.logger.Error("Cache set error in read-through", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		// Don't fail the request if cache set fails
	}

	// Copy data to destination
	return copyData(data, dest)
}

// WriteThrough implements write-through caching pattern
func (cm *CacheManager) WriteThrough(ctx context.Context, key string, value interface{}, writeFunc func(value interface{}) error, expiration time.Duration) error {
	// Write to source first
	err := writeFunc(value)
	if err != nil {
		cm.logger.Error("Source write error in write-through", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	// Write to cache
	if err := cm.cache.Set(ctx, key, value, expiration); err != nil {
		cm.logger.Error("Cache set error in write-through", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		// Don't fail the request if cache set fails, source is already updated
	}

	cm.logger.Debug("Write-through completed", map[string]interface{}{"key": key})
	return nil
}

// WriteBehind implements write-behind caching pattern
func (cm *CacheManager) WriteBehind(ctx context.Context, key string, value interface{}, writeFunc func(value interface{}) error, expiration time.Duration) error {
	// Write to cache immediately
	if err := cm.cache.Set(ctx, key, value, expiration); err != nil {
		cm.logger.Error("Cache set error in write-behind", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	// Write to source asynchronously
	go func() {
		if err := writeFunc(value); err != nil {
			cm.logger.Error("Async source write error in write-behind", map[string]interface{}{
				"key":   key,
				"error": err.Error(),
			})
			// In production, you might want to retry or queue for later
		} else {
			cm.logger.Debug("Async write-behind completed", map[string]interface{}{"key": key})
		}
	}()

	return nil
}

// CacheAside implements cache-aside pattern
func (cm *CacheManager) CacheAside(ctx context.Context, key string, dest interface{}, fetchFunc func() (interface{}, error), expiration time.Duration) error {
	// Try to get from cache
	err := cm.cache.Get(ctx, key, dest)
	if err == nil {
		cm.logger.Debug("Cache hit for cache-aside", map[string]interface{}{"key": key})
		return nil
	}

	if err != ErrCacheMiss {
		cm.logger.Error("Cache error in cache-aside", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
	}

	// Fetch from source
	data, err := fetchFunc()
	if err != nil {
		return err
	}

	// Manually cache the result
	if err := cm.cache.Set(ctx, key, data, expiration); err != nil {
		cm.logger.Error("Cache set error in cache-aside", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
	}

	return copyData(data, dest)
}

// Helper functions for cache key generation
func UserCacheKey(userID int64) string {
	return fmt.Sprintf(UserByIDKey, userID)
}

func UserCacheKeyByUsername(username string) string {
	return fmt.Sprintf(UserByUsernameKey, username)
}

func UserCacheKeyByEmail(email string) string {
	return fmt.Sprintf(UserByEmailKey, email)
}

func BalanceCacheKey(userID int64) string {
	return fmt.Sprintf(BalanceByUserKey, userID)
}

func BalanceHistoryCacheKey(userID int64) string {
	return fmt.Sprintf(BalanceHistoryKey, userID)
}

func TransactionCacheKey(transactionID int64) string {
	return fmt.Sprintf(TransactionByIDKey, transactionID)
}

func TransactionUserCacheKey(userID int64) string {
	return fmt.Sprintf(TransactionByUserKey, userID)
}

func TransactionStatsCacheKey(userID int64) string {
	return fmt.Sprintf(TransactionStatsKey, userID)
}

func EventCacheKey(aggregateType, aggregateID string) string {
	return fmt.Sprintf(EventByAggregateKey, aggregateType, aggregateID)
}

func EventTypeCacheKey(eventType string) string {
	return fmt.Sprintf(EventByTypeKey, eventType)
}

// Cache invalidation helpers
func InvalidateUserCache(ctx context.Context, cache Cache, userID int64) error {
	keys := []string{
		UserCacheKey(userID),
		BalanceCacheKey(userID),
		BalanceHistoryCacheKey(userID),
		TransactionUserCacheKey(userID),
		TransactionStatsCacheKey(userID),
	}
	return cache.DeleteMultiple(ctx, keys)
}

func InvalidateBalanceCache(ctx context.Context, cache Cache, userID int64) error {
	keys := []string{
		BalanceCacheKey(userID),
		BalanceHistoryCacheKey(userID),
		TransactionStatsCacheKey(userID),
	}
	return cache.DeleteMultiple(ctx, keys)
}

func InvalidateTransactionCache(ctx context.Context, cache Cache, userID int64, transactionID int64) error {
	keys := []string{
		TransactionCacheKey(transactionID),
		TransactionUserCacheKey(userID),
		TransactionStatsCacheKey(userID),
		DashboardStatsKey,
		RecentTransactionsKey,
	}
	return cache.DeleteMultiple(ctx, keys)
}

// Helper function to copy data between interfaces
func copyData(src, dest interface{}) error {
	// This is a simple implementation. In production, you might want to use reflection
	// or a more sophisticated copying mechanism
	switch d := dest.(type) {
	case *interface{}:
		*d = src
		return nil
	default:
		// For typed destinations, you might need JSON marshal/unmarshal
		data, err := json.Marshal(src)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, dest)
	}
}
