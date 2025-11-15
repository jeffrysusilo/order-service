package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"order-service/internal/models"

	"github.com/segmentio/kafka-go"
)

// EventPublisher handles publishing domain events
type EventPublisher struct {
	producer *Producer
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(producer *Producer) *EventPublisher {
	return &EventPublisher{producer: producer}
}

// PublishOrderCreated publishes OrderCreated event
func (ep *EventPublisher) PublishOrderCreated(ctx context.Context, event *models.OrderCreatedEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// PublishOrderReserved publishes OrderReserved event
func (ep *EventPublisher) PublishOrderReserved(ctx context.Context, event *models.OrderReservedEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// PublishOrderPaid publishes OrderPaid event
func (ep *EventPublisher) PublishOrderPaid(ctx context.Context, event *models.OrderPaidEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// PublishOrderCancelled publishes OrderCancelled event
func (ep *EventPublisher) PublishOrderCancelled(ctx context.Context, event *models.OrderCancelledEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// PublishPaymentSuccess publishes PaymentSuccess event
func (ep *EventPublisher) PublishPaymentSuccess(ctx context.Context, event *models.PaymentSuccessEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// PublishPaymentFailed publishes PaymentFailed event
func (ep *EventPublisher) PublishPaymentFailed(ctx context.Context, event *models.PaymentFailedEvent) error {
	key := fmt.Sprintf("order-%d", event.OrderID)
	return ep.producer.PublishEvent(ctx, key, event)
}

// EventHandler handles incoming events
type EventHandler struct {
	onPaymentSuccess func(context.Context, *models.PaymentSuccessEvent) error
	onPaymentFailed  func(context.Context, *models.PaymentFailedEvent) error
}

// NewEventHandler creates a new event handler
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// OnPaymentSuccess registers a handler for PaymentSuccess events
func (eh *EventHandler) OnPaymentSuccess(handler func(context.Context, *models.PaymentSuccessEvent) error) {
	eh.onPaymentSuccess = handler
}

// OnPaymentFailed registers a handler for PaymentFailed events
func (eh *EventHandler) OnPaymentFailed(handler func(context.Context, *models.PaymentFailedEvent) error) {
	eh.onPaymentFailed = handler
}

// HandleMessage routes messages to appropriate handlers
func (eh *EventHandler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	var baseEvent models.BaseEvent
	if err := json.Unmarshal(msg.Value, &baseEvent); err != nil {
		return fmt.Errorf("failed to unmarshal base event: %w", err)
	}

	log.Printf("Handling event: type=%s, id=%s", baseEvent.EventType, baseEvent.EventID)

	switch baseEvent.EventType {
	case models.EventTypePaymentSuccess:
		if eh.onPaymentSuccess != nil {
			var event models.PaymentSuccessEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return fmt.Errorf("failed to unmarshal PaymentSuccess event: %w", err)
			}
			return eh.onPaymentSuccess(ctx, &event)
		}

	case models.EventTypePaymentFailed:
		if eh.onPaymentFailed != nil {
			var event models.PaymentFailedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return fmt.Errorf("failed to unmarshal PaymentFailed event: %w", err)
			}
			return eh.onPaymentFailed(ctx, &event)
		}

	default:
		log.Printf("Unhandled event type: %s", baseEvent.EventType)
	}

	return nil
}
