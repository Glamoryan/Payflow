package repository

import (
	"database/sql"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type BalanceRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewBalanceRepository(db *sql.DB, logger logger.Logger) domain.BalanceRepository {
	return &BalanceRepository{
		db:     db,
		logger: logger,
	}
}

func (r *BalanceRepository) FindByUserID(userID int64) (*domain.Balance, error) {
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
