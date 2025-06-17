package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

// WarmUpManager handles cache warming strategies
type WarmUpManager struct {
	cache          Cache
	logger         logger.Logger
	userService    domain.UserService
	balanceService domain.BalanceService
	txService      domain.TransactionService
}

// NewWarmUpManager creates a new warm-up manager
func NewWarmUpManager(
	cache Cache,
	logger logger.Logger,
	userService domain.UserService,
	balanceService domain.BalanceService,
	txService domain.TransactionService,
) *WarmUpManager {
	return &WarmUpManager{
		cache:          cache,
		logger:         logger,
		userService:    userService,
		balanceService: balanceService,
		txService:      txService,
	}
}

// WarmUpUserData warms up user-related cache data
func (w *WarmUpManager) WarmUpUserData(ctx context.Context, userID int64) error {
	w.logger.Info("User data warm-up başlatılıyor", map[string]interface{}{"userID": userID})

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Warm up user data
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpUser(ctx, userID); err != nil {
			errChan <- fmt.Errorf("user warm-up hatası: %w", err)
		}
	}()

	// Warm up balance data
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpBalance(ctx, userID); err != nil {
			errChan <- fmt.Errorf("balance warm-up hatası: %w", err)
		}
	}()

	// Warm up transaction data
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpTransactions(ctx, userID); err != nil {
			errChan <- fmt.Errorf("transaction warm-up hatası: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			w.logger.Error("Warm-up hatası", map[string]interface{}{
				"userID": userID,
				"error":  err.Error(),
			})
			return err
		}
	}

	w.logger.Info("User data warm-up tamamlandı", map[string]interface{}{"userID": userID})
	return nil
}

// WarmUpTopUsers warms up data for top active users
func (w *WarmUpManager) WarmUpTopUsers(ctx context.Context, limit int) error {
	w.logger.Info("Top users warm-up başlatılıyor", map[string]interface{}{"limit": limit})

	// For this example, we'll warm up users with IDs 1 to limit
	// In production, you'd fetch actual top user IDs from database
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent warm-ups

	for i := int64(1); i <= int64(limit); i++ {
		wg.Add(1)
		go func(userID int64) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			if err := w.WarmUpUserData(ctx, userID); err != nil {
				w.logger.Error("Top user warm-up hatası", map[string]interface{}{
					"userID": userID,
					"error":  err.Error(),
				})
			}
		}(i)
	}

	wg.Wait()
	w.logger.Info("Top users warm-up tamamlandı", map[string]interface{}{"limit": limit})
	return nil
}

// WarmUpFrequentlyAccessedData warms up commonly accessed data
func (w *WarmUpManager) WarmUpFrequentlyAccessedData(ctx context.Context) error {
	w.logger.Info("Frequently accessed data warm-up başlatılıyor", map[string]interface{}{})

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Warm up dashboard stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpDashboardStats(ctx); err != nil {
			errChan <- fmt.Errorf("dashboard stats warm-up hatası: %w", err)
		}
	}()

	// Warm up recent transactions
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpRecentTransactions(ctx); err != nil {
			errChan <- fmt.Errorf("recent transactions warm-up hatası: %w", err)
		}
	}()

	// Warm up top users list
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.warmUpTopUsersList(ctx); err != nil {
			errChan <- fmt.Errorf("top users warm-up hatası: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			w.logger.Error("Frequently accessed data warm-up hatası", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
	}

	w.logger.Info("Frequently accessed data warm-up tamamlandı", map[string]interface{}{})
	return nil
}

