package service

import (
	"context"
	"fmt"
	"time"

	"order-service/internal/broker"
	"order-service/internal/models"
	"order-service/internal/redisclient"
	"order-service/internal/store"
	"order-service/internal/util"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OrderService handles order business logic
type OrderService struct {
	store           *store.Store
	redis           *redisclient.Client
	eventPublisher  *broker.EventPublisher
	inventoryClient *InventoryClient
	logger          *zap.Logger
}

// NewOrderService creates a new order service
func NewOrderService(
	store *store.Store,
	redis *redisclient.Client,
	eventPublisher *broker.EventPublisher,
	inventoryClient *InventoryClient,
) *OrderService {
	return &OrderService{
		store:           store,
		redis:           redis,
		eventPublisher:  eventPublisher,
		inventoryClient: inventoryClient,
		logger:          util.GetLogger(),
	}
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	UserID         int64              `json:"user_id" binding:"required"`
	Items          []OrderItemRequest `json:"items" binding:"required,min=1"`
	PaymentMethod  string             `json:"payment_method" binding:"required"`
	IdempotencyKey string             `json:"idempotency_key,omitempty"`
}

// OrderItemRequest represents an item in an order
type OrderItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required,min=1"`
}

// CreateOrderResponse represents the response after creating an order
type CreateOrderResponse struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}

// CreateOrder creates a new order with saga orchestration
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	ctx, span := util.StartSpan(ctx, "OrderService.CreateOrder")
	defer span.End()

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = uuid.New().String()
	}

	existingOrder, err := s.store.GetOrderByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	if existingOrder != nil {
		s.logger.Info("Duplicate order request detected",
			zap.String("idempotency_key", req.IdempotencyKey),
			zap.Int64("order_id", existingOrder.ID))
		return &CreateOrderResponse{
			OrderID: existingOrder.ID,
			Status:  existingOrder.Status,
		}, nil
	}

	products, err := s.validateOrderItems(ctx, req.Items)
	if err != nil {
		util.OrdersFailedTotal.WithLabelValues("invalid_items").Inc()
		return nil, err
	}

	totalAmount := s.calculateTotal(req.Items, products)

	order := &models.Order{
		UserID:         req.UserID,
		TotalAmount:    totalAmount,
		Status:         models.OrderStatusCreated,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := s.store.CreateOrder(ctx, order); err != nil {
		util.OrdersFailedTotal.WithLabelValues("db_error").Inc()
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	util.OrdersCreatedTotal.Inc()
	s.logger.Info("Order created", zap.Int64("order_id", order.ID))

	// Create order items
	orderItems := make([]models.OrderItemData, 0, len(req.Items))
	for _, item := range req.Items {
		product := products[item.ProductID]
		orderItem := &models.OrderItem{
			OrderID:   order.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: product.Price,
		}

		if err := s.store.CreateOrderItem(ctx, orderItem); err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}

		orderItems = append(orderItems, models.OrderItemData{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: product.Price,
		})
	}

	event := &models.OrderCreatedEvent{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.EventTypeOrderCreated,
			Timestamp: time.Now(),
		},
		OrderID:     order.ID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Items:       orderItems,
	}

	if err := s.eventPublisher.PublishOrderCreated(ctx, event); err != nil {
		s.logger.Error("Failed to publish OrderCreated event", zap.Error(err))
	}

	if err := s.reserveInventory(ctx, order.ID, req.Items); err != nil {
		_ = s.store.UpdateOrderStatus(ctx, order.ID, models.OrderStatusFailed)
		util.OrdersFailedTotal.WithLabelValues("reservation_failed").Inc()
		return nil, fmt.Errorf("inventory reservation failed: %w", err)
	}

	if err := s.store.UpdateOrderStatus(ctx, order.ID, models.OrderStatusReserved); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	util.OrdersReservedTotal.Inc()

	reservedEvent := &models.OrderReservedEvent{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.EventTypeOrderReserved,
			Timestamp: time.Now(),
		},
		OrderID:     order.ID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Items:       orderItems,
	}

	if err := s.eventPublisher.PublishOrderReserved(ctx, reservedEvent); err != nil {
		s.logger.Error("Failed to publish OrderReserved event", zap.Error(err))
	}

	return &CreateOrderResponse{
		OrderID: order.ID,
		Status:  models.OrderStatusReserved,
	}, nil
}

// reserveInventory reserves inventory for order items
func (s *OrderService) reserveInventory(ctx context.Context, orderID int64, items []OrderItemRequest) error {
	timer := util.InventoryReserveLatency
	start := time.Now()
	defer func() {
		timer.Observe(time.Since(start).Seconds())
	}()

	for _, item := range items {
		success, err := s.inventoryClient.ReserveStock(ctx, item.ProductID, item.Quantity)
		if err != nil {
			util.InventoryReservationsFailed.WithLabelValues("error").Inc()
			s.compensateReservations(ctx, orderID, items)
			return fmt.Errorf("failed to reserve stock for product %d: %w", item.ProductID, err)
		}

		if !success {
			util.InventoryReservationsFailed.WithLabelValues("insufficient_stock").Inc()
			s.compensateReservations(ctx, orderID, items)
			return fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}
	}

	return nil
}

// compensateReservations rolls back inventory reservations
func (s *OrderService) compensateReservations(ctx context.Context, orderID int64, items []OrderItemRequest) {
	for _, item := range items {
		if err := s.inventoryClient.ReleaseStock(ctx, item.ProductID, item.Quantity); err != nil {
			s.logger.Error("Failed to compensate reservation",
				zap.Int64("order_id", orderID),
				zap.Int64("product_id", item.ProductID),
				zap.Error(err))
		}
	}
}

// validateOrderItems validates that all products exist
func (s *OrderService) validateOrderItems(ctx context.Context, items []OrderItemRequest) (map[int64]*models.Product, error) {
	productIDs := make([]int64, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	products, err := s.store.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	if len(products) != len(items) {
		return nil, fmt.Errorf("some products not found")
	}

	productMap := make(map[int64]*models.Product)
	for i := range products {
		productMap[products[i].ID] = &products[i]
	}

	return productMap, nil
}

// calculateTotal calculates the total amount for an order
func (s *OrderService) calculateTotal(items []OrderItemRequest, products map[int64]*models.Product) int64 {
	var total int64
	for _, item := range items {
		product := products[item.ProductID]
		total += product.Price * int64(item.Quantity)
	}
	return total
}

// GetOrder retrieves an order by ID
func (s *OrderService) GetOrder(ctx context.Context, orderID int64) (*models.Order, []models.OrderItem, error) {
	order, err := s.store.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	items, err := s.store.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}

	return order, items, nil
}
