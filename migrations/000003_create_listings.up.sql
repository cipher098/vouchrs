CREATE TABLE listings (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    seller_id        UUID NOT NULL REFERENCES users(id),
    brand_id         UUID NOT NULL REFERENCES brands(id),
    face_value       NUMERIC(12,2) NOT NULL,
    buyer_price      NUMERIC(12,2) NOT NULL,
    seller_payout    NUMERIC(12,2) NOT NULL,
    discount_pct     NUMERIC(5,2) NOT NULL,
    is_pool          BOOLEAN NOT NULL DEFAULT FALSE,
    code_encrypted   TEXT NOT NULL,
    code_hash        VARCHAR(64) NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'LIVE',
    lock_buyer_id    UUID REFERENCES users(id),
    lock_expires_at  TIMESTAMPTZ,
    gate1_at         TIMESTAMPTZ,
    sold_at          TIMESTAMPTZ,
    verified_balance NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_listings_seller_id  ON listings(seller_id);
CREATE INDEX idx_listings_brand_id   ON listings(brand_id);
CREATE INDEX idx_listings_status     ON listings(status);
CREATE INDEX idx_listings_code_hash  ON listings(code_hash);
CREATE INDEX idx_listings_pool_live  ON listings(brand_id, face_value, is_pool, status)
    WHERE status = 'LIVE' AND is_pool = TRUE;
CREATE INDEX idx_listings_locked_exp ON listings(lock_expires_at)
    WHERE status = 'LOCKED';
