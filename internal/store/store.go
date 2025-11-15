package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"order-service/internal/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Store struct {
	db *sqlx.DB
}

// NewStore creates a new database store
func NewStore(databaseURL string) (*Store, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// GetDB returns the underlying database connection
func (s *Store) GetDB() *sqlx.DB {
	return s.db
}

// GetProductByID retrieves a product by ID
func (s *Store) GetProductByID(ctx context.Context, id int64) (*models.Product, error) {
	var product models.Product
	err := s.db.GetContext(ctx, &product, "SELECT * FROM products WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found: %d", id)
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetProductBySKU retrieves a product by SKU
func (s *Store) GetProductBySKU(ctx context.Context, sku string) (*models.Product, error) {
	var product models.Product
	err := s.db.GetContext(ctx, &product, "SELECT * FROM products WHERE sku = $1", sku)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found: %s", sku)
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetProducts retrieves all products
func (s *Store) GetProducts(ctx context.Context) ([]models.Product, error) {
	var products []models.Product
	err := s.db.SelectContext(ctx, &products, "SELECT * FROM products ORDER BY id")
	return products, err
}

// GetProductsByIDs retrieves multiple products by IDs
func (s *Store) GetProductsByIDs(ctx context.Context, ids []int64) ([]models.Product, error) {
	if len(ids) == 0 {
		return []models.Product{}, nil
	}

	query, args, err := sqlx.In("SELECT * FROM products WHERE id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	var products []models.Product
	err = s.db.SelectContext(ctx, &products, query, args...)
	return products, err
}

// GetInventory retrieves inventory for a product
func (s *Store) GetInventory(ctx context.Context, productID int64) (*models.Inventory, error) {
	var inv models.Inventory
	err := s.db.GetContext(ctx, &inv, "SELECT * FROM inventory WHERE product_id = $1", productID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory not found for product: %d", productID)
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ReserveStockTx reserves stock within a transaction (FOR UPDATE lock)
func (s *Store) ReserveStockTx(ctx context.Context, productID int64, quantity int) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var available int
	err = tx.GetContext(ctx, &available,
		"SELECT available FROM inventory WHERE product_id = $1 FOR UPDATE", productID)
	if err != nil {
		return fmt.Errorf("failed to lock inventory: %w", err)
	}

	if available < quantity {
		return fmt.Errorf("insufficient stock: available=%d, requested=%d", available, quantity)
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE inventory SET available = available - $1, reserved = reserved + $1, updated_at = NOW() WHERE product_id = $2",
		quantity, productID)
	if err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	return tx.Commit()
}

// ReleaseStock releases reserved stock (compensation)
func (s *Store) ReleaseStock(ctx context.Context, productID int64, quantity int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE inventory SET available = available + $1, reserved = reserved - $1, updated_at = NOW() WHERE product_id = $2",
		quantity, productID)
	return err
}

// CommitStock commits reserved stock (final deduction)
func (s *Store) CommitStock(ctx context.Context, productID int64, quantity int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE inventory SET reserved = reserved - $1, updated_at = NOW() WHERE product_id = $2",
		quantity, productID)
	return err
}

// UpdateInventory updates inventory counts
func (s *Store) UpdateInventory(ctx context.Context, productID int64, available, reserved int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE inventory SET available = $1, reserved = $2, updated_at = NOW() WHERE product_id = $3",
		available, reserved, productID)
	return err
}
