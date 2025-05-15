package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"payflow/internal/domain"
	"payflow/pkg/logger"
	"payflow/pkg/metrics"
	"payflow/pkg/tracing"
)

type BalanceService struct {
	repo         domain.BalanceRepository
	auditLogRepo domain.AuditLogRepository
	logger       logger.Logger
	redisClient  *redis.Client
	ctx          context.Context
}

func NewBalanceService(repo domain.BalanceRepository, auditLogRepo domain.AuditLogRepository, logger logger.Logger, redisClient *redis.Client) domain.BalanceService {
	return &BalanceService{
		repo:         repo,
		auditLogRepo: auditLogRepo,
		logger:       logger,
		redisClient:  redisClient,
		ctx:          context.Background(),
	}
}

func (s *BalanceService) GetUserBalance(userID int64) (*domain.Balance, error) {
	ctx, span := tracing.StartSpan(context.Background(), "BalanceService.GetUserBalance")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)

	startTime := time.Now()
	cachedBalance, err := s.getCachedBalanceFromRedis(userID)
	if err == nil && cachedBalance != nil {
		metrics.RecordCacheHit()
		tracing.AddAttribute(span, "cache_hit", true)

		s.logger.DebugContext(ctx, "Bakiye önbellekten alındı", map[string]interface{}{
			"user_id": userID,
		})
		return cachedBalance, nil
	}

	metrics.RecordCacheMiss()
	tracing.AddAttribute(span, "cache_hit", false)

	startTime = time.Now()
	balance, err := s.repo.FindByUserID(userID)
	metrics.RecordDatabaseOperation("read", "balance", time.Since(startTime))

	if err != nil {
		tracing.RecordError(span, err, "Bakiye veritabanından alınamadı")
		s.logger.ErrorContext(ctx, "Bakiye bulunamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("bakiye bulunamadı: %w", err)
	}

	if balance == nil {
		err := fmt.Errorf("kullanıcının bakiyesi bulunamadı: %d", userID)
		tracing.RecordError(span, err, "Bakiye bulunamadı")
		s.logger.ErrorContext(ctx, "Bakiye bulunamadı", map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	err = s.cacheBalanceToRedis(balance)
	if err != nil {
		s.logger.WarnContext(ctx, "Bakiye önbelleğe eklenemedi", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	tracing.AddAttribute(span, "balance_amount", balance.Amount)
	return balance, nil
}

func (s *BalanceService) InitializeBalance(userID int64) error {
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye kontrolü sırasında hata oluştu", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return fmt.Errorf("bakiye başlatılamadı: %w", err)
	}

	if balance != nil {
		s.logger.Info("Bakiye zaten mevcut", map[string]interface{}{"user_id": userID})
		return nil
	}

	newBalance := &domain.Balance{
		UserID:        userID,
		Amount:        0,
		LastUpdatedAt: time.Now(),
	}

	if err := s.repo.Create(newBalance); err != nil {
		s.logger.Error("Bakiye oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return fmt.Errorf("bakiye oluşturulamadı: %w", err)
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeCreate,
		Details:    "Yeni bakiye oluşturuldu",
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}

	return nil
}

func (s *BalanceService) UpdateBalance(userID int64, amount float64) error {
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return fmt.Errorf("bakiye güncellenemedi: %w", err)
	}

	if balance == nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID})
		return fmt.Errorf("kullanıcının bakiyesi bulunamadı: %d", userID)
	}

	oldAmount := balance.Amount
	balance.Amount = amount
	balance.LastUpdatedAt = time.Now()

	if err := s.repo.Update(balance); err != nil {
		s.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return fmt.Errorf("bakiye güncellenemedi: %w", err)
	}

	s.cacheBalanceToRedis(balance)

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeUpdate,
		Details:    fmt.Sprintf("Bakiye güncellendi: %.2f -> %.2f", oldAmount, amount),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}

	return nil
}

