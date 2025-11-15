package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"order-service/internal/broker"
	"order-service/internal/models"
	"order-service/internal/store"
	"order-service/internal/util"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentService handles payment processing (mocked)
type PaymentService struct {
	store          *store.Store
	eventPublisher *broker.EventPublisher
	logger         *zap.Logger
	successRate    float64 // Mock success rate (0.0 - 1.0)
}

// NewPaymentService creates a new payment service
func NewPaymentService(store *store.Store, eventPublisher *broker.EventPublisher) *PaymentService {
	return &PaymentService{
		store:          store,
		eventPublisher: eventPublisher,
		logger:         util.GetLogger(),
		successRate:    0.9, // 90% success rate for testing
	}
}

// ProcessPayment processes payment for an order (mocked)
func (ps *PaymentService) ProcessPayment(ctx context.Context, orderID int64, amount int64) error {
	ctx, span := util.StartSpan(ctx, "PaymentService.ProcessPayment")
	defer span.End()

	util.PaymentAttemptsTotal.Inc()
	start := time.Now()
	defer func() {
		util.PaymentProcessingLatency.Observe(time.Since(start).Seconds())
	}()

	ps.logger.Info("Processing payment",
		zap.Int64("order_id", orderID),
		zap.Int64("amount", amount))

	payment := &models.Payment{
		OrderID:      orderID,
		Status:       models.PaymentStatusPending,
		Amount:       amount,
		ProviderTxID: "",
	}

	if err := ps.store.CreatePayment(ctx, payment); err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

	success := rand.Float64() < ps.successRate
	providerTxID := fmt.Sprintf("TXN-%s", uuid.New().String()[:8])

	if success {
		ps.logger.Info("Payment succeeded",
			zap.Int64("order_id", orderID),
			zap.String("tx_id", providerTxID))

		if err := ps.store.UpdatePaymentStatus(ctx, payment.ID, models.PaymentStatusSuccess, providerTxID); err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		util.PaymentSuccessTotal.Inc()

		event := &models.PaymentSuccessEvent{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				EventType: models.EventTypePaymentSuccess,
				Timestamp: time.Now(),
			},
			OrderID:   orderID,
			PaymentID: payment.ID,
			Amount:    amount,
			TxID:      providerTxID,
		}

		if err := ps.eventPublisher.PublishPaymentSuccess(ctx, event); err != nil {
			ps.logger.Error("Failed to publish PaymentSuccess event", zap.Error(err))
		}

	} else {
		ps.logger.Warn("Payment failed",
			zap.Int64("order_id", orderID))

		if err := ps.store.UpdatePaymentStatus(ctx, payment.ID, models.PaymentStatusFailed, ""); err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		util.PaymentFailedTotal.Inc()

		event := &models.PaymentFailedEvent{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				EventType: models.EventTypePaymentFailed,
				Timestamp: time.Now(),
			},
			OrderID:   orderID,
			PaymentID: payment.ID,
			Reason:    "mock_payment_declined",
		}

		if err := ps.eventPublisher.PublishPaymentFailed(ctx, event); err != nil {
			ps.logger.Error("Failed to publish PaymentFailed event", zap.Error(err))
		}
	}

	return nil
}

// GetPayment retrieves payment for an order
func (ps *PaymentService) GetPayment(ctx context.Context, orderID int64) (*models.Payment, error) {
	return ps.store.GetPaymentByOrderID(ctx, orderID)
}
