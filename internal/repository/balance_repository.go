package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type BalanceRepository struct {
	db     *sql.DB
	logger logger.Logger
	mutex  sync.RWMutex
}

func NewBalanceRepository(db *sql.DB, logger logger.Logger) domain.BalanceRepository {
	return &BalanceRepository{
		db:     db,
		logger: logger,
	}
}

func (r *BalanceRepository) FindByUserID(userID int64) (*domain.Balance, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	query := `
		SELECT user_id, amount, last_updated_at
		FROM balances
		WHERE user_id = $1
	`

	var balance domain.Balance
	err := r.db.QueryRow(query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Bakiye bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("bakiye bulunamadı: %w", err)
	}

	return &balance, nil
}

func (r *BalanceRepository) Create(balance *domain.Balance) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query := `
		INSERT INTO balances (user_id, amount, last_updated_at)
		VALUES ($1, $2, $3)
	`

	balance.LastUpdatedAt = time.Now()

	_, err := r.db.Exec(
		query,
		balance.UserID,
		balance.Amount,
		balance.LastUpdatedAt,
	)

	if err != nil {
		r.logger.Error("Bakiye oluşturulamadı", map[string]interface{}{"user_id": balance.UserID, "error": err.Error()})
		return fmt.Errorf("bakiye oluşturulamadı: %w", err)
	}

	return nil
}

func (r *BalanceRepository) Update(balance *domain.Balance) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query := `
		UPDATE balances
		SET amount = $1, last_updated_at = $2
		WHERE user_id = $3
	`

	balance.LastUpdatedAt = time.Now()

	result, err := r.db.Exec(
		query,
		balance.Amount,
		balance.LastUpdatedAt,
		balance.UserID,
	)

	if err != nil {
		r.logger.Error("Bakiye güncellenemedi", map[string]interface{}{"user_id": balance.UserID, "error": err.Error()})
		return fmt.Errorf("bakiye güncellenemedi: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Etkilenen satır sayısı alınamadı", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("bakiye güncellenemedi: %w", err)
	}

	if rows == 0 {
		r.logger.Error("Bakiye güncellenemedi, kullanıcı bulunamadı", map[string]interface{}{"user_id": balance.UserID})
		return fmt.Errorf("bakiye güncellenemedi, kullanıcı bulunamadı: %d", balance.UserID)
	}

	return nil
}

func (r *BalanceRepository) AtomicUpdate(userID int64, updateFn func(currentAmount float64) float64) (float64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	balance, err := r.FindByUserIDNoLock(userID)
	if err != nil {
		return 0, err
	}

	if balance == nil {
		return 0, fmt.Errorf("kullanıcının bakiyesi bulunamadı: %d", userID)
	}

	newAmount := updateFn(balance.Amount)

	query := `UPDATE balances SET amount = ?, last_updated_at = ? WHERE user_id = ?`

	now := time.Now()
	lastUpdatedAt := now.Format(time.RFC3339)

	result, err := r.db.Exec(query, newAmount, lastUpdatedAt, userID)
	if err != nil {
		r.logger.Error("Bakiye atomik olarak güncellenirken hata oluştu", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return 0, fmt.Errorf("bakiye atomik olarak güncellenirken hata oluştu: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return 0, fmt.Errorf("bakiye güncellenemedi, kullanıcı bulunamadı: %d", userID)
	}

	return newAmount, nil
}

func (r *BalanceRepository) FindByUserIDNoLock(userID int64) (*domain.Balance, error) {
	query := `
		SELECT user_id, amount, last_updated_at
		FROM balances
		WHERE user_id = $1
	`

	var balance domain.Balance
	err := r.db.QueryRow(query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("bakiye sorgulanırken hata oluştu: %w", err)
	}

	return &balance, nil
}