func (s *BalanceService) GetBalanceHistory(userID int64, limit, offset int) ([]*domain.BalanceHistory, error) {
	history, err := s.repo.GetBalanceHistory(userID, limit, offset)
	if err != nil {
		s.logger.Error("Bakiye geçmişi alınamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("bakiye geçmişi alınamadı: %w", err)
	}

	return history, nil
}

func (s *BalanceService) GetBalanceHistoryByDateRange(userID int64, startDate, endDate time.Time) ([]*domain.BalanceHistory, error) {
	history, err := s.repo.GetBalanceHistoryByDateRange(userID, startDate, endDate)
	if err != nil {
		s.logger.Error("Bakiye geçmişi alınamadı", map[string]interface{}{
			"user_id":    userID,
			"start_date": startDate,
			"end_date":   endDate,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("bakiye geçmişi alınamadı: %w", err)
	}

	return history, nil
}

func (s *BalanceService) DepositAtomically(userID int64, amount float64) (*domain.Balance, error) {
	ctx, span := tracing.StartSpan(context.Background(), "BalanceService.DepositAtomically")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)
	tracing.AddAttribute(span, "amount", amount)

	if amount <= 0 {
		err := fmt.Errorf("geçersiz miktar: %.2f", amount)
		tracing.RecordError(span, err, "Geçersiz para yatırma miktarı")
		return nil, err
	}

	startTime := time.Now()
	newAmount, err := s.repo.AtomicUpdate(userID, func(currentAmount float64) float64 {
		return currentAmount + amount
	})
	metrics.RecordDatabaseOperation("update", "balance", time.Since(startTime))

	if err != nil {
		tracing.RecordError(span, err, "Para yatırma işlemi sırasında hata")
		s.logger.Error("Para yatırma işlemi sırasında hata oluştu", map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("para yatırma işlemi yapılamadı: %w", err)
	}

	balanceUpdated := &domain.Balance{
		UserID:        userID,
		Amount:        newAmount,
		LastUpdatedAt: time.Now(),
	}

	s.cacheBalanceToRedis(balanceUpdated)

	metrics.RecordTransaction("deposit", "completed")
	tracing.AddAttribute(span, "new_balance", newAmount)

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeUpdate,
		Details:    fmt.Sprintf("Atomik para yatırma: +%.2f", amount),
		CreatedAt:  time.Now(),
	}

	startTime = time.Now()
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}
	metrics.RecordDatabaseOperation("create", "audit_log", time.Since(startTime))

	s.logger.InfoContext(ctx, "Para yatırma işlemi başarıyla tamamlandı", map[string]interface{}{
		"user_id":     userID,
		"amount":      amount,
		"new_balance": newAmount,
	})

	return balanceUpdated, nil
}

func (s *BalanceService) WithdrawAtomically(userID int64, amount float64) (*domain.Balance, error) {
	ctx, span := tracing.StartSpan(context.Background(), "BalanceService.WithdrawAtomically")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)
	tracing.AddAttribute(span, "amount", amount)

	if amount <= 0 {
		err := fmt.Errorf("geçersiz miktar: %.2f", amount)
		tracing.RecordError(span, err, "Geçersiz para çekme miktarı")
		return nil, err
	}

	startTime := time.Now()
	currentBalance, err := s.GetUserBalance(userID)
	metrics.RecordDatabaseOperation("read", "balance", time.Since(startTime))

	if err != nil {
		tracing.RecordError(span, err, "Mevcut bakiye alınamadı")
		return nil, err
	}

	previousAmount := currentBalance.Amount

	startTime = time.Now()
	newAmount, err := s.repo.AtomicUpdate(userID, func(currentAmount float64) float64 {
		if currentAmount < amount {
			return -1
		}
		return currentAmount - amount
	})
	metrics.RecordDatabaseOperation("update", "balance", time.Since(startTime))

	if err != nil {
		tracing.RecordError(span, err, "Para çekme işlemi sırasında hata")
		s.logger.ErrorContext(ctx, "Para çekme işlemi sırasında hata oluştu", map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("para çekme işlemi yapılamadı: %w", err)
	}

	if newAmount < 0 {
		err := fmt.Errorf("yetersiz bakiye: %.2f, çekilmek istenen: %.2f", currentBalance.Amount, amount)
		tracing.RecordError(span, err, "Yetersiz bakiye")
		s.logger.ErrorContext(ctx, "Yetersiz bakiye", map[string]interface{}{
			"user_id": userID,
			"balance": currentBalance.Amount,
			"amount":  amount,
		})
		return nil, err
	}

	balanceUpdated := &domain.Balance{
		UserID:        userID,
		Amount:        newAmount,
		LastUpdatedAt: time.Now(),
	}

	s.cacheBalanceToRedis(balanceUpdated)

	metrics.RecordTransaction("withdraw", "completed")
	tracing.AddAttribute(span, "new_balance", newAmount)

	history := &domain.BalanceHistory{
		UserID:         userID,
		Amount:         newAmount,
		PreviousAmount: previousAmount,
		TransactionID:  0,
		Operation:      "withdraw",
		CreatedAt:      time.Now(),
	}

	startTime = time.Now()
	if err := s.repo.AddBalanceHistory(history); err != nil {
		s.logger.WarnContext(ctx, "Bakiye geçmişi eklenemedi", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}
	metrics.RecordDatabaseOperation("create", "balance_history", time.Since(startTime))

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeUpdate,
		Details:    fmt.Sprintf("Atomik para çekme: -%.2f", amount),
		CreatedAt:  time.Now(),
	}

	startTime = time.Now()
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.WarnContext(ctx, "Denetim kaydı oluşturulamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}
	metrics.RecordDatabaseOperation("create", "audit_log", time.Since(startTime))

	s.logger.InfoContext(ctx, "Para çekme işlemi başarıyla tamamlandı", map[string]interface{}{
		"user_id":     userID,
		"amount":      amount,
		"new_balance": newAmount,
	})

	return balanceUpdated, nil
}

func (s *BalanceService) getCachedBalanceFromRedis(userID int64) (*domain.Balance, error) {
	key := fmt.Sprintf("balance:%d", userID)
	data, err := s.redisClient.Get(s.ctx, key).Result()

	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		s.logger.Error("Redis'ten bakiye alınamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
	}

	var balance domain.Balance
	if err := json.Unmarshal([]byte(data), &balance); err != nil {
		s.logger.Error("Redis'ten alınan veri çözümlenemedi", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
	}

	s.logger.Info("Redis önbelleğinden bakiye alındı", map[string]interface{}{
		"user_id": userID,
	})

	return &balance, nil
}

func (s *BalanceService) cacheBalanceToRedis(balance *domain.Balance) error {
	key := fmt.Sprintf("balance:%d", balance.UserID)
	data, err := json.Marshal(balance)
	if err != nil {
		s.logger.Error("Bakiye JSON formatına dönüştürülemedi", map[string]interface{}{
			"user_id": balance.UserID,
			"error":   err.Error(),
		})
		return err
	}

	err = s.redisClient.Set(s.ctx, key, data, 15*time.Minute).Err()
	if err != nil {
		s.logger.Error("Bakiye Redis'e kaydedilemedi", map[string]interface{}{
			"user_id": balance.UserID,
			"error":   err.Error(),
		})
		return err
	}

	s.logger.Info("Bakiye Redis önbelleğine kaydedildi", map[string]interface{}{
		"user_id": balance.UserID,
	})

	return nil
}

func (s *BalanceService) GetCachedBalance(userID int64) (*domain.Balance, error) {
	balance, err := s.getCachedBalanceFromRedis(userID)
	if err == nil && balance != nil {
		return balance, nil
	}

	balance, err = s.GetUserBalance(userID)
	if err != nil {
		return nil, err
	}

	s.cacheBalanceToRedis(balance)
	return balance, nil
}

func (s *BalanceService) InvalidateCache(userID int64) error {
	key := fmt.Sprintf("balance:%d", userID)
	err := s.redisClient.Del(s.ctx, key).Err()
	if err != nil {
		s.logger.Error("Redis önbelleği temizlenemedi", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return err
	}

	s.logger.Info("Redis önbelleği temizlendi", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

func (s *BalanceService) RecalculateBalance(userID int64) (*domain.Balance, error) {
	balance, err := s.GetUserBalance(userID)
	if err != nil {
		return nil, err
	}

	startDate := time.Time{}
	endDate := time.Now()

	history, err := s.GetBalanceHistoryByDateRange(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(history) > 0 {
		lastHistoryRecord := history[0]

		if lastHistoryRecord.Amount != balance.Amount {
			s.logger.Warn("Bakiye tutarsızlığı tespit edildi", map[string]interface{}{
				"user_id":            userID,
				"db_balance":         balance.Amount,
				"calculated_balance": lastHistoryRecord.Amount,
			})

			balance.Amount = lastHistoryRecord.Amount
			if err := s.repo.Update(balance); err != nil {
				return nil, fmt.Errorf("bakiye güncellenemedi: %w", err)
			}

			s.cacheBalanceToRedis(balance)

			s.logger.Info("Bakiye düzeltildi", map[string]interface{}{
				"user_id":    userID,
				"new_amount": balance.Amount,
			})
		}
	}

	return balance, nil
}
