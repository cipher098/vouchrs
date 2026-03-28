CREATE TABLE verification_logs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id    UUID NOT NULL REFERENCES listings(id),
    gate          SMALLINT NOT NULL CHECK (gate IN (1, 2)),
    result        VARCHAR(10) NOT NULL CHECK (result IN ('pass', 'fail')),
    balance_found NUMERIC(12,2) NOT NULL DEFAULT 0,
    fail_reason   TEXT NOT NULL DEFAULT '',
    response_hash VARCHAR(64) NOT NULL DEFAULT '',
    checked_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fraud_flags (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id),
    listing_id  UUID REFERENCES listings(id),
    reason      TEXT NOT NULL,
    severity    VARCHAR(10) NOT NULL DEFAULT 'medium',
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_verify_logs_listing ON verification_logs(listing_id, gate);
CREATE INDEX idx_fraud_flags_user    ON fraud_flags(user_id);
CREATE INDEX idx_fraud_flags_open    ON fraud_flags(is_resolved) WHERE is_resolved = FALSE;
