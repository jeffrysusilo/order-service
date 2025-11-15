package service

import (
	"context"
	"fmt"

	"order-service/internal/broker"
	"order-service/internal/models"
	"order-service/internal/store"
	"order-service/internal/util"

	"go.uber.org/zap"
)

// SagaOrchestrator orchestrates the order saga workflow
type SagaOrchestrator struct {
	store           *store.Store
	inventoryClient *InventoryClient
	paymentService  *PaymentService
	eventPublisher  *broker.EventPublisher
	logger          *zap.Logger
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(
	store *store.Store,
	inventoryClient *InventoryClient,
	paymentService *PaymentService,
	eventPublisher *broker.EventPublisher,
) *SagaOrchestrator {
	return &SagaOrchestrator{
		store:           store,
		inventoryClient: inventoryClient,
		paymentService:  paymentService,
		eventPublisher:  eventPublisher,
		logger:          util.GetLogger(),
	}
}

// HandlePaymentSuccess handles successful payment event
func (so *SagaOrchestrator) HandlePaymentSuccess(ctx context.Context, event *models.PaymentSuccessEvent) error {
	ctx, span := util.StartSpan(ctx, "SagaOrchestrator.HandlePaymentSuccess")
	defer span.End()

	processed, err := so.store.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event processed: %w", err)
	}
	if processed {
		so.logger.Info("Event already processed", zap.String("event_id", event.EventID))
		return nil
	}

	so.logger.Info("Handling payment success",
		zap.Int64("order_id", event.OrderID),
		zap.String("tx_id", event.TxID))

	if err := so.store.UpdateOrderStatus(ctx, event.OrderID, models.OrderStatusPaid); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	util.OrdersPaidTotal.Inc()

	items, err := so.store.GetOrderItemsByOrderID(ctx, event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	for _, item := range items {
		if err := so.inventoryClient.CommitStock(ctx, item.ProductID, item.Quantity); err != nil {
			so.logger.Error("Failed to commit stock",
				zap.Int64("product_id", item.ProductID),
				zap.Error(err))
		}
	}

	// Update order to CONFIRMED
	if err := so.store.UpdateOrderStatus(ctx, event.OrderID, models.OrderStatusConfirmed); err != nil {
		so.logger.Error("Failed to confirm order", zap.Error(err))
	}

	if err := so.store.MarkEventProcessed(ctx, event.EventID, event.EventType); err != nil {
		so.logger.Error("Failed to mark event processed", zap.Error(err))
	}

	so.logger.Info("Order confirmed", zap.Int64("order_id", event.OrderID))
	return nil
}

// HandlePaymentFailed handles failed payment event (compensation)
func (so *SagaOrchestrator) HandlePaymentFailed(ctx context.Context, event *models.PaymentFailedEvent) error {
	ctx, span := util.StartSpan(ctx, "SagaOrchestrator.HandlePaymentFailed")
	defer span.End()

	processed, err := so.store.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event processed: %w", err)
	}
	if processed {
		so.logger.Info("Event already processed", zap.String("event_id", event.EventID))
		return nil
	}

	so.logger.Warn("Handling payment failure - starting compensation",
		zap.Int64("order_id", event.OrderID),
		zap.String("reason", event.Reason))

	items, err := so.store.GetOrderItemsByOrderID(ctx, event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	for _, item := range items {
		if err := so.inventoryClient.ReleaseStock(ctx, item.ProductID, item.Quantity); err != nil {
			so.logger.Error("Failed to release stock during compensation",
				zap.Int64("product_id", item.ProductID),
				zap.Error(err))
		}
	}

	if err := so.store.UpdateOrderStatus(ctx, event.OrderID, models.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	util.OrdersCancelledTotal.Inc()

	if err := so.store.MarkEventProcessed(ctx, event.EventID, event.EventType); err != nil {
		so.logger.Error("Failed to mark event processed", zap.Error(err))
	}

	so.logger.Info("Order cancelled and compensated", zap.Int64("order_id", event.OrderID))
	return nil
}
