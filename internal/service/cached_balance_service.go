package service

import (
	"context"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/cache"
	"payflow/pkg/logger"
)

// CachedBalanceService wraps BalanceService with caching functionality
type CachedBalanceService struct {
	balanceService domain.BalanceService
	cache          cache.Cache
	cacheManager   cache.CacheStrategy
	logger         logger.Logger
}

// NewCachedBalanceService creates a new cached balance service
func NewCachedBalanceService(
	balanceService domain.BalanceService,
	cacheInstance cache.Cache,
	cacheManager cache.CacheStrategy,
	logger logger.Logger,
) domain.BalanceService {
	return &CachedBalanceService{
		balanceService: balanceService,
		cache:          cacheInstance,
		cacheManager:   cacheManager,
		logger:         logger,
	}
}

func (s *CachedBalanceService) GetBalance(userID int64) (*domain.Balance, error) {
	ctx := context.Background()
	key := cache.BalanceCacheKey(userID)

	var balance *domain.Balance
	err := s.cacheManager.ReadThrough(ctx, key, &balance, func() (interface{}, error) {
		return s.balanceService.GetBalance(userID)
	}, cache.MediumExpiration)

	if err != nil {
		s.logger.Error("Cache read-through error for balance", map[string]interface{}{
			"userID": userID,
			"error":  err.Error(),
		})
		// Fallback to direct service call
		return s.balanceService.GetBalance(userID)
	}

	return balance, nil
}

func (s *CachedBalanceService) DepositAtomically(userID int64, amount float64) (*domain.Balance, error) {
	// Perform the deposit operation
	balance, err := s.balanceService.DepositAtomically(userID, amount)
	if err != nil {
		return nil, err
	}

	// Invalidate related cache entries
	ctx := context.Background()
	if cacheErr := cache.InvalidateBalanceCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating balance cache after deposit", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	// Cache the new balance
	key := cache.BalanceCacheKey(userID)
	if setErr := s.cache.Set(ctx, key, balance, cache.MediumExpiration); setErr != nil {
		s.logger.Error("Error caching balance after deposit", map[string]interface{}{
			"userID": userID,
			"error":  setErr.Error(),
		})
	}

	return balance, nil
}

func (s *CachedBalanceService) WithdrawAtomically(userID int64, amount float64) (*domain.Balance, error) {
	// Perform the withdrawal operation
	balance, err := s.balanceService.WithdrawAtomically(userID, amount)
	if err != nil {
		return nil, err
	}

	// Invalidate related cache entries
	ctx := context.Background()
	if cacheErr := cache.InvalidateBalanceCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating balance cache after withdrawal", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	// Cache the new balance
	key := cache.BalanceCacheKey(userID)
	if setErr := s.cache.Set(ctx, key, balance, cache.MediumExpiration); setErr != nil {
		s.logger.Error("Error caching balance after withdrawal", map[string]interface{}{
			"userID": userID,
			"error":  setErr.Error(),
		})
	}

	return balance, nil
}

func (s *CachedBalanceService) InitializeBalance(userID int64) error {
	err := s.balanceService.InitializeBalance(userID)
	if err != nil {
		return err
	}

	// Invalidate cache as new balance has been initialized
	ctx := context.Background()
	if cacheErr := cache.InvalidateBalanceCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating balance cache after initialization", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	return nil
}

func (s *CachedBalanceService) GetBalanceHistory(userID int64, startTime, endTime time.Time) ([]*domain.Balance, error) {
	ctx := context.Background()
	key := cache.BalanceHistoryCacheKey(userID)

	var history []*domain.Balance
	err := s.cacheManager.ReadThrough(ctx, key, &history, func() (interface{}, error) {
		return s.balanceService.GetBalanceHistory(userID, startTime, endTime)
	}, cache.LongExpiration)

	if err != nil {
		s.logger.Error("Cache read-through error for balance history", map[string]interface{}{
			"userID": userID,
			"error":  err.Error(),
		})
		// Fallback to direct service call
		return s.balanceService.GetBalanceHistory(userID, startTime, endTime)
	}

	return history, nil
}

func (s *CachedBalanceService) ReplayBalanceEvents(userID int64) error {
	err := s.balanceService.ReplayBalanceEvents(userID)
	if err != nil {
		return err
	}

	// Invalidate balance cache as state might have changed
	ctx := context.Background()
	if cacheErr := cache.InvalidateBalanceCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating balance cache after replay", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	return nil
}

func (s *CachedBalanceService) RebuildBalanceState(userID int64) error {
	err := s.balanceService.RebuildBalanceState(userID)
	if err != nil {
		return err
	}

	// Invalidate balance cache as state has been rebuilt
	ctx := context.Background()
	if cacheErr := cache.InvalidateBalanceCache(ctx, s.cache, userID); cacheErr != nil {
		s.logger.Error("Error invalidating balance cache after rebuild", map[string]interface{}{
			"userID": userID,
			"error":  cacheErr.Error(),
		})
	}

	return nil
}
