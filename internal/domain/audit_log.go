package domain

import "time"

type EntityType string
type ActionType string

const (
	EntityTypeUser        EntityType = "user"
	EntityTypeTransaction EntityType = "transaction"
	EntityTypeBalance     EntityType = "balance"

	ActionTypeCreate ActionType = "create"
	ActionTypeUpdate ActionType = "update"
	ActionTypeDelete ActionType = "delete"
)

type AuditLog struct {
	ID         int64      `json:"id"`
	EntityType EntityType `json:"entity_type"`
	EntityID   int64      `json:"entity_id"`
	Action     ActionType `json:"action"`
	Details    string     `json:"details,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AuditLogRepository interface {
	Create(log *AuditLog) error
	FindByEntityID(entityType EntityType, entityID int64) ([]*AuditLog, error)
	FindAll(limit, offset int) ([]*AuditLog, error)
}

type AuditLogService interface {
	LogAction(entityType EntityType, entityID int64, action ActionType, details string) error
	GetEntityLogs(entityType EntityType, entityID int64) ([]*AuditLog, error)
	GetAllLogs(page, pageSize int) ([]*AuditLog, error)
}
