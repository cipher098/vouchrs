ALTER TABLE listings
    DROP COLUMN IF EXISTS expiry_date,
    DROP COLUMN IF EXISTS pin_encrypted;

ALTER TABLE brands
    DROP COLUMN IF EXISTS requires_pin;
