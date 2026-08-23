-- Down migration for 0001_init — drop in reverse dependency order
DROP TRIGGER IF EXISTS trg_payments_updated_at ON payments;
DROP TRIGGER IF EXISTS trg_bookings_updated_at ON bookings;
DROP TRIGGER IF EXISTS trg_customers_updated_at ON customers;
DROP TRIGGER IF EXISTS trg_staff_updated_at ON staff;
DROP TRIGGER IF EXISTS trg_services_updated_at ON services;
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS customers CASCADE;
DROP TABLE IF EXISTS availability_overrides CASCADE;
DROP TABLE IF EXISTS availability CASCADE;
DROP TABLE IF EXISTS staff_services CASCADE;
DROP TABLE IF EXISTS staff CASCADE;
DROP TABLE IF EXISTS services CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;

-- Keep extensions (shared) — don't drop btree_gist/pgcrypto to avoid breaking other schemas
-- DROP EXTENSION IF EXISTS "btree_gist";
-- DROP EXTENSION IF EXISTS "pgcrypto";
