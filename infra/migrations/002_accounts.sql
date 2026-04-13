CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_code VARCHAR(100) UNIQUE NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPZ NOT NULL DEFAULT now()
);
ALTER TABLE accounts ADD CONSTRAINT check_positive_balance CHECK (balance >= 0);
CREATE INDEX idx_accounts_code ON accounts(account_code);