package service

import (
	"testing"

	"order-service/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestCalculateTotal(t *testing.T) {
	os := &OrderService{}

	items := []OrderItemRequest{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 1},
	}

	products := map[int64]*models.Product{
		1: {ID: 1, Price: 1000},
		2: {ID: 2, Price: 500},
	}

	total := os.calculateTotal(items, products)

	expected := int64(2*1000 + 1*500) // 2500
	assert.Equal(t, expected, total)
}

func TestValidateOrderItems(t *testing.T) {
	// This would require mocking the store
	// Placeholder for demonstration
	t.Skip("Requires mocked store")
}
