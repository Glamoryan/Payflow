package service

import (
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionService struct {
	repo         domain.TransactionRepository
	balanceRepo  domain.BalanceRepository
	balanceSvc   domain.BalanceService
	auditLogRepo domain.AuditLogRepository
	logger       logger.Logger
}

func NewTransactionService(
	repo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	balanceSvc domain.BalanceService,
	auditLogRepo domain.AuditLogRepository,
	logger logger.Logger,
) domain.TransactionService {
	return &TransactionService{
		repo:         repo,
		balanceRepo:  balanceRepo,
		balanceSvc:   balanceSvc,
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

func (s *TransactionService) GetTransactionByID(id int64) (*domain.Transaction, error) {
	transaction, err := s.repo.FindByID(id)
	if err != nil {
		s.logger.Error("İşlem bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		return nil, fmt.Errorf("işlem bulunamadı: %w", err)
	}

	if transaction == nil {
		s.logger.Error("İşlem bulunamadı", map[string]interface{}{"id": id})
		return nil, fmt.Errorf("işlem bulunamadı: %d", id)
	}

	return transaction, nil
}

func (s *TransactionService) GetUserTransactions(userID int64) ([]*domain.Transaction, error) {
	transactions, err := s.repo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Kullanıcı işlemleri bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı işlemleri bulunamadı: %w", err)
	}

	return transactions, nil
}

func (s *TransactionService) DepositFunds(userID int64, amount float64) (*domain.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("geçersiz miktar: %.2f", amount)
	}

	balance, err := s.balanceRepo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("para yatırma işlemi yapılamadı: %w", err)
	}

	if balance == nil {
		if err := s.balanceSvc.InitializeBalance(userID); err != nil {
			s.logger.Error("Bakiye başlatılamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
			return nil, fmt.Errorf("para yatırma işlemi yapılamadı: %w", err)
		}
		balance, _ = s.balanceRepo.FindByUserID(userID)
	}

	transaction := &domain.Transaction{
		ToUserID:  &userID,
		Amount:    amount,
		Type:      domain.TransactionTypeDeposit,
		Status:    domain.TransactionStatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(transaction); err != nil {
		s.logger.Error("İşlem oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("para yatırma işlemi yapılamadı: %w", err)
	}

	newBalance := balance.Amount + amount
	if err := s.balanceSvc.UpdateBalance(userID, newBalance); err != nil {
		s.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": userID, "error": err.Error()})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("para yatırma işlemi yapılamadı: %w", err)
	}

	if err := s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": transaction.ID, "error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   transaction.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para yatırma işlemi: %.2f", amount),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": transaction.ID, "error": err.Error()})
	}

	return transaction, nil
}

func (s *TransactionService) WithdrawFunds(userID int64, amount float64) (*domain.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("geçersiz miktar: %.2f", amount)
	}

	balance, err := s.balanceRepo.FindByUserID(userID)
	if err != nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("para çekme işlemi yapılamadı: %w", err)
	}

	if balance == nil {
		s.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID})
		return nil, fmt.Errorf("kullanıcının bakiyesi bulunamadı: %d", userID)
	}

	if balance.Amount < amount {
		s.logger.Error("Yetersiz bakiye", map[string]interface{}{"user_id": userID, "balance": balance.Amount, "amount": amount})
		return nil, fmt.Errorf("yetersiz bakiye: %.2f, çekilmek istenen: %.2f", balance.Amount, amount)
	}

	transaction := &domain.Transaction{
		FromUserID: &userID,
		Amount:     amount,
		Type:       domain.TransactionTypeWithdraw,
		Status:     domain.TransactionStatusPending,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(transaction); err != nil {
		s.logger.Error("İşlem oluşturulamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("para çekme işlemi yapılamadı: %w", err)
	}

	newBalance := balance.Amount - amount
	if err := s.balanceSvc.UpdateBalance(userID, newBalance); err != nil {
		s.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": userID, "error": err.Error()})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("para çekme işlemi yapılamadı: %w", err)
	}

	if err := s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": transaction.ID, "error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   transaction.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para çekme işlemi: %.2f", amount),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": transaction.ID, "error": err.Error()})
	}

	return transaction, nil
}

func (s *TransactionService) TransferFunds(fromUserID, toUserID int64, amount float64) (*domain.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("geçersiz miktar: %.2f", amount)
	}

	if fromUserID == toUserID {
		return nil, fmt.Errorf("aynı kullanıcıya transfer yapılamaz")
	}

	fromBalance, err := s.balanceRepo.FindByUserID(fromUserID)
	if err != nil {
		s.logger.Error("Gönderen bakiyesi bulunamadı", map[string]interface{}{"user_id": fromUserID, "error": err.Error()})
		return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
	}

	if fromBalance == nil {
		s.logger.Error("Gönderen bakiyesi bulunamadı", map[string]interface{}{"user_id": fromUserID})
		return nil, fmt.Errorf("gönderen kullanıcının bakiyesi bulunamadı: %d", fromUserID)
	}

	if fromBalance.Amount < amount {
		s.logger.Error("Yetersiz bakiye", map[string]interface{}{"user_id": fromUserID, "balance": fromBalance.Amount, "amount": amount})
		return nil, fmt.Errorf("yetersiz bakiye: %.2f, transfer edilmek istenen: %.2f", fromBalance.Amount, amount)
	}

	toBalance, err := s.balanceRepo.FindByUserID(toUserID)
	if err != nil {
		s.logger.Error("Alıcı bakiyesi bulunamadı", map[string]interface{}{"user_id": toUserID, "error": err.Error()})
		return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
	}

	if toBalance == nil {
		if err := s.balanceSvc.InitializeBalance(toUserID); err != nil {
			s.logger.Error("Alıcı bakiyesi başlatılamadı", map[string]interface{}{"user_id": toUserID, "error": err.Error()})
			return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
		}
		toBalance, _ = s.balanceRepo.FindByUserID(toUserID)
	}

	transaction := &domain.Transaction{
		FromUserID: &fromUserID,
		ToUserID:   &toUserID,
		Amount:     amount,
		Type:       domain.TransactionTypeTransfer,
		Status:     domain.TransactionStatusPending,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(transaction); err != nil {
		s.logger.Error("İşlem oluşturulamadı", map[string]interface{}{"from_user_id": fromUserID, "to_user_id": toUserID, "error": err.Error()})
		return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
	}

	fromNewBalance := fromBalance.Amount - amount
	if err := s.balanceSvc.UpdateBalance(fromUserID, fromNewBalance); err != nil {
		s.logger.Error("Gönderen bakiyesi güncellenemedi", map[string]interface{}{"user_id": fromUserID, "error": err.Error()})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
	}

	toNewBalance := toBalance.Amount + amount
	if err := s.balanceSvc.UpdateBalance(toUserID, toNewBalance); err != nil {
		s.logger.Error("Alıcı bakiyesi güncellenemedi", map[string]interface{}{"user_id": toUserID, "error": err.Error()})

		if err := s.balanceSvc.UpdateBalance(fromUserID, fromBalance.Amount); err != nil {
			s.logger.Error("Gönderen bakiyesi geri yüklenemedi", map[string]interface{}{"user_id": fromUserID, "error": err.Error()})
		}

		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("transfer işlemi yapılamadı: %w", err)
	}

	if err := s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": transaction.ID, "error": err.Error()})
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   transaction.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para transferi: %.2f, %d -> %d", amount, fromUserID, toUserID),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": transaction.ID, "error": err.Error()})
	}

	return transaction, nil
}
