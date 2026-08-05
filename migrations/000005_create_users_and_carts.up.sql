-- 1. Create Users Table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed a test user so user_id: 1 works out of the box
INSERT INTO users (id, username, email) 
VALUES (1, 'Test User', 'test@example.com') 
ON CONFLICT (id) DO NOTHING;

-- 2. Create Carts Table
CREATE TABLE IF NOT EXISTS carts (
    id VARCHAR(100) PRIMARY KEY,
    user_id INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create Cart Items Table (Normalized)
CREATE TABLE IF NOT EXISTS cart_items (
    cart_id VARCHAR(100) NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    item_id INT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    quantity INT NOT NULL CHECK (quantity > 0),
    price INT NOT NULL, -- snapshot of price per item
    PRIMARY KEY (cart_id, item_id)
);