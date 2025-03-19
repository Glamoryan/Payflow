package service

import (
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type AuditLogService struct {
	repo   domain.AuditLogRepository
	logger logger.Logger
}

func NewAuditLogService(repo domain.AuditLogRepository, logger logger.Logger) domain.AuditLogService {
	return &AuditLogService{
		repo:   repo,
		logger: logger,
	}
}

func (s *AuditLogService) LogAction(entityType domain.EntityType, entityID int64, action domain.ActionType, details string) error {
	auditLog := &domain.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Details:    details,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(auditLog); err != nil {
		s.logger.Error("Denetim kaydı oluşturulamadı", map[string]interface{}{
			"entity_type": entityType,
			"entity_id":   entityID,
			"action":      action,
			"error":       err.Error(),
		})
		return fmt.Errorf("denetim kaydı oluşturulamadı: %w", err)
	}

	return nil
}

func (s *AuditLogService) GetEntityLogs(entityType domain.EntityType, entityID int64) ([]*domain.AuditLog, error) {
	logs, err := s.repo.FindByEntityID(entityType, entityID)
	if err != nil {
		s.logger.Error("Denetim kayıtları bulunamadı", map[string]interface{}{
			"entity_type": entityType,
			"entity_id":   entityID,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("denetim kayıtları bulunamadı: %w", err)
	}

	return logs, nil
}

func (s *AuditLogService) GetAllLogs(page, pageSize int) ([]*domain.AuditLog, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	limit := pageSize

	logs, err := s.repo.FindAll(limit, offset)
	if err != nil {
		s.logger.Error("Denetim kayıtları bulunamadı", map[string]interface{}{
			"page":      page,
			"page_size": pageSize,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("denetim kayıtları bulunamadı: %w", err)
	}

	return logs, nil
}