// ScheduledWarmUp performs scheduled cache warm-up
func (w *WarmUpManager) ScheduledWarmUp(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.logger.Info("Scheduled warm-up başlatıldı", map[string]interface{}{
		"interval": interval,
	})

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Scheduled warm-up durduruldu", map[string]interface{}{})
			return
		case <-ticker.C:
			w.logger.Debug("Scheduled warm-up çalışıyor", map[string]interface{}{})

			if err := w.WarmUpFrequentlyAccessedData(ctx); err != nil {
				w.logger.Error("Scheduled warm-up hatası", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// warmUpUser warms up user cache
func (w *WarmUpManager) warmUpUser(ctx context.Context, userID int64) error {
	user, err := w.userService.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Cache user by ID
	key := UserCacheKey(userID)
	if err := w.cache.Set(ctx, key, user, LongExpiration); err != nil {
		return err
	}

	// Cache user by username
	if user.Username != "" {
		usernameKey := UserCacheKeyByUsername(user.Username)
		if err := w.cache.Set(ctx, usernameKey, user, LongExpiration); err != nil {
			return err
		}
	}

	// Cache user by email
	if user.Email != "" {
		emailKey := UserCacheKeyByEmail(user.Email)
		if err := w.cache.Set(ctx, emailKey, user, LongExpiration); err != nil {
			return err
		}
	}

	w.logger.Debug("User cache warmed up", map[string]interface{}{"userID": userID})
	return nil
}

// warmUpBalance warms up balance cache
func (w *WarmUpManager) warmUpBalance(ctx context.Context, userID int64) error {
	balance, err := w.balanceService.GetBalance(userID)
	if err != nil {
		return err
	}

	// Cache current balance
	key := BalanceCacheKey(userID)
	if err := w.cache.Set(ctx, key, balance, MediumExpiration); err != nil {
		return err
	}

	// Cache balance history
	history, err := w.balanceService.GetBalanceHistory(userID, time.Now().Add(-30*24*time.Hour), time.Now())
	if err == nil {
		historyKey := BalanceHistoryCacheKey(userID)
		if err := w.cache.Set(ctx, historyKey, history, LongExpiration); err != nil {
			w.logger.Error("Balance history cache set hatası", map[string]interface{}{
				"userID": userID,
				"error":  err.Error(),
			})
		}
	}

	w.logger.Debug("Balance cache warmed up", map[string]interface{}{"userID": userID})
	return nil
}

// warmUpTransactions warms up transaction cache
func (w *WarmUpManager) warmUpTransactions(ctx context.Context, userID int64) error {
	// Cache recent user transactions
	transactions, err := w.txService.GetUserTransactions(userID)
	if err != nil {
		return err
	}

	key := TransactionUserCacheKey(userID)
	if err := w.cache.Set(ctx, key, transactions, MediumExpiration); err != nil {
		return err
	}

	// Cache individual transactions
	for _, tx := range transactions {
		txKey := TransactionCacheKey(tx.ID)
		if err := w.cache.Set(ctx, txKey, tx, LongExpiration); err != nil {
			w.logger.Error("Transaction cache set hatası", map[string]interface{}{
				"transactionID": tx.ID,
				"error":         err.Error(),
			})
		}
	}

	w.logger.Debug("Transaction cache warmed up", map[string]interface{}{"userID": userID})
	return nil
}

// warmUpDashboardStats warms up dashboard statistics
func (w *WarmUpManager) warmUpDashboardStats(ctx context.Context) error {
	// Mock dashboard stats - in production, you'd fetch real stats
	stats := map[string]interface{}{
		"total_users":        1000,
		"total_transactions": 5000,
		"total_volume":       1000000.0,
		"active_users_today": 150,
		"updated_at":         time.Now(),
	}

	if err := w.cache.Set(ctx, DashboardStatsKey, stats, ShortExpiration); err != nil {
		return err
	}

	w.logger.Debug("Dashboard stats cache warmed up", map[string]interface{}{})
	return nil
}

// warmUpRecentTransactions warms up recent transactions list
func (w *WarmUpManager) warmUpRecentTransactions(ctx context.Context) error {
	// Mock recent transactions - in production, you'd fetch real recent transactions
	recentTxs := []map[string]interface{}{
		{"id": 1, "amount": 100.0, "type": "deposit", "timestamp": time.Now()},
		{"id": 2, "amount": 50.0, "type": "withdrawal", "timestamp": time.Now().Add(-time.Hour)},
	}

	if err := w.cache.Set(ctx, RecentTransactionsKey, recentTxs, ShortExpiration); err != nil {
		return err
	}

	w.logger.Debug("Recent transactions cache warmed up", map[string]interface{}{})
	return nil
}

// warmUpTopUsersList warms up top users list
func (w *WarmUpManager) warmUpTopUsersList(ctx context.Context) error {
	// Mock top users - in production, you'd fetch real top users by activity/balance
	topUsers := []map[string]interface{}{
		{"id": 1, "username": "user1", "balance": 1000.0, "rank": 1},
		{"id": 2, "username": "user2", "balance": 800.0, "rank": 2},
		{"id": 3, "username": "user3", "balance": 600.0, "rank": 3},
	}

	if err := w.cache.Set(ctx, TopUsersKey, topUsers, MediumExpiration); err != nil {
		return err
	}

	w.logger.Debug("Top users cache warmed up", map[string]interface{}{})
	return nil
}
