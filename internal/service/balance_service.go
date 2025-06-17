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
	eventStore   domain.EventStoreService
	logger       logger.Logger
	redisClient  *redis.Client
}

func NewBalanceService(
	repo domain.BalanceRepository,
	auditLogRepo domain.AuditLogRepository,
	eventStore domain.EventStoreService,
	logger logger.Logger,
	redisClient *redis.Client,
) domain.BalanceService {
	return &BalanceService{
		repo:         repo,
		auditLogRepo: auditLogRepo,
		eventStore:   eventStore,
		logger:       logger,
		redisClient:  redisClient,
	}
}

func (s *BalanceService) saveEvent(balance *domain.Balance, eventType domain.EventType) error {
	eventData, err := json.Marshal(balance)
	if err != nil {
		return err
	}

	lastVersion, err := s.eventStore.GetLastVersion("balance", fmt.Sprintf("%d", balance.UserID))
	if err != nil {
		return err
	}

	event := &domain.Event{
		AggregateID:   fmt.Sprintf("%d", balance.UserID),
		AggregateType: "balance",
		EventType:     eventType,
		EventData:     eventData,
		Version:       lastVersion + 1,
		CreatedAt:     time.Now(),
	}

	return s.eventStore.SaveEvent(event)
}

func (s *BalanceService) GetBalance(userID int64) (*domain.Balance, error) {
	_, span := tracing.StartSpan(context.Background(), "BalanceService.GetBalance")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)

	startTime := time.Now()
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("find", "balance", time.Since(startTime))

	return balance, nil
}

func (s *BalanceService) DepositAtomically(userID int64, amount float64) (*domain.Balance, error) {
	_, span := tracing.StartSpan(context.Background(), "BalanceService.DepositAtomically")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)
	tracing.AddAttribute(span, "amount", amount)

	startTime := time.Now()
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("find", "balance", time.Since(startTime))

	if balance == nil {
		balance = &domain.Balance{
			UserID:        userID,
			Amount:        0,
			LastUpdatedAt: time.Now(),
		}
	}

	newAmount := balance.Amount + amount
	balance.Amount = newAmount
	balance.LastUpdatedAt = time.Now()

	startTime = time.Now()
	balanceUpdated, err := s.repo.Update(balance)
	if err != nil {
		s.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("update", "balance", time.Since(startTime))

	if err := s.saveEvent(balanceUpdated, domain.EventTypeBalanceUpdated); err != nil {
		s.logger.Error("Event kaydedilemedi", map[string]interface{}{"error": err.Error()})
	}

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

	s.logger.InfoContext(context.Background(), "Para yatırma işlemi başarıyla tamamlandı", map[string]interface{}{
		"user_id":     userID,
		"amount":      amount,
		"new_balance": newAmount,
	})

	return balanceUpdated, nil
}

func (s *BalanceService) WithdrawAtomically(userID int64, amount float64) (*domain.Balance, error) {
	_, span := tracing.StartSpan(context.Background(), "BalanceService.WithdrawAtomically")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)
	tracing.AddAttribute(span, "amount", amount)

	startTime := time.Now()
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("find", "balance", time.Since(startTime))

	if balance == nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID})
		return nil, domain.ErrBalanceNotFound
	}

	if balance.Amount < amount {
		s.logger.Error("Yetersiz bakiye", map[string]interface{}{"user_id": userID, "balance": balance.Amount, "amount": amount})
		return nil, domain.ErrInsufficientFunds
	}

	newAmount := balance.Amount - amount
	balance.Amount = newAmount
	balance.LastUpdatedAt = time.Now()

	startTime = time.Now()
	balanceUpdated, err := s.repo.Update(balance)
	if err != nil {
		s.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("update", "balance", time.Since(startTime))

	if err := s.saveEvent(balanceUpdated, domain.EventTypeBalanceUpdated); err != nil {
		s.logger.Error("Event kaydedilemedi", map[string]interface{}{"error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeUpdate,
		Details:    fmt.Sprintf("Atomik para çekme: -%.2f", amount),
		CreatedAt:  time.Now(),
	}

	startTime = time.Now()
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}
	metrics.RecordDatabaseOperation("create", "audit_log", time.Since(startTime))

	s.logger.InfoContext(context.Background(), "Para çekme işlemi başarıyla tamamlandı", map[string]interface{}{
		"user_id":     userID,
		"amount":      amount,
		"new_balance": newAmount,
	})

	return balanceUpdated, nil
}

func (s *BalanceService) InitializeBalance(userID int64) error {
	_, span := tracing.StartSpan(context.Background(), "BalanceService.InitializeBalance")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)

	startTime := time.Now()
	err := s.repo.InitializeBalance(userID)
	if err != nil {
		s.logger.Error("Bakiye başlatılamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return err
	}
	metrics.RecordDatabaseOperation("initialize", "balance", time.Since(startTime))

	balance := &domain.Balance{
		UserID:        userID,
		Amount:        0,
		LastUpdatedAt: time.Now(),
	}

	if err := s.saveEvent(balance, domain.EventTypeBalanceUpdated); err != nil {
		s.logger.Error("Event kaydedilemedi", map[string]interface{}{"error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeBalance,
		EntityID:   userID,
		Action:     domain.ActionTypeCreate,
		Details:    "Bakiye başlatıldı",
		CreatedAt:  time.Now(),
	}

	startTime = time.Now()
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
	}
	metrics.RecordDatabaseOperation("create", "audit_log", time.Since(startTime))

	s.logger.InfoContext(context.Background(), "Bakiye başarıyla başlatıldı", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

func (s *BalanceService) GetBalanceHistory(userID int64, startTime, endTime time.Time) ([]*domain.Balance, error) {
	_, span := tracing.StartSpan(context.Background(), "BalanceService.GetBalanceHistory")
	defer span.End()

	tracing.AddAttribute(span, "user_id", userID)
	tracing.AddAttribute(span, "start_time", startTime)
	tracing.AddAttribute(span, "end_time", endTime)

	startTime = time.Now()
	history, err := s.repo.GetBalanceHistory(userID, startTime, endTime)
	if err != nil {
		s.logger.Error("Bakiye geçmişi alınamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, err
	}
	metrics.RecordDatabaseOperation("find", "balance_history", time.Since(startTime))

	return history, nil
}

func (s *BalanceService) ReplayBalanceEvents(userID int64) error {
	events, err := s.eventStore.GetAggregateEvents("balance", fmt.Sprintf("%d", userID))
	if err != nil {
		return err
	}

	for _, event := range events {
		var balance domain.Balance
		if err := json.Unmarshal(event.EventData, &balance); err != nil {
			return err
		}

		switch event.EventType {
		case domain.EventTypeBalanceUpdated:
			if _, err := s.repo.Update(&balance); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *BalanceService) RebuildBalanceState(userID int64) error {
	events, err := s.eventStore.GetAggregateEvents("balance", fmt.Sprintf("%d", userID))
	if err != nil {
		return err
	}

	var balance domain.Balance
	for _, event := range events {
		if err := json.Unmarshal(event.EventData, &balance); err != nil {
			return err
		}

		switch event.EventType {
		case domain.EventTypeBalanceUpdated:
			if _, err := s.repo.Update(&balance); err != nil {
				return err
			}
		}
	}

	return nil
}
