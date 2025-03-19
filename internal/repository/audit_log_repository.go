package repository

import (
	"database/sql"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type AuditLogRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewAuditLogRepository(db *sql.DB, logger logger.Logger) domain.AuditLogRepository {
	return &AuditLogRepository{
		db:     db,
		logger: logger,
	}
}

func (r *AuditLogRepository) Create(log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (entity_type, entity_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	log.CreatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		string(log.EntityType),
		log.EntityID,
		string(log.Action),
		log.Details,
		log.CreatedAt,
	).Scan(&log.ID)

	if err != nil {
		r.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("denetim kaydı oluşturulamadı: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) FindByEntityID(entityType domain.EntityType, entityID int64) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, created_at
		FROM audit_logs
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, string(entityType), entityID)
	if err != nil {
		r.logger.Error("Denetim kayıtları bulunamadı", map[string]interface{}{
			"entity_type": entityType,
			"entity_id":   entityID,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("denetim kayıtları bulunamadı: %w", err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		var log domain.AuditLog
		var entityTypeStr, actionStr string

		err := rows.Scan(
			&log.ID,
			&entityTypeStr,
			&log.EntityID,
			&actionStr,
			&log.Details,
			&log.CreatedAt,
		)
		if err != nil {
			r.logger.Error("Denetim kaydı verileri okunamadı", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("denetim kaydı verileri okunamadı: %w", err)
		}

		log.EntityType = domain.EntityType(entityTypeStr)
		log.Action = domain.ActionType(actionStr)

		logs = append(logs, &log)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Satır döngüsü sırasında hata oluştu", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("denetim kaydı verileri okunamadı: %w", err)
	}

	return logs, nil
}

func (r *AuditLogRepository) FindAll(limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		r.logger.Error("Denetim kayıtları bulunamadı", map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("denetim kayıtları bulunamadı: %w", err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		var log domain.AuditLog
		var entityTypeStr, actionStr string

		err := rows.Scan(
			&log.ID,
			&entityTypeStr,
			&log.EntityID,
			&actionStr,
			&log.Details,
			&log.CreatedAt,
		)
		if err != nil {
			r.logger.Error("Denetim kaydı verileri okunamadı", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("denetim kaydı verileri okunamadı: %w", err)
		}

		log.EntityType = domain.EntityType(entityTypeStr)
		log.Action = domain.ActionType(actionStr)

		logs = append(logs, &log)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Satır döngüsü sırasında hata oluştu", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("denetim kaydı verileri okunamadı: %w", err)
	}

	return logs, nil
}
