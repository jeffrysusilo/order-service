package worker

import (
	"context"
	"encoding/json"
	"log"

	"order-service/internal/broker"
	"order-service/internal/models"
	"order-service/internal/service"

	"github.com/segmentio/kafka-go"
)

// OrderWorker handles background processing for order events
type OrderWorker struct {
	consumer         *broker.Consumer
	eventHandler     *broker.EventHandler
	sagaOrchestrator *service.SagaOrchestrator
}

// NewOrderWorker creates a new order worker
func NewOrderWorker(
	consumer *broker.Consumer,
	sagaOrchestrator *service.SagaOrchestrator,
) *OrderWorker {
	eventHandler := broker.NewEventHandler()

	eventHandler.OnPaymentSuccess(sagaOrchestrator.HandlePaymentSuccess)
	eventHandler.OnPaymentFailed(sagaOrchestrator.HandlePaymentFailed)

	return &OrderWorker{
		consumer:         consumer,
		eventHandler:     eventHandler,
		sagaOrchestrator: sagaOrchestrator,
	}
}

// Start starts the worker
func (w *OrderWorker) Start(ctx context.Context) error {
	log.Println("Starting order worker...")
	return w.consumer.StartConsuming(ctx, w.eventHandler.HandleMessage)
}

// Stop stops the worker
func (w *OrderWorker) Stop() error {
	log.Println("Stopping order worker...")
	return w.consumer.Close()
}

// PaymentWorker handles payment processing
type PaymentWorker struct {
	consumer       *broker.Consumer
	paymentService *service.PaymentService
}

// NewPaymentWorker creates a new payment worker
func NewPaymentWorker(
	consumer *broker.Consumer,
	paymentService *service.PaymentService,
) *PaymentWorker {
	return &PaymentWorker{
		consumer:       consumer,
		paymentService: paymentService,
	}
}

// Start starts the payment worker
func (pw *PaymentWorker) Start(ctx context.Context) error {
	log.Println("Starting payment worker...")

	return pw.consumer.StartConsuming(ctx, func(ctx context.Context, msg kafka.Message) error {
		var baseEvent models.BaseEvent
		if err := json.Unmarshal(msg.Value, &baseEvent); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			return err
		}

		if baseEvent.EventType == models.EventTypeOrderReserved {
			var event models.OrderReservedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal OrderReserved event: %v", err)
				return err
			}

			log.Printf("Processing payment for order: %d", event.OrderID)

			return pw.paymentService.ProcessPayment(ctx, event.OrderID, event.TotalAmount)
		}

		return nil
	})
}

// Stop stops the payment worker
func (pw *PaymentWorker) Stop() error {
	log.Println("Stopping payment worker...")
	return pw.consumer.Close()
}
