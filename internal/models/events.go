package models

import "time"

// Event types
const (
	EventTypeOrderCreated   = "ORDER_CREATED"
	EventTypeOrderReserved  = "ORDER_RESERVED"
	EventTypeOrderPaid      = "ORDER_PAID"
	EventTypeOrderConfirmed = "ORDER_CONFIRMED"
	EventTypeOrderCancelled = "ORDER_CANCELLED"
	EventTypeOrderFailed    = "ORDER_FAILED"
	EventTypePaymentSuccess = "PAYMENT_SUCCESS"
	EventTypePaymentFailed  = "PAYMENT_FAILED"
)

// BaseEvent contains common fields for all events
type BaseEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderCreatedEvent published when order is created
type OrderCreatedEvent struct {
	BaseEvent
	OrderID     int64           `json:"order_id"`
	UserID      int64           `json:"user_id"`
	TotalAmount int64           `json:"total_amount"`
	Items       []OrderItemData `json:"items"`
}

// OrderReservedEvent published when inventory is reserved
type OrderReservedEvent struct {
	BaseEvent
	OrderID     int64           `json:"order_id"`
	UserID      int64           `json:"user_id"`
	TotalAmount int64           `json:"total_amount"`
	Items       []OrderItemData `json:"items"`
}

// OrderPaidEvent published when payment succeeds
type OrderPaidEvent struct {
	BaseEvent
	OrderID   int64  `json:"order_id"`
	PaymentID int64  `json:"payment_id"`
	Amount    int64  `json:"amount"`
	TxID      string `json:"tx_id"`
}

// OrderConfirmedEvent published when order is fully confirmed
type OrderConfirmedEvent struct {
	BaseEvent
	OrderID int64 `json:"order_id"`
	UserID  int64 `json:"user_id"`
}

// OrderCancelledEvent published when order is cancelled (compensation)
type OrderCancelledEvent struct {
	BaseEvent
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason"`
}

// PaymentSuccessEvent published by payment service
type PaymentSuccessEvent struct {
	BaseEvent
	OrderID   int64  `json:"order_id"`
	PaymentID int64  `json:"payment_id"`
	Amount    int64  `json:"amount"`
	TxID      string `json:"tx_id"`
}

// PaymentFailedEvent published by payment service
type PaymentFailedEvent struct {
	BaseEvent
	OrderID   int64  `json:"order_id"`
	PaymentID int64  `json:"payment_id"`
	Reason    string `json:"reason"`
}

// OrderItemData represents item data in events
type OrderItemData struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	UnitPrice int64 `json:"unit_price"`
}
