package store

import (
	"context"
	"database/sql"
	"fmt"

	"order-service/internal/models"
)

// CreateOrder creates a new order
func (s *Store) CreateOrder(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (user_id, total_amount, status, idempotency_key)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return s.db.GetContext(ctx, order, query,
		order.UserID, order.TotalAmount, order.Status, order.IdempotencyKey)
}

// GetOrderByID retrieves an order by ID
func (s *Store) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	var order models.Order
	err := s.db.GetContext(ctx, &order, "SELECT * FROM orders WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %d", id)
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrderByIdempotencyKey retrieves an order by idempotency key
func (s *Store) GetOrderByIdempotencyKey(ctx context.Context, key string) (*models.Order, error) {
	var order models.Order
	err := s.db.GetContext(ctx, &order, "SELECT * FROM orders WHERE idempotency_key = $1", key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// UpdateOrderStatus updates order status
func (s *Store) UpdateOrderStatus(ctx context.Context, orderID int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2",
		status, orderID)
	return err
}

// GetOrdersByUserID retrieves orders for a user
func (s *Store) GetOrdersByUserID(ctx context.Context, userID int64) ([]models.Order, error) {
	var orders []models.Order
	err := s.db.SelectContext(ctx, &orders,
		"SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC", userID)
	return orders, err
}

// CreateOrderItem creates a new order item
func (s *Store) CreateOrderItem(ctx context.Context, item *models.OrderItem) error {
	query := `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	return s.db.GetContext(ctx, &item.ID, query,
		item.OrderID, item.ProductID, item.Quantity, item.UnitPrice)
}

// GetOrderItemsByOrderID retrieves all items for an order
func (s *Store) GetOrderItemsByOrderID(ctx context.Context, orderID int64) ([]models.OrderItem, error) {
	var items []models.OrderItem
	err := s.db.SelectContext(ctx, &items,
		"SELECT * FROM order_items WHERE order_id = $1", orderID)
	return items, err
}

// CreatePayment creates a new payment record
func (s *Store) CreatePayment(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (order_id, status, provider_tx_id, amount)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return s.db.GetContext(ctx, payment, query,
		payment.OrderID, payment.Status, payment.ProviderTxID, payment.Amount)
}

// GetPaymentByOrderID retrieves payment for an order
func (s *Store) GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error) {
	var payment models.Payment
	err := s.db.GetContext(ctx, &payment,
		"SELECT * FROM payments WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1", orderID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found for order: %d", orderID)
	}
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePaymentStatus updates payment status
func (s *Store) UpdatePaymentStatus(ctx context.Context, paymentID int64, status, providerTxID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE payments SET status = $1, provider_tx_id = $2, updated_at = NOW() WHERE id = $3",
		status, providerTxID, paymentID)
	return err
}

// IsEventProcessed checks if an event has been processed
func (s *Store) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := s.db.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)", eventID)
	return exists, err
}

// MarkEventProcessed marks an event as processed
func (s *Store) MarkEventProcessed(ctx context.Context, eventID, eventType string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO processed_events (event_id, event_type) VALUES ($1, $2) ON CONFLICT (event_id) DO NOTHING",
		eventID, eventType)
	return err
}
