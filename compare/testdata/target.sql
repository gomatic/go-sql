-- Target SQL file for comparison testing
-- Contains modifications from source

-- Schemas (unchanged)
CREATE SCHEMA IF NOT EXISTS app_schema;
CREATE SCHEMA IF NOT EXISTS audit_schema;

-- Tables (users table modified, orders unchanged)
CREATE TABLE app_schema.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP  -- NEW: added column
);

CREATE TABLE app_schema.orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES app_schema.users(id),
    total_amount DECIMAL(10, 2),
    status VARCHAR(50) DEFAULT 'pending'
);

-- NEW: products table added
CREATE TABLE app_schema.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2)
);

-- Views (changed query)
CREATE VIEW app_schema.active_users AS
    SELECT id, username, email, created_at FROM app_schema.users WHERE created_at > NOW() - INTERVAL '7 days';

-- Indexes (one unchanged, one removed, one added)
CREATE INDEX idx_users_email ON app_schema.users(email);
-- idx_orders_user removed
CREATE INDEX idx_products_name ON app_schema.products(name);  -- NEW

-- Functions (unchanged signature, body changes ignored)
CREATE FUNCTION app_schema.get_user_count() RETURNS INTEGER AS $$
BEGIN
    -- Modified body but same signature
    RETURN (SELECT COUNT(*) FROM app_schema.users WHERE created_at IS NOT NULL);
END;
$$ LANGUAGE plpgsql;

-- Triggers (unchanged)
CREATE TRIGGER audit_user_changes
    AFTER INSERT OR UPDATE ON app_schema.users
    FOR EACH ROW EXECUTE FUNCTION audit_schema.log_change();

-- Grants (changed grantee)
GRANT SELECT ON app_schema.users TO public_role;  -- Changed from readonly_role
GRANT ALL ON app_schema.orders TO admin_role;

-- Comments (changed)
COMMENT ON TABLE app_schema.users IS 'User accounts with profile data';  -- Changed
COMMENT ON COLUMN app_schema.users.email IS 'Unique email address';

-- Alter statements (different changes)
ALTER TABLE app_schema.users ADD COLUMN verified BOOLEAN DEFAULT FALSE;  -- Different column
ALTER TABLE app_schema.orders DROP COLUMN IF EXISTS old_field;
