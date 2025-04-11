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
	Update(balance *Balance) error

	AtomicUpdate(userID int64, updateFn func(currentAmount float64) float64) (float64, error)

	AddBalanceHistory(history *BalanceHistory) error
	GetBalanceHistory(userID int64, limit, offset int) ([]*BalanceHistory, error)
	GetBalanceHistoryByDateRange(userID int64, startDate, endDate time.Time) ([]*BalanceHistory, error)
}

type BalanceService interface {
	GetUserBalance(userID int64) (*Balance, error)
	InitializeBalance(userID int64) error
	UpdateBalance(userID int64, amount float64) error

	DepositAtomically(userID int64, amount float64) (*Balance, error)
	WithdrawAtomically(userID int64, amount float64) (*Balance, error)

	GetBalanceHistory(userID int64, limit, offset int) ([]*BalanceHistory, error)
	GetBalanceHistoryByDateRange(userID int64, startDate, endDate time.Time) ([]*BalanceHistory, error)

	GetCachedBalance(userID int64) (*Balance, error)
	RecalculateBalance(userID int64) (*Balance, error)
}
