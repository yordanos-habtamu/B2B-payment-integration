package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type EventPublisher struct {
	redisClient *redis.Client
	prefix      string
}

type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	TenantID  string                 `json:"tenant_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
}

type EventMetadata struct {
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
	EventType     string `json:"event_type"`
	Source       string `json:"source"`
	Timestamp    time.Time `json:"timestamp"`
}

func NewEventPublisher(redisClient *redis.Client, prefix string) *EventPublisher {
	if prefix == "" {
		prefix = "b2b_payments"
	}
	
	return &EventPublisher{
		redisClient: redisClient,
		prefix:      prefix,
	}
}

func (p *EventPublisher) Publish(ctx context.Context, eventType string, tenantID string, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "b2b-payments-api",
		TenantID:  tenantID,
		Data:      data,
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	return p.publishEvent(ctx, event)
}

func (p *EventPublisher) PublishWithMetadata(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Version == "" {
		event.Version = "1.0"
	}

	return p.publishEvent(ctx, event)
}

func (p *EventPublisher) publishEvent(ctx context.Context, event *Event) error {
	// Serialize event
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine channel name
	channel := fmt.Sprintf("%s.events.%s", p.prefix, event.Type)

	// Publish to Redis
	if err := p.redisClient.Publish(ctx, channel, eventJSON).Err(); err != nil {
		return fmt.Errorf("failed to publish event to channel %s: %w", channel, err)
	}

	// Also store in event log for replay/audit
	eventLogKey := fmt.Sprintf("%s.event_log:%s", p.prefix, event.TenantID)
	if err := p.redisClient.LPush(ctx, eventLogKey, eventJSON).Err(); err != nil {
		log.Printf("Failed to store event in log: %v", err)
	}

	// Trim event log to prevent unlimited growth
	if err := p.redisClient.LTrim(ctx, eventLogKey, 0, 9999).Err(); err != nil {
		log.Printf("Failed to trim event log: %v", err)
	}

	log.Printf("Published event %s (type: %s, tenant: %s)", event.ID, event.Type, event.TenantID)
	return nil
}

func (p *EventPublisher) PublishPaymentCreated(ctx context.Context, tenantID, paymentID string, amount float64, currency string) error {
	data := map[string]interface{}{
		"payment_id": paymentID,
		"amount":     amount,
		"currency":   currency,
		"action":     "created",
	}

	return p.Publish(ctx, "payment.created", tenantID, data)
}

func (p *EventPublisher) PublishPaymentUpdated(ctx context.Context, tenantID, paymentID string, changes map[string]interface{}) error {
	data := map[string]interface{}{
		"payment_id": paymentID,
		"changes":    changes,
		"action":     "updated",
	}

	return p.Publish(ctx, "payment.updated", tenantID, data)
}

func (p *EventPublisher) PublishPaymentStatusChanged(ctx context.Context, tenantID, paymentID, oldStatus, newStatus string) error {
	data := map[string]interface{}{
		"payment_id":  paymentID,
		"old_status":  oldStatus,
		"new_status":  newStatus,
		"action":      "status_changed",
	}

	return p.Publish(ctx, "payment.status_changed", tenantID, data)
}

func (p *EventPublisher) PublishPaymentProcessed(ctx context.Context, tenantID, paymentID string, processedAt time.Time) error {
	data := map[string]interface{}{
		"payment_id":   paymentID,
		"processed_at": processedAt,
		"action":       "processed",
	}

	return p.Publish(ctx, "payment.processed", tenantID, data)
}

func (p *EventPublisher) PublishPaymentCompleted(ctx context.Context, tenantID, paymentID string, completedAt time.Time) error {
	data := map[string]interface{}{
		"payment_id":   paymentID,
		"completed_at": completedAt,
		"action":       "completed",
	}

	return p.Publish(ctx, "payment.completed", tenantID, data)
}

func (p *EventPublisher) PublishPaymentFailed(ctx context.Context, tenantID, paymentID, reason string, failedAt time.Time) error {
	data := map[string]interface{}{
		"payment_id":  paymentID,
		"failure_reason": reason,
		"failed_at":    failedAt,
		"action":       "failed",
	}

	return p.Publish(ctx, "payment.failed", tenantID, data)
}

func (p *EventPublisher) PublishPaymentCancelled(ctx context.Context, tenantID, paymentID string, cancelledAt time.Time) error {
	data := map[string]interface{}{
		"payment_id":   paymentID,
		"cancelled_at": cancelledAt,
		"action":       "cancelled",
	}

	return p.Publish(ctx, "payment.cancelled", tenantID, data)
}

func (p *EventPublisher) PublishTenantCreated(ctx context.Context, tenantID, name string) error {
	data := map[string]interface{}{
		"tenant_id": tenantID,
		"name":      name,
		"action":    "created",
	}

	return p.Publish(ctx, "tenant.created", tenantID, data)
}

func (p *EventPublisher) PublishTenantUpdated(ctx context.Context, tenantID string, changes map[string]interface{}) error {
	data := map[string]interface{}{
		"tenant_id": tenantID,
		"changes":   changes,
		"action":    "updated",
	}

	return p.Publish(ctx, "tenant.updated", tenantID, data)
}

func (p *EventPublisher) GetEventLog(ctx context.Context, tenantID string, limit int64) ([]*Event, error) {
	eventLogKey := fmt.Sprintf("%s.event_log:%s", p.prefix, tenantID)
	
	results, err := p.redisClient.LRange(ctx, eventLogKey, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event log: %w", err)
	}

	var events []*Event
	for _, result := range results {
		var event Event
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			log.Printf("Failed to unmarshal event from log: %v", err)
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
