package redisclient

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

//go:embed scripts/reserve_stock.lua
var reserveStockScript string

//go:embed scripts/release_stock.lua
var releaseStockScript string

//go:embed scripts/commit_stock.lua
var commitStockScript string

type Client struct {
	rdb           *redis.Client
	reserveScript *redis.Script
	releaseScript *redis.Script
	commitScript  *redis.Script
}

// NewClient creates a new Redis client with Lua scripts loaded
func NewClient(addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{
		rdb:           rdb,
		reserveScript: redis.NewScript(reserveStockScript),
		releaseScript: redis.NewScript(releaseStockScript),
		commitScript:  redis.NewScript(commitStockScript),
	}, nil
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// ReserveStock atomically reserves stock using Lua script
// Returns true if reservation successful, false if insufficient stock
func (c *Client) ReserveStock(ctx context.Context, productID int64, quantity int) (bool, error) {
	key := fmt.Sprintf("inventory:%d", productID)

	result, err := c.reserveScript.Run(ctx, c.rdb, []string{key}, quantity).Result()
	if err != nil {
		return false, fmt.Errorf("reserve stock script failed: %w", err)
	}

	success, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected script result type")
	}

	return success == 1, nil
}

// ReleaseStock atomically releases reserved stock (compensation)
func (c *Client) ReleaseStock(ctx context.Context, productID int64, quantity int) error {
	key := fmt.Sprintf("inventory:%d", productID)

	_, err := c.releaseScript.Run(ctx, c.rdb, []string{key}, quantity).Result()
	if err != nil {
		return fmt.Errorf("release stock script failed: %w", err)
	}

	return nil
}

// CommitStock atomically commits reserved stock (final deduction)
func (c *Client) CommitStock(ctx context.Context, productID int64, quantity int) error {
	key := fmt.Sprintf("inventory:%d", productID)

	_, err := c.commitScript.Run(ctx, c.rdb, []string{key}, quantity).Result()
	if err != nil {
		return fmt.Errorf("commit stock script failed: %w", err)
	}

	return nil
}

// InitInventory initializes inventory count in Redis
func (c *Client) InitInventory(ctx context.Context, productID int64, available, reserved int) error {
	key := fmt.Sprintf("inventory:%d", productID)

	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, "available", available)
	pipe.HSet(ctx, key, "reserved", reserved)

	_, err := pipe.Exec(ctx)
	return err
}

// GetInventory retrieves current inventory counts
func (c *Client) GetInventory(ctx context.Context, productID int64) (available, reserved int, err error) {
	key := fmt.Sprintf("inventory:%d", productID)

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}

	if len(result) == 0 {
		return 0, 0, fmt.Errorf("inventory not found for product %d", productID)
	}

	var availableInt, reservedInt int
	fmt.Sscanf(result["available"], "%d", &availableInt)
	fmt.Sscanf(result["reserved"], "%d", &reservedInt)

	return availableInt, reservedInt, nil
}

// SetIdempotencyKey stores an idempotency key with TTL
func (c *Client) SetIdempotencyKey(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Set(ctx, fmt.Sprintf("idempotency:%s", key), value, ttl).Err()
}

// CheckIdempotencyKey checks if an idempotency key exists
func (c *Client) CheckIdempotencyKey(ctx context.Context, key string) (bool, error) {
	result, err := c.rdb.Exists(ctx, fmt.Sprintf("idempotency:%s", key)).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// AcquireLock acquires a distributed lock
func (c *Client) AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, fmt.Sprintf("lock:%s", lockKey), "1", ttl).Result()
}

// ReleaseLock releases a distributed lock
func (c *Client) ReleaseLock(ctx context.Context, lockKey string) error {
	return c.rdb.Del(ctx, fmt.Sprintf("lock:%s", lockKey)).Err()
}
