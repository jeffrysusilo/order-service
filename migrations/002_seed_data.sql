-- Seed sample products
INSERT INTO products (sku, name, price) VALUES
    ('LAPTOP-001', 'Gaming Laptop RTX 4070', 1500000),
    ('PHONE-001', 'Smartphone Pro Max', 1200000),
    ('HEADSET-001', 'Wireless Gaming Headset', 50000),
    ('MOUSE-001', 'RGB Gaming Mouse', 30000),
    ('KEYBOARD-001', 'Mechanical Keyboard', 80000)
ON CONFLICT (sku) DO NOTHING;

-- Seed initial inventory
INSERT INTO inventory (product_id, available, reserved)
SELECT id, 100, 0 FROM products
ON CONFLICT (product_id) DO NOTHING;
