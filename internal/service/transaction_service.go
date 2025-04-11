package service

import (
	"fmt"
	"sync"
	"time"

	"payflow/internal/concurrent"
	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionService struct {
	repo         domain.TransactionRepository
	balanceRepo  domain.BalanceRepository
	balanceSvc   domain.BalanceService
	auditLogRepo domain.AuditLogRepository
	logger       logger.Logger

	// Worker pool ve istatistikler için alanlar
	workerPool          *concurrent.WorkerPool
	pendingTransactions sync.Map // ID -> Transaction
	initialized         bool
	initMutex           sync.Mutex
}

func NewTransactionService(
	repo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	balanceSvc domain.BalanceService,
	auditLogRepo domain.AuditLogRepository,
	logger logger.Logger,
) domain.TransactionService {
	svc := &TransactionService{
		repo:         repo,
		balanceRepo:  balanceRepo,
		balanceSvc:   balanceSvc,
		auditLogRepo: auditLogRepo,
		logger:       logger,
		initialized:  false,
	}

	return svc
}

func (s *TransactionService) initWorkerPool() {
	s.initMutex.Lock()
	defer s.initMutex.Unlock()

	if s.initialized {
		return
	}

	processor := func(tx *domain.Transaction) error {
		switch tx.Type {
		case domain.TransactionTypeDeposit:
			return s.processDeposit(tx)
		case domain.TransactionTypeWithdraw:
			return s.processWithdraw(tx)
		case domain.TransactionTypeTransfer:
			return s.processTransfer(tx)
		default:
			return fmt.Errorf("bilinmeyen işlem tipi: %s", tx.Type)
		}
	}

	s.workerPool = concurrent.NewWorkerPool(5, 100, processor, s.logger)
	s.workerPool.Start()
	s.initialized = true

	s.logger.Info("İşlem worker pool'u başlatıldı", map[string]interface{}{})
}

func (s *TransactionService) ensureWorkerPoolInitialized() {
	if !s.initialized {
		s.initWorkerPool()
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
	s.ensureWorkerPoolInitialized()

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

	submitted := s.workerPool.Submit(transaction)
	if !submitted {
		s.logger.Error("İşlem kuyruğa eklenemedi", map[string]interface{}{"transaction_id": transaction.ID})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("işlem şu anda işlenemiyor, lütfen daha sonra tekrar deneyin")
	}

	s.pendingTransactions.Store(transaction.ID, transaction)

	return transaction, nil
}

func (s *TransactionService) WithdrawFunds(userID int64, amount float64) (*domain.Transaction, error) {
	s.ensureWorkerPoolInitialized()

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

	submitted := s.workerPool.Submit(transaction)
	if !submitted {
		s.logger.Error("İşlem kuyruğa eklenemedi", map[string]interface{}{"transaction_id": transaction.ID})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("işlem şu anda işlenemiyor, lütfen daha sonra tekrar deneyin")
	}

	s.pendingTransactions.Store(transaction.ID, transaction)

	return transaction, nil
}

func (s *TransactionService) TransferFunds(fromUserID, toUserID int64, amount float64) (*domain.Transaction, error) {
	s.ensureWorkerPoolInitialized()

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

	submitted := s.workerPool.Submit(transaction)
	if !submitted {
		s.logger.Error("İşlem kuyruğa eklenemedi", map[string]interface{}{"transaction_id": transaction.ID})
		s.repo.UpdateStatus(transaction.ID, domain.TransactionStatusFailed)
		return nil, fmt.Errorf("işlem şu anda işlenemiyor, lütfen daha sonra tekrar deneyin")
	}

	s.pendingTransactions.Store(transaction.ID, transaction)

	return transaction, nil
}

func (s *TransactionService) processDeposit(tx *domain.Transaction) error {
	userID := *tx.ToUserID

	_, err := s.balanceSvc.DepositAtomically(userID, tx.Amount)
	if err != nil {
		s.logger.Error("Para yatırma işlemi başarısız oldu", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
		s.repo.UpdateStatus(tx.ID, domain.TransactionStatusFailed)
		return err
	}

	if err := s.repo.UpdateStatus(tx.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": tx.ID, "error": err.Error()})
		return err
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   tx.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para yatırma işlemi: %.2f", tx.Amount),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
	}

	s.pendingTransactions.Delete(tx.ID)

	return nil
}

func (s *TransactionService) processWithdraw(tx *domain.Transaction) error {
	userID := *tx.FromUserID

	_, err := s.balanceSvc.WithdrawAtomically(userID, tx.Amount)
	if err != nil {
		s.logger.Error("Para çekme işlemi başarısız oldu", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
		s.repo.UpdateStatus(tx.ID, domain.TransactionStatusFailed)
		return err
	}

	if err := s.repo.UpdateStatus(tx.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": tx.ID, "error": err.Error()})
		return err
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   tx.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para çekme işlemi: %.2f", tx.Amount),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
	}

	s.pendingTransactions.Delete(tx.ID)

	return nil
}

func (s *TransactionService) processTransfer(tx *domain.Transaction) error {
	fromUserID := *tx.FromUserID
	toUserID := *tx.ToUserID

	_, err := s.balanceSvc.WithdrawAtomically(fromUserID, tx.Amount)
	if err != nil {
		s.logger.Error("Transfer işlemi sırasında para çekme başarısız oldu", map[string]interface{}{
			"transaction_id": tx.ID,
			"from_user_id":   fromUserID,
			"error":          err.Error(),
		})
		s.repo.UpdateStatus(tx.ID, domain.TransactionStatusFailed)
		return err
	}

	_, err = s.balanceSvc.DepositAtomically(toUserID, tx.Amount)
	if err != nil {

		_, rollbackErr := s.balanceSvc.DepositAtomically(fromUserID, tx.Amount)
		if rollbackErr != nil {
			s.logger.Error("Geri alma işlemi başarısız oldu", map[string]interface{}{
				"transaction_id": tx.ID,
				"from_user_id":   fromUserID,
				"error":          rollbackErr.Error(),
			})
		}

		s.logger.Error("Transfer işlemi sırasında para yatırma başarısız oldu", map[string]interface{}{
			"transaction_id": tx.ID,
			"to_user_id":     toUserID,
			"error":          err.Error(),
		})
		s.repo.UpdateStatus(tx.ID, domain.TransactionStatusFailed)
		return err
	}

	if err := s.repo.UpdateStatus(tx.ID, domain.TransactionStatusCompleted); err != nil {
		s.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": tx.ID, "error": err.Error()})
		return err
	}

	auditLog := &domain.AuditLog{
		EntityType: domain.EntityTypeTransaction,
		EntityID:   tx.ID,
		Action:     domain.ActionTypeCreate,
		Details:    fmt.Sprintf("Para transferi: %.2f, %d -> %d", tx.Amount, fromUserID, toUserID),
		CreatedAt:  time.Now(),
	}

	if err := s.auditLogRepo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
	}

	s.pendingTransactions.Delete(tx.ID)

	return nil
}

func (s *TransactionService) GetWorkerPoolStats() (domain.TransactionStats, error) {
	s.ensureWorkerPoolInitialized()

	concurrentStats := s.workerPool.GetStats()
	stats := domain.TransactionStats{
		Submitted:      concurrentStats.Submitted,
		Completed:      concurrentStats.Completed,
		Failed:         concurrentStats.Failed,
		Rejected:       concurrentStats.Rejected,
		AvgProcessTime: concurrentStats.AvgProcessTime,
		QueueLength:    s.workerPool.QueueLength(),
		QueueCapacity:  s.workerPool.QueueCapacity(),
	}

	return stats, nil
}

func (s *TransactionService) ProcessBatchTransactions(transactions []*domain.Transaction) (processed int, failed int, err error) {
	s.ensureWorkerPoolInitialized()

	if len(transactions) == 0 {
		return 0, 0, nil
	}

	var wg sync.WaitGroup
	processMutex := sync.Mutex{}

	processedCount := 0
	failedCount := 0

	for _, tx := range transactions {
		wg.Add(1)

		go func(transaction *domain.Transaction) {
			defer wg.Done()

			var processErr error
			switch transaction.Type {
			case domain.TransactionTypeDeposit:
				processErr = s.processDeposit(transaction)
			case domain.TransactionTypeWithdraw:
				processErr = s.processWithdraw(transaction)
			case domain.TransactionTypeTransfer:
				processErr = s.processTransfer(transaction)
			default:
				processErr = fmt.Errorf("bilinmeyen işlem tipi: %s", transaction.Type)
			}

			processMutex.Lock()
			if processErr != nil {
				failedCount++
			} else {
				processedCount++
			}
			processMutex.Unlock()
		}(tx)
	}
	wg.Wait()

	return processedCount, failedCount, nil
}

func (s *TransactionService) Shutdown() {
	s.initMutex.Lock()
	defer s.initMutex.Unlock()

	if s.initialized {
		s.workerPool.Stop()
		s.initialized = false
	}
}
