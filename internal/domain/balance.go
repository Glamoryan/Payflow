package domain

import "time"

type Balance struct {
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

type BalanceRepository interface {
	FindByUserID(userID int64) (*Balance, error)
	Create(balance *Balance) error
	Update(balance *Balance) error
}

type BalanceService interface {
	GetUserBalance(userID int64) (*Balance, error)
	InitializeBalance(userID int64) error
	UpdateBalance(userID int64, amount float64) error
}
