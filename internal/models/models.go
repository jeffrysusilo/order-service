package models

import "time"

// Product represents a product in the catalog
type Product struct {
	ID        int64     `db:"id" json:"id"`
	SKU       string    `db:"sku" json:"sku"`
	Name      string    `db:"name" json:"name"`
	Price     int64     `db:"price" json:"price"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Inventory represents product stock
type Inventory struct {
	ProductID int64     `db:"product_id" json:"product_id"`
	Available int       `db:"available" json:"available"`
	Reserved  int       `db:"reserved" json:"reserved"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Order represents a customer order
type Order struct {
	ID             int64     `db:"id" json:"id"`
	UserID         int64     `db:"user_id" json:"user_id"`
	TotalAmount    int64     `db:"total_amount" json:"total_amount"`
	Status         string    `db:"status" json:"status"`
	IdempotencyKey string    `db:"idempotency_key" json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// OrderItem represents items in an order
type OrderItem struct {
	ID        int64 `db:"id" json:"id"`
	OrderID   int64 `db:"order_id" json:"order_id"`
	ProductID int64 `db:"product_id" json:"product_id"`
	Quantity  int   `db:"quantity" json:"quantity"`
	UnitPrice int64 `db:"unit_price" json:"unit_price"`
}

// Payment represents a payment transaction
type Payment struct {
	ID           int64     `db:"id" json:"id"`
	OrderID      int64     `db:"order_id" json:"order_id"`
	Status       string    `db:"status" json:"status"`
	ProviderTxID string    `db:"provider_tx_id" json:"provider_tx_id,omitempty"`
	Amount       int64     `db:"amount" json:"amount"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// Order statuses
const (
	OrderStatusCreated   = "CREATED"
	OrderStatusReserved  = "RESERVED"
	OrderStatusPaid      = "PAID"
	OrderStatusConfirmed = "CONFIRMED"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusFailed    = "FAILED"
)

// Payment statuses
const (
	PaymentStatusPending = "PENDING"
	PaymentStatusSuccess = "SUCCESS"
	PaymentStatusFailed  = "FAILED"
)

// ProcessedEvent for idempotency
type ProcessedEvent struct {
	EventID     string    `db:"event_id"`
	EventType   string    `db:"event_type"`
	ProcessedAt time.Time `db:"processed_at"`
}
