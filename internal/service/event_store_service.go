package service

import (
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type EventStoreService struct {
	repo   domain.EventStoreRepository
	logger logger.Logger
}

func NewEventStoreService(repo domain.EventStoreRepository, logger logger.Logger) domain.EventStoreService {
	return &EventStoreService{
		repo:   repo,
		logger: logger,
	}
}

func (s *EventStoreService) SaveEvent(event *domain.Event) error {
	lastVersion, err := s.repo.GetLastVersion(event.AggregateType, event.AggregateID)
	if err != nil {
		s.logger.Error("Son versiyon alınamadı", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		return err
	}

	if event.Version != lastVersion+1 {
		s.logger.Error("Versiyon uyumsuzluğu", map[string]interface{}{
			"expected_version": lastVersion + 1,
			"actual_version":   event.Version,
			"event":            event,
		})
		return domain.ErrConcurrentModification
	}

	if err := s.repo.Save(event); err != nil {
		s.logger.Error("Event kaydedilemedi", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		return err
	}

	return nil
}

func (s *EventStoreService) GetAggregateEvents(aggregateType string, aggregateID string) ([]*domain.Event, error) {
	events, err := s.repo.GetEvents(aggregateType, aggregateID)
	if err != nil {
		s.logger.Error("Aggregate eventleri alınamadı", map[string]interface{}{
			"error":         err.Error(),
			"aggregateType": aggregateType,
			"aggregateID":   aggregateID,
		})
		return nil, err
	}

	return events, nil
}

func (s *EventStoreService) GetEventsByType(eventType domain.EventType) ([]*domain.Event, error) {
	events, err := s.repo.GetEventsByType(eventType)
	if err != nil {
		s.logger.Error("Event tipine göre eventler alınamadı", map[string]interface{}{
			"error":     err.Error(),
			"eventType": eventType,
		})
		return nil, err
	}

	return events, nil
}

func (s *EventStoreService) GetEventsByTimeRange(startTime, endTime time.Time) ([]*domain.Event, error) {
	events, err := s.repo.GetEventsByTimeRange(startTime, endTime)
	if err != nil {
		s.logger.Error("Zaman aralığına göre eventler alınamadı", map[string]interface{}{
			"error":     err.Error(),
			"startTime": startTime,
			"endTime":   endTime,
		})
		return nil, err
	}

	return events, nil
}

func (s *EventStoreService) ReplayEvents(aggregateType string, aggregateID string, handler func(*domain.Event) error) error {
	events, err := s.GetAggregateEvents(aggregateType, aggregateID)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := handler(event); err != nil {
			s.logger.Error("Event replay edilemedi", map[string]interface{}{
				"error": err.Error(),
				"event": event,
			})
			return err
		}
	}

	return nil
}

func (s *EventStoreService) GetLastVersion(aggregateType string, aggregateID string) (int, error) {
	return s.repo.GetLastVersion(aggregateType, aggregateID)
}
