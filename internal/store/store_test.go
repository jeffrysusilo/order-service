package store

import (
	"context"
	"testing"

	"order-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder(t *testing.T) {
	// This is a placeholder test - requires actual database connection
	// In real scenarios, use testcontainers or mock database

	t.Skip("Integration test - requires database")

	store, err := NewStore("postgres://app:secret@localhost:5432/app_test?sslmode=disable")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	order := &models.Order{
		UserID:         123,
		TotalAmount:    1000000,
		Status:         models.OrderStatusCreated,
		IdempotencyKey: "test-key-123",
	}

	err = store.CreateOrder(ctx, order)
	assert.NoError(t, err)
	assert.NotZero(t, order.ID)

	// Retrieve order
	retrieved, err := store.GetOrderByID(ctx, order.ID)
	assert.NoError(t, err)
	assert.Equal(t, order.UserID, retrieved.UserID)
	assert.Equal(t, order.TotalAmount, retrieved.TotalAmount)
}

func TestIdempotency(t *testing.T) {
	t.Skip("Integration test - requires database")

	store, err := NewStore("postgres://app:secret@localhost:5432/app_test?sslmode=disable")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	order := &models.Order{
		UserID:         123,
		TotalAmount:    1000000,
		Status:         models.OrderStatusCreated,
		IdempotencyKey: "idempotent-key-456",
	}

	// First creation
	err = store.CreateOrder(ctx, order)
	assert.NoError(t, err)

	// Second creation with same key should fail (unique constraint)
	order2 := &models.Order{
		UserID:         456,
		TotalAmount:    2000000,
		Status:         models.OrderStatusCreated,
		IdempotencyKey: "idempotent-key-456",
	}

	err = store.CreateOrder(ctx, order2)
	assert.Error(t, err) // Should fail due to unique constraint
}
