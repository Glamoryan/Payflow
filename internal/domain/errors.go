package domain

import "errors"

var (
	ErrConcurrentModification = errors.New("eşzamanlı değişiklik tespit edildi")
	ErrInsufficientFunds      = errors.New("yetersiz bakiye")
	ErrInvalidAmount          = errors.New("geçersiz miktar")
	ErrInvalidTransaction     = errors.New("geçersiz işlem")
	ErrUserNotFound           = errors.New("kullanıcı bulunamadı")
	ErrTransactionNotFound    = errors.New("işlem bulunamadı")
	ErrBalanceNotFound        = errors.New("bakiye bulunamadı")
)
