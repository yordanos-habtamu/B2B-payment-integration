package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

type EventHandler func(ctx context.Context, event *Event) error

type EventSubscriber struct {
	redisClient   *redis.Client
	prefix        string
	handlers      map[string][]EventHandler
	pubsub        *redis.PubSub
	subscriptions []string
}

func NewEventSubscriber(redisClient *redis.Client, prefix string) *EventSubscriber {
	if prefix == "" {
		prefix = "b2b_payments"
	}

	return &EventSubscriber{
		redisClient: redisClient,
		prefix:      prefix,
		handlers:     make(map[string][]EventHandler),
	}
}

func (s *EventSubscriber) Subscribe(eventType string, handler EventHandler) {
	if s.handlers[eventType] == nil {
		s.handlers[eventType] = make([]EventHandler, 0)
	}
	s.handlers[eventType] = append(s.handlers[eventType], handler)
}

func (s *EventSubscriber) SubscribeToAll(handler EventHandler) {
	// This handler will be called for all events
	s.Subscribe("*", handler)
}

func (s *EventSubscriber) Start(ctx context.Context) error {
	// Build list of channels to subscribe to
	var channels []string
	for eventType := range s.handlers {
		channel := fmt.Sprintf("%s.events.%s", s.prefix, eventType)
		channels = append(channels, channel)
	}

	// Subscribe to all channels
	s.pubsub = s.redisClient.Subscribe(ctx, channels...)
	if err := s.pubsub.Ping(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to event channels: %w", err)
	}

	log.Printf("Subscribed to %d event channels", len(channels))

	// Start listening for events
	go s.listen(ctx)

	return nil
}

func (s *EventSubscriber) Stop() error {
	if s.pubsub != nil {
		return s.pubsub.Close()
	}
	return nil
}

func (s *EventSubscriber) listen(ctx context.Context) {
	ch := s.pubsub.Channel()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Event subscriber shutting down...")
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			
			if err := s.handleMessage(ctx, msg); err != nil {
				log.Printf("Error handling event message: %v", err)
			}
		}
	}
}

func (s *EventSubscriber) handleMessage(ctx context.Context, msg *redis.Message) error {
	// Extract event type from channel name
	eventType := s.extractEventType(msg.Channel)
	if eventType == "" {
		return fmt.Errorf("could not extract event type from channel: %s", msg.Channel)
	}

	// Parse event
	var event Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Find handlers for this event type
	handlers := s.getHandlersForEvent(eventType)
	if len(handlers) == 0 {
		log.Printf("No handlers found for event type: %s", eventType)
		return nil
	}

	// Execute all handlers
	for i, handler := range handlers {
		if err := handler(ctx, &event); err != nil {
			log.Printf("Handler %d for event %s failed: %v", i, eventType, err)
			// Continue with other handlers even if one fails
		}
	}

	return nil
}

func (s *EventSubscriber) extractEventType(channel string) string {
	prefix := fmt.Sprintf("%s.events.", s.prefix)
	if strings.HasPrefix(channel, prefix) {
		return strings.TrimPrefix(channel, prefix)
	}
	return ""
}

func (s *EventSubscriber) getHandlersForEvent(eventType string) []EventHandler {
	var handlers []EventHandler

	// Add specific event handlers
	if eventHandlers, exists := s.handlers[eventType]; exists {
		handlers = append(handlers, eventHandlers...)
	}

	// Add wildcard handlers (for all events)
	if wildcardHandlers, exists := s.handlers["*"]; exists {
		handlers = append(handlers, wildcardHandlers...)
	}

	return handlers
}

// Convenience methods for common event handlers

func (s *EventSubscriber) SubscribeToPaymentEvents(handler EventHandler) {
	s.Subscribe("payment.created", handler)
	s.Subscribe("payment.updated", handler)
	s.Subscribe("payment.status_changed", handler)
	s.Subscribe("payment.processed", handler)
	s.Subscribe("payment.completed", handler)
	s.Subscribe("payment.failed", handler)
	s.Subscribe("payment.cancelled", handler)
}

func (s *EventSubscriber) SubscribeToTenantEvents(handler EventHandler) {
	s.Subscribe("tenant.created", handler)
	s.Subscribe("tenant.updated", handler)
}

// Event handler utilities

func LogEventHandler(ctx context.Context, event *Event) error {
	log.Printf("Event received: %s (type: %s, tenant: %s)", 
		event.ID, event.Type, event.TenantID)
	return nil
}

func AuditEventHandler(ctx context.Context, event *Event) error {
	// In a real implementation, this would store events in an audit log
	// For now, we'll just log them
	log.Printf("Audit: Event %s of type %s for tenant %s at %s", 
		event.ID, event.Type, event.TenantID, event.Timestamp)
	return nil
}

func MetricsEventHandler(ctx context.Context, event *Event) error {
	// In a real implementation, this would update metrics
	// For now, we'll just log them
	log.Printf("Metrics: Event %s type %s for tenant %s", 
		event.ID, event.Type, event.TenantID)
	return nil
}

// Event filter utilities

func CreateTenantFilter(tenantID string) EventHandler {
	return func(ctx context.Context, event *Event) error {
		if event.TenantID != tenantID {
			return nil // Skip events for other tenants
		}
		return nil
	}
}

func CreateEventTypeFilter(eventTypes ...string) EventHandler {
	return func(ctx context.Context, event *Event) error {
		for _, eventType := range eventTypes {
			if event.Type == eventType {
				return nil
			}
		}
		return nil // Skip if event type not in allowed list
	}
}

func CreateChainedHandler(handlers ...EventHandler) EventHandler {
	return func(ctx context.Context, event *Event) error {
		for _, handler := range handlers {
			if err := handler(ctx, event); err != nil {
				return err
			}
		}
		return nil
	}
}
