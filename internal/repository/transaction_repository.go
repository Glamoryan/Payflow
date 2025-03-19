package repository

import (
	"database/sql"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewTransactionRepository(db *sql.DB, logger logger.Logger) domain.TransactionRepository {
	return &TransactionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *TransactionRepository) FindByID(id int64) (*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, created_at
		FROM transactions
		WHERE id = $1
	`

	var transaction domain.Transaction
	var fromUserID, toUserID sql.NullInt64
	var transactionType, status string

	err := r.db.QueryRow(query, id).Scan(
		&transaction.ID,
		&fromUserID,
		&toUserID,
		&transaction.Amount,
		&transactionType,
		&status,
		&transaction.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("İşlem ID'ye göre bulunamadı", map[string]interface{}{"id": id, "error": err.Error()})
		return nil, fmt.Errorf("işlem bulunamadı: %w", err)
	}

	if fromUserID.Valid {
		fuid := fromUserID.Int64
		transaction.FromUserID = &fuid
	}

	if toUserID.Valid {
		tuid := toUserID.Int64
		transaction.ToUserID = &tuid
	}

	transaction.Type = domain.TransactionType(transactionType)
	transaction.Status = domain.TransactionStatus(status)

	return &transaction, nil
}

func (r *TransactionRepository) FindByUserID(userID int64) ([]*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, created_at
		FROM transactions
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		r.logger.Error("Kullanıcı işlemleri bulunamadı", map[string]interface{}{"user_id": userID, "error": err.Error()})
		return nil, fmt.Errorf("kullanıcı işlemleri bulunamadı: %w", err)
	}
	defer rows.Close()

	transactions := make([]*domain.Transaction, 0)
	for rows.Next() {
		var transaction domain.Transaction
		var fromUserID, toUserID sql.NullInt64
		var transactionType, status string

		err := rows.Scan(
			&transaction.ID,
			&fromUserID,
			&toUserID,
			&transaction.Amount,
			&transactionType,
			&status,
			&transaction.CreatedAt,
		)
		if err != nil {
			r.logger.Error("İşlem verileri okunamadı", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("işlem verileri okunamadı: %w", err)
		}

		if fromUserID.Valid {
			fuid := fromUserID.Int64
			transaction.FromUserID = &fuid
		}

		if toUserID.Valid {
			tuid := toUserID.Int64
			transaction.ToUserID = &tuid
		}

		transaction.Type = domain.TransactionType(transactionType)
		transaction.Status = domain.TransactionStatus(status)

		transactions = append(transactions, &transaction)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Satır döngüsü sırasında hata oluştu", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("işlem verileri okunamadı: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) Create(transaction *domain.Transaction) error {
	query := `
		INSERT INTO transactions (from_user_id, to_user_id, amount, type, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var fromUserID, toUserID interface{}
	if transaction.FromUserID != nil {
		fromUserID = *transaction.FromUserID
	} else {
		fromUserID = nil
	}

	if transaction.ToUserID != nil {
		toUserID = *transaction.ToUserID
	} else {
		toUserID = nil
	}

	transaction.CreatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		fromUserID,
		toUserID,
		transaction.Amount,
		string(transaction.Type),
		string(transaction.Status),
		transaction.CreatedAt,
	).Scan(&transaction.ID)

	if err != nil {
		r.logger.Error("İşlem oluşturulamadı", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("işlem oluşturulamadı: %w", err)
	}

	return nil
}

func (r *TransactionRepository) UpdateStatus(id int64, status domain.TransactionStatus) error {
	query := `
		UPDATE transactions
		SET status = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, string(status), id)
	if err != nil {
		r.logger.Error("İşlem durumu güncellenemedi", map[string]interface{}{"id": id, "error": err.Error()})
		return fmt.Errorf("işlem durumu güncellenemedi: %w", err)
	}

	return nil
}
