CREATE TABLE transactions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id       UUID NOT NULL REFERENCES listings(id),
    buyer_id         UUID NOT NULL REFERENCES users(id),
    seller_id        UUID NOT NULL REFERENCES users(id),
    buyer_amount     NUMERIC(12,2) NOT NULL,
    seller_payout    NUMERIC(12,2) NOT NULL,
    payment_ref      VARCHAR(100) NOT NULL DEFAULT '',
    payout_ref       VARCHAR(100) NOT NULL DEFAULT '',
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    lock_started_at  TIMESTAMPTZ,
    paid_at          TIMESTAMPTZ,
    code_revealed_at TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_txn_buyer_id    ON transactions(buyer_id);
CREATE INDEX idx_txn_seller_id   ON transactions(seller_id);
CREATE INDEX idx_txn_listing_id  ON transactions(listing_id);
CREATE INDEX idx_txn_payment_ref ON transactions(payment_ref) WHERE payment_ref != '';
CREATE INDEX idx_txn_status      ON transactions(status);
