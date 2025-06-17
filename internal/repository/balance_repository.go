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

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		r.logger.Error("Bakiye bulunamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
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

func (r *BalanceRepository) Update(balance *domain.Balance) (*domain.Balance, error) {
	query := `
		INSERT INTO balances (user_id, amount, last_updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET amount = $2, last_updated_at = $3
		RETURNING user_id, amount, last_updated_at
	`

	var updatedBalance domain.Balance
	err := r.db.QueryRow(
		query,
		balance.UserID,
		balance.Amount,
		balance.LastUpdatedAt,
	).Scan(
		&updatedBalance.UserID,
		&updatedBalance.Amount,
		&updatedBalance.LastUpdatedAt,
	)

	if err != nil {
		r.logger.Error("Bakiye güncellenemedi", map[string]interface{}{
			"user_id": balance.UserID,
			"error":   err.Error(),
		})
		return nil, err
	}

	return &updatedBalance, nil
}

func (r *BalanceRepository) InitializeBalance(userID int64) error {
	query := `
		INSERT INTO balances (user_id, amount, last_updated_at)
		VALUES ($1, 0, $2)
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err := r.db.Exec(query, userID, time.Now())
	if err != nil {
		r.logger.Error("Bakiye başlatılamadı", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return err
	}

	return nil
}

func (r *BalanceRepository) GetBalanceHistory(userID int64, startTime, endTime time.Time) ([]*domain.Balance, error) {
	query := `
		SELECT user_id, amount, last_updated_at
		FROM balance_history
		WHERE user_id = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []*domain.Balance
	for rows.Next() {
		var balance domain.Balance
		err := rows.Scan(
			&balance.UserID,
			&balance.Amount,
			&balance.LastUpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		balances = append(balances, &balance)
	}

	return balances, nil
}
