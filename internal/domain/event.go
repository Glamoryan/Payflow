package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventTypeTransactionCreated   EventType = "transaction_created"
	EventTypeTransactionCompleted EventType = "transaction_completed"
	EventTypeTransactionFailed    EventType = "transaction_failed"
	EventTypeBalanceUpdated       EventType = "balance_updated"
	EventTypeUserCreated          EventType = "user_created"
)

type Event struct {
	ID            int64           `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	EventType     EventType       `json:"event_type"`
	EventData     json.RawMessage `json:"event_data"`
	Version       int             `json:"version"`
	CreatedAt     time.Time       `json:"created_at"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type EventStoreRepository interface {
	Save(event *Event) error
	GetEvents(aggregateType string, aggregateID string) ([]*Event, error)
	GetEventsByType(eventType EventType) ([]*Event, error)
	GetEventsByTimeRange(startTime, endTime time.Time) ([]*Event, error)
	GetLastVersion(aggregateType string, aggregateID string) (int, error)
}

type EventStoreService interface {
	SaveEvent(event *Event) error
	GetAggregateEvents(aggregateType string, aggregateID string) ([]*Event, error)
	GetEventsByType(eventType EventType) ([]*Event, error)
	GetEventsByTimeRange(startTime, endTime time.Time) ([]*Event, error)
	ReplayEvents(aggregateType string, aggregateID string, handler func(*Event) error) error
	GetLastVersion(aggregateType string, aggregateID string) (int, error)
}
