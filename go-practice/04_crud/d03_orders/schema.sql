CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock BIGINT NOT NULL CHECK (stock >= 0)
);

CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'confirmed'))
);

CREATE TABLE order_items (
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    qty BIGINT NOT NULL CHECK (qty > 0),
    price_cents BIGINT NULL CHECK (price_cents >= 0),
    PRIMARY KEY (order_id, product_id)
);
