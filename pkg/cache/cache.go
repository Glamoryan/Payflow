package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"payflow/pkg/logger"
)

// Cache interface - caching operations
type Cache interface {
	// Basic operations
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Pattern-based operations
	DeletePattern(ctx context.Context, pattern string) error
	GetKeys(ctx context.Context, pattern string) ([]string, error)

	// Batch operations
	SetMultiple(ctx context.Context, items map[string]interface{}, expiration time.Duration) error
	GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error)
	DeleteMultiple(ctx context.Context, keys []string) error

	// Cache warm-up and invalidation
	WarmUp(ctx context.Context, warmUpFunc func(ctx context.Context) error) error
	InvalidatePrefix(ctx context.Context, prefix string) error

	// Health check
	Ping(ctx context.Context) error
}

// RedisCache implements Cache interface
type RedisCache struct {
	client *redis.Client
	logger logger.Logger
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, logger logger.Logger, prefix string) Cache {
	return &RedisCache{
		client: client,
		logger: logger,
		prefix: prefix,
	}
}

// makeKey adds prefix to the key
func (r *RedisCache) makeKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

// Set stores a value in cache
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Error("Cache set marshal hatası", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	fullKey := r.makeKey(key)
	err = r.client.Set(ctx, fullKey, data, expiration).Err()
	if err != nil {
		r.logger.Error("Cache set hatası", map[string]interface{}{
			"key":   fullKey,
			"error": err.Error(),
		})
		return err
	}

	r.logger.Debug("Cache set başarılı", map[string]interface{}{
		"key":        fullKey,
		"expiration": expiration,
	})
	return nil
}

// Get retrieves a value from cache
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.makeKey(key)
	data, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			r.logger.Debug("Cache miss", map[string]interface{}{"key": fullKey})
			return ErrCacheMiss
		}
		r.logger.Error("Cache get hatası", map[string]interface{}{
			"key":   fullKey,
			"error": err.Error(),
		})
		return err
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		r.logger.Error("Cache get unmarshal hatası", map[string]interface{}{
			"key":   fullKey,
			"error": err.Error(),
		})
		return err
	}

	r.logger.Debug("Cache hit", map[string]interface{}{"key": fullKey})
	return nil
}

// Delete removes a key from cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.makeKey(key)
	err := r.client.Del(ctx, fullKey).Err()
	if err != nil {
		r.logger.Error("Cache delete hatası", map[string]interface{}{
			"key":   fullKey,
			"error": err.Error(),
		})
		return err
	}

	r.logger.Debug("Cache delete başarılı", map[string]interface{}{"key": fullKey})
	return nil
}

// Exists checks if a key exists in cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.makeKey(key)
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("Cache exists hatası", map[string]interface{}{
			"key":   fullKey,
			"error": err.Error(),
		})
		return false, err
	}

	return count > 0, nil
}

// DeletePattern deletes all keys matching a pattern
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := r.makeKey(pattern)
	keys, err := r.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		r.logger.Error("Cache delete pattern hatası", map[string]interface{}{
			"pattern": fullPattern,
			"error":   err.Error(),
		})
		return err
	}

	if len(keys) == 0 {
		r.logger.Debug("Cache delete pattern - anahtar bulunamadı", map[string]interface{}{
			"pattern": fullPattern,
		})
		return nil
	}

	err = r.client.Del(ctx, keys...).Err()
	if err != nil {
		r.logger.Error("Cache delete pattern hatası", map[string]interface{}{
			"pattern": fullPattern,
			"keys":    len(keys),
			"error":   err.Error(),
		})
		return err
	}

	r.logger.Info("Cache delete pattern başarılı", map[string]interface{}{
		"pattern":      fullPattern,
		"deleted_keys": len(keys),
	})
	return nil
}

// GetKeys returns all keys matching a pattern
func (r *RedisCache) GetKeys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := r.makeKey(pattern)
	keys, err := r.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		r.logger.Error("Cache get keys hatası", map[string]interface{}{
			"pattern": fullPattern,
			"error":   err.Error(),
		})
		return nil, err
	}

	// Remove prefix from keys
	if r.prefix != "" {
		for i, key := range keys {
			keys[i] = strings.TrimPrefix(key, r.prefix+":")
		}
	}

	return keys, nil
}

// SetMultiple sets multiple key-value pairs
func (r *RedisCache) SetMultiple(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			r.logger.Error("Cache set multiple marshal hatası", map[string]interface{}{
				"key":   key,
				"error": err.Error(),
			})
			return err
		}

		fullKey := r.makeKey(key)
		pipe.Set(ctx, fullKey, data, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Cache set multiple hatası", map[string]interface{}{
			"count": len(items),
			"error": err.Error(),
		})
		return err
	}

	r.logger.Debug("Cache set multiple başarılı", map[string]interface{}{
		"count":      len(items),
		"expiration": expiration,
	})
	return nil
}

// GetMultiple gets multiple values by keys
func (r *RedisCache) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.makeKey(key)
	}

	values, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		r.logger.Error("Cache get multiple hatası", map[string]interface{}{
			"keys":  len(keys),
			"error": err.Error(),
		})
		return nil, err
	}

	result := make(map[string]interface{})
	for i, val := range values {
		if val != nil {
			var data interface{}
			if strVal, ok := val.(string); ok {
				if err := json.Unmarshal([]byte(strVal), &data); err == nil {
					result[keys[i]] = data
				}
			}
		}
	}

	r.logger.Debug("Cache get multiple başarılı", map[string]interface{}{
		"requested": len(keys),
		"found":     len(result),
	})
	return result, nil
}

// DeleteMultiple deletes multiple keys
func (r *RedisCache) DeleteMultiple(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.makeKey(key)
	}

	err := r.client.Del(ctx, fullKeys...).Err()
	if err != nil {
		r.logger.Error("Cache delete multiple hatası", map[string]interface{}{
			"keys":  len(keys),
			"error": err.Error(),
		})
		return err
	}

	r.logger.Debug("Cache delete multiple başarılı", map[string]interface{}{
		"count": len(keys),
	})
	return nil
}

// WarmUp executes cache warm-up function
func (r *RedisCache) WarmUp(ctx context.Context, warmUpFunc func(ctx context.Context) error) error {
	r.logger.Info("Cache warm-up başlatılıyor", map[string]interface{}{})

	start := time.Now()
	err := warmUpFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("Cache warm-up hatası", map[string]interface{}{
			"duration": duration,
			"error":    err.Error(),
		})
		return err
	}

	r.logger.Info("Cache warm-up tamamlandı", map[string]interface{}{
		"duration": duration,
	})
	return nil
}

// InvalidatePrefix invalidates all keys with given prefix
func (r *RedisCache) InvalidatePrefix(ctx context.Context, prefix string) error {
	pattern := fmt.Sprintf("%s*", prefix)
	return r.DeletePattern(ctx, pattern)
}

// Ping checks Redis connection
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Custom errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)
