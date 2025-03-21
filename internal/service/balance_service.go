package service

import (
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type BalanceService struct {
	repo         domain.BalanceRepository
	auditLogRepo domain.AuditLogRepository
	logger       logger.Logger
}

func NewBalanceService(repo domain.BalanceRepository, auditLogRepo domain.AuditLogRepository, logger logger.Logger) domain.BalanceService {
	return &BalanceService{
		repo:         repo,
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

func (s *BalanceService) GetUserBalance(userID int64) (*domain.Balance, error) {
	balance, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("bakiye bulunamadı: %w", err)
	}

	if balance == nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID})
		return nil, fmt.Errorf("kullanıcının bakiyesi bulunamadı: %d", userID)
	}

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
