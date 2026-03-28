CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone               VARCHAR(20) UNIQUE,
    email               VARCHAR(255) UNIQUE,
    full_name           VARCHAR(255) NOT NULL DEFAULT '',
    role                VARCHAR(20) NOT NULL DEFAULT 'buyer',
    is_verified         BOOLEAN NOT NULL DEFAULT FALSE,
    is_banned           BOOLEAN NOT NULL DEFAULT FALSE,
    is_flagged          BOOLEAN NOT NULL DEFAULT FALSE,
    listing_count_today INTEGER NOT NULL DEFAULT 0,
    listing_count_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    upi_id              VARCHAR(255) NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role  ON users(role);
