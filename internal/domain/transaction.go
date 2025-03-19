package domain

import (
	"time"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeTransfer TransactionType = "transfer"

	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

type Transaction struct {
	ID         int64             `json:"id"`
	FromUserID *int64            `json:"from_user_id,omitempty"`
	ToUserID   *int64            `json:"to_user_id,omitempty"`
	Amount     float64           `json:"amount"`
	Type       TransactionType   `json:"type"`
	Status     TransactionStatus `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
}

type TransactionRepository interface {
	FindByID(id int64) (*Transaction, error)
	FindByUserID(userID int64) ([]*Transaction, error)
	Create(transaction *Transaction) error
	UpdateStatus(id int64, status TransactionStatus) error
}

type TransactionService interface {
	GetTransactionByID(id int64) (*Transaction, error)
	GetUserTransactions(userID int64) ([]*Transaction, error)
	DepositFunds(userID int64, amount float64) (*Transaction, error)
	WithdrawFunds(userID int64, amount float64) (*Transaction, error)
	TransferFunds(fromUserID, toUserID int64, amount float64) (*Transaction, error)
}
