package service

import (
	"context"
	"fmt"
	"time"

	"order-service/internal/models"
	"order-service/internal/redisclient"
	"order-service/internal/store"
	"order-service/internal/util"

	"go.uber.org/zap"
)

// InventoryClient handles inventory operations
type InventoryClient struct {
	store  *store.Store
	redis  *redisclient.Client
	logger *zap.Logger
}

// NewInventoryClient creates a new inventory client
func NewInventoryClient(store *store.Store, redis *redisclient.Client) *InventoryClient {
	return &InventoryClient{
		store:  store,
		redis:  redis,
		logger: util.GetLogger(),
	}
}

// ReserveStock reserves stock for a product (fast path via Redis)
func (ic *InventoryClient) ReserveStock(ctx context.Context, productID int64, quantity int) (bool, error) {
	ctx, span := util.StartSpan(ctx, "InventoryClient.ReserveStock")
	defer span.End()

	success, err := ic.redis.ReserveStock(ctx, productID, quantity)
	if err != nil {
		ic.logger.Warn("Redis reservation failed, falling back to DB",
			zap.Int64("product_id", productID),
			zap.Error(err))

		return ic.reserveStockDB(ctx, productID, quantity)
	}

	if !success {
		return false, nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := ic.store.ReserveStockTx(ctx, productID, quantity); err != nil {
			ic.logger.Error("Failed to sync reservation to DB",
				zap.Int64("product_id", productID),
				zap.Error(err))
		}
	}()

	return true, nil
}

// reserveStockDB reserves stock using database transaction (fallback)
func (ic *InventoryClient) reserveStockDB(ctx context.Context, productID int64, quantity int) (bool, error) {
	err := ic.store.ReserveStockTx(ctx, productID, quantity)
	if err != nil {
		if err.Error() == "insufficient stock" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ReleaseStock releases reserved stock (compensation)
func (ic *InventoryClient) ReleaseStock(ctx context.Context, productID int64, quantity int) error {
	ctx, span := util.StartSpan(ctx, "InventoryClient.ReleaseStock")
	defer span.End()

	if err := ic.redis.ReleaseStock(ctx, productID, quantity); err != nil {
		ic.logger.Error("Failed to release stock in Redis",
			zap.Int64("product_id", productID),
			zap.Error(err))
	}

	return ic.store.ReleaseStock(ctx, productID, quantity)
}

// CommitStock commits reserved stock (final deduction)
func (ic *InventoryClient) CommitStock(ctx context.Context, productID int64, quantity int) error {
	ctx, span := util.StartSpan(ctx, "InventoryClient.CommitStock")
	defer span.End()

	if err := ic.redis.CommitStock(ctx, productID, quantity); err != nil {
		ic.logger.Error("Failed to commit stock in Redis",
			zap.Int64("product_id", productID),
			zap.Error(err))
	}

	return ic.store.CommitStock(ctx, productID, quantity)
}

// SyncInventoryToRedis synchronizes database inventory to Redis
func (ic *InventoryClient) SyncInventoryToRedis(ctx context.Context) error {
	ic.logger.Info("Starting inventory sync to Redis")

	products, err := ic.store.GetProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get products: %w", err)
	}

	for _, product := range products {
		inv, err := ic.store.GetInventory(ctx, product.ID)
		if err != nil {
			ic.logger.Error("Failed to get inventory",
				zap.Int64("product_id", product.ID),
				zap.Error(err))
			continue
		}

		if err := ic.redis.InitInventory(ctx, product.ID, inv.Available, inv.Reserved); err != nil {
			ic.logger.Error("Failed to init Redis inventory",
				zap.Int64("product_id", product.ID),
				zap.Error(err))
		}
	}

	ic.logger.Info("Inventory sync completed", zap.Int("count", len(products)))
	return nil
}

// GetInventory retrieves inventory for a product
func (ic *InventoryClient) GetInventory(ctx context.Context, productID int64) (*models.Inventory, error) {
	return ic.store.GetInventory(ctx, productID)
}
