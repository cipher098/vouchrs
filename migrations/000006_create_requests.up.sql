CREATE TABLE buy_requests (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id),
    brand_id      UUID NOT NULL REFERENCES brands(id),
    min_value     NUMERIC(12,2) NOT NULL,
    max_value     NUMERIC(12,2) NOT NULL,
    max_price     NUMERIC(12,2) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    alerted_count INTEGER NOT NULL DEFAULT 0,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE card_requests (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id),
    brand         VARCHAR(100) NOT NULL,
    desired_value NUMERIC(12,2) NOT NULL,
    urgency       VARCHAR(30) NOT NULL DEFAULT 'flexible',
    status        VARCHAR(30) NOT NULL DEFAULT 'PENDING_ADMIN_REVIEW',
    admin_notes   TEXT NOT NULL DEFAULT '',
    fulfilled_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_buy_requests_user    ON buy_requests(user_id);
CREATE INDEX idx_buy_requests_brand   ON buy_requests(brand_id, status) WHERE status = 'active';
CREATE INDEX idx_card_requests_user   ON card_requests(user_id);
CREATE INDEX idx_card_requests_status ON card_requests(status);
