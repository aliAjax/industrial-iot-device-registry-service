CREATE TABLE IF NOT EXISTS device_products (
    id text PRIMARY KEY,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
    id text PRIMARY KEY,
    product_id text NOT NULL,
    status text NOT NULL,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_devices_product_status ON devices (product_id, status);

CREATE TABLE IF NOT EXISTS device_groups (
    id text PRIMARY KEY,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_principals (
    id text PRIMARY KEY,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_rates (
    rate_key text PRIMARY KEY,
    window_start timestamptz NOT NULL,
    count integer NOT NULL
);

CREATE TABLE IF NOT EXISTS commands (
    id text PRIMARY KEY,
    device_id text NOT NULL,
    idempotency_key text NOT NULL,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_commands_idempotency ON commands (device_id, idempotency_key);

CREATE TABLE IF NOT EXISTS firmware (
    id text PRIMARY KEY,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS firmware_tasks (
    id text PRIMARY KEY,
    firmware_id text NOT NULL,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS firmware_receipts (
    id text PRIMARY KEY,
    task_id text NOT NULL,
    device_id text NOT NULL,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_firmware_receipts_device ON firmware_receipts (device_id);

CREATE TABLE IF NOT EXISTS twin_documents (
    id text PRIMARY KEY,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS rules (
    id text PRIMARY KEY,
    device_id text NOT NULL,
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS rule_executions (
    id text PRIMARY KEY,
    rule_id text NOT NULL,
    device_id text NOT NULL,
    data jsonb NOT NULL,
    executed_at timestamptz NOT NULL
);
