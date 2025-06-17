package domain

import "time"

type Balance struct {
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

type BalanceHistory struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Amount         float64   `json:"amount"`
	PreviousAmount float64   `json:"previous_amount"`
	TransactionID  int64     `json:"transaction_id"`
	Operation      string    `json:"operation"`
	CreatedAt      time.Time `json:"created_at"`
}

type BalanceRepository interface {
	FindByUserID(userID int64) (*Balance, error)
	Create(balance *Balance) error
	Update(balance *Balance) (*Balance, error)
	InitializeBalance(userID int64) error
	GetBalanceHistory(userID int64, startTime, endTime time.Time) ([]*Balance, error)
}

type BalanceService interface {
	GetBalance(userID int64) (*Balance, error)
	DepositAtomically(userID int64, amount float64) (*Balance, error)
	WithdrawAtomically(userID int64, amount float64) (*Balance, error)
	InitializeBalance(userID int64) error
	GetBalanceHistory(userID int64, startTime, endTime time.Time) ([]*Balance, error)
	ReplayBalanceEvents(userID int64) error
	RebuildBalanceState(userID int64) error
}
