package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type EventStoreRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewEventStoreRepository(db *sql.DB, logger logger.Logger) domain.EventStoreRepository {
	return &EventStoreRepository{
		db:     db,
		logger: logger,
	}
}

func (r *EventStoreRepository) Save(event *domain.Event) error {
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		r.logger.Error("Event data JSON'a çevrilemedi", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		return err
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		r.logger.Error("Metadata JSON'a çevrilemedi", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		return err
	}

	query := `
		INSERT INTO event_store (
			aggregate_id, aggregate_type, event_type, event_data, version, created_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err = r.db.QueryRow(
		query,
		event.AggregateID,
		event.AggregateType,
		event.EventType,
		eventDataJSON,
		event.Version,
		event.CreatedAt,
		metadataJSON,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Event kaydedilemedi", map[string]interface{}{
			"error": err.Error(),
			"event": event,
		})
		return err
	}

	event.ID = id
	return nil
}

func (r *EventStoreRepository) GetEvents(aggregateType string, aggregateID string) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, event_data, version, created_at, metadata
		FROM event_store
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC
	`

	rows, err := r.db.Query(query, aggregateType, aggregateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var eventData, metadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.AggregateType,
			&event.EventType,
			&eventData,
			&event.Version,
			&event.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, err
		}

		event.EventData = eventData
		event.Metadata = metadata
		events = append(events, event)
	}

	return events, nil
}

func (r *EventStoreRepository) GetEventsByType(eventType domain.EventType) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, event_data, version, created_at, metadata
		FROM event_store
		WHERE event_type = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var eventData, metadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.AggregateType,
			&event.EventType,
			&eventData,
			&event.Version,
			&event.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, err
		}

		event.EventData = eventData
		event.Metadata = metadata
		events = append(events, event)
	}

	return events, nil
}

func (r *EventStoreRepository) GetEventsByTimeRange(startTime, endTime time.Time) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, event_data, version, created_at, metadata
		FROM event_store
		WHERE created_at BETWEEN $1 AND $2
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var eventData, metadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.AggregateType,
			&event.EventType,
			&eventData,
			&event.Version,
			&event.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, err
		}

		event.EventData = eventData
		event.Metadata = metadata
		events = append(events, event)
	}

	return events, nil
}

func (r *EventStoreRepository) GetLastVersion(aggregateType string, aggregateID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM event_store
		WHERE aggregate_type = $1 AND aggregate_id = $2
	`

	var version int
	err := r.db.QueryRow(query, aggregateType, aggregateID).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}
