CREATE TABLE pool_groups (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    brand_id          UUID NOT NULL REFERENCES brands(id),
    face_value        NUMERIC(12,2) NOT NULL,
    recommended_price NUMERIC(12,2) NOT NULL,
    buyer_price       NUMERIC(12,2) NOT NULL,
    discount_pct      NUMERIC(5,2) NOT NULL,
    active_count      INTEGER NOT NULL DEFAULT 0,
    avg_sell_time_mins NUMERIC(8,2) NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(brand_id, face_value)
);

CREATE INDEX idx_pool_groups_active ON pool_groups(brand_id, face_value)
    WHERE active_count > 0;
