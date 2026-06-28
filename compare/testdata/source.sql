-- Source SQL file for comparison testing
-- Contains a variety of DDL statements

-- Schemas
CREATE SCHEMA IF NOT EXISTS app_schema;
CREATE SCHEMA IF NOT EXISTS audit_schema;

-- Tables
CREATE TABLE app_schema.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE app_schema.orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES app_schema.users(id),
    total_amount DECIMAL(10, 2),
    status VARCHAR(50) DEFAULT 'pending'
);

-- Views
CREATE VIEW app_schema.active_users AS
    SELECT id, username, email FROM app_schema.users WHERE created_at > NOW() - INTERVAL '30 days';

-- Indexes
CREATE INDEX idx_users_email ON app_schema.users(email);
CREATE INDEX idx_orders_user ON app_schema.orders(user_id);

-- Functions
CREATE FUNCTION app_schema.get_user_count() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM app_schema.users);
END;
$$ LANGUAGE plpgsql;

-- Triggers
CREATE TRIGGER audit_user_changes
    AFTER INSERT OR UPDATE ON app_schema.users
    FOR EACH ROW EXECUTE FUNCTION audit_schema.log_change();

-- Grants
GRANT SELECT ON app_schema.users TO readonly_role;
GRANT ALL ON app_schema.orders TO admin_role;

-- Comments
COMMENT ON TABLE app_schema.users IS 'User accounts table';
COMMENT ON COLUMN app_schema.users.email IS 'Unique email address';

-- Alter statements
ALTER TABLE app_schema.users ADD COLUMN last_login TIMESTAMP;
ALTER TABLE app_schema.orders DROP COLUMN IF EXISTS old_field;
