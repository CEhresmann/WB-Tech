CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(50) PRIMARY KEY,
    track_number VARCHAR(50),
    entry VARCHAR(10),
    locale VARCHAR(10),
    internal_signature TEXT,
    customer_id VARCHAR(50),
    delivery_service VARCHAR(50),
    shardkey VARCHAR(10),
    sm_id INT,
    date_created TIMESTAMPTZ,
    oof_shard VARCHAR(10)
    );

CREATE TABLE IF NOT EXISTS delivery (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(50) REFERENCES orders(order_uid) ON DELETE CASCADE,
    name VARCHAR(100),
    phone VARCHAR(20),
    zip VARCHAR(20),
    city VARCHAR(100),
    address VARCHAR(255),
    region VARCHAR(100),
    email VARCHAR(100)
    );

CREATE TABLE IF NOT EXISTS payment (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(50) REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction VARCHAR(50),
    request_id VARCHAR(50),
    currency VARCHAR(10),
    provider VARCHAR(50),
    amount INT,
    payment_dt BIGINT,
    bank VARCHAR(50),
    delivery_cost INT,
    goods_total INT,
    custom_fee INT
    );

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(50) REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id INT,
    track_number VARCHAR(50),
    price INT,
    rid VARCHAR(50),
    name VARCHAR(100),
    sale INT,
    size VARCHAR(10),
    total_price INT,
    nm_id INT,
    brand VARCHAR(100),
    status INT
    );