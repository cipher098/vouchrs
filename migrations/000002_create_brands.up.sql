CREATE TABLE brands (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                VARCHAR(100) NOT NULL,
    slug                VARCHAR(100) NOT NULL UNIQUE,
    logo_url            TEXT NOT NULL DEFAULT '',
    verification_source VARCHAR(50) NOT NULL DEFAULT 'qwikcilver',
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed Amazon India as the first active brand
INSERT INTO brands (name, slug, verification_source, status)
VALUES ('Amazon India', 'amazon', 'qwikcilver', 'active');
