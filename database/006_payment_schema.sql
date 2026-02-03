-- =============================================================================
-- PAYMENT SYSTEM SCHEMA
-- Multi-provider payment processing, escrow, and wallet management
-- =============================================================================

-- Transactions table - stores all payment transactions
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    reference VARCHAR(100) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    vendor_id UUID REFERENCES users(id),
    booking_id UUID REFERENCES bookings(id),

    type VARCHAR(50) NOT NULL CHECK (type IN ('payment', 'payout', 'refund', 'escrow_hold', 'escrow_release', 'subscription')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'processing', 'success', 'failed', 'refunded', 'cancelled', 'held')),
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('paystack', 'flutterwave', 'stripe', 'internal')),

    amount BIGINT NOT NULL,  -- In kobo/cents
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    fee BIGINT DEFAULT 0,
    net_amount BIGINT NOT NULL,

    description TEXT,
    metadata JSONB,

    provider_ref VARCHAR(255),
    provider_data JSONB,

    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_vendor_id ON transactions(vendor_id);
CREATE INDEX idx_transactions_booking_id ON transactions(booking_id);
CREATE INDEX idx_transactions_reference ON transactions(reference);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);

-- Wallets table - internal wallet for users and vendors
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    balance BIGINT NOT NULL DEFAULT 0,  -- In kobo/cents
    pending_balance BIGINT NOT NULL DEFAULT 0,  -- Held in escrow
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, currency)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_wallets_currency ON wallets(currency);

-- Escrow accounts - holds funds until service delivery
CREATE TABLE IF NOT EXISTS escrow_accounts (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    booking_id UUID NOT NULL REFERENCES bookings(id),
    customer_id UUID NOT NULL REFERENCES users(id),
    vendor_id UUID NOT NULL REFERENCES users(id),

    amount BIGINT NOT NULL,  -- In kobo/cents
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',

    status VARCHAR(50) NOT NULL CHECK (status IN ('held', 'released', 'disputed', 'refunded', 'expired')),
    release_condition VARCHAR(255),

    released_at TIMESTAMP,
    dispute_id UUID,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_escrow_transaction_id ON escrow_accounts(transaction_id);
CREATE INDEX idx_escrow_booking_id ON escrow_accounts(booking_id);
CREATE INDEX idx_escrow_customer_id ON escrow_accounts(customer_id);
CREATE INDEX idx_escrow_vendor_id ON escrow_accounts(vendor_id);
CREATE INDEX idx_escrow_status ON escrow_accounts(status);
CREATE INDEX idx_escrow_expires_at ON escrow_accounts(expires_at);

-- Payment methods - stored payment methods for users
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),

    type VARCHAR(50) NOT NULL CHECK (type IN ('card', 'bank_account', 'mobile_money')),
    provider VARCHAR(50) NOT NULL,

    -- Card details (last 4 digits only)
    card_last4 VARCHAR(4),
    card_brand VARCHAR(50),
    card_expiry_month INTEGER,
    card_expiry_year INTEGER,

    -- Bank account details
    bank_code VARCHAR(50),
    account_number VARCHAR(50),
    account_name VARCHAR(255),

    -- Mobile money details
    phone_number VARCHAR(20),

    provider_ref VARCHAR(255),  -- Authorization code from payment provider
    provider_data JSONB,

    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX idx_payment_methods_provider_ref ON payment_methods(provider_ref);

-- Payouts - vendor withdrawal requests
CREATE TABLE IF NOT EXISTS payouts (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    vendor_id UUID NOT NULL REFERENCES users(id),

    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',

    bank_code VARCHAR(50) NOT NULL,
    account_number VARCHAR(50) NOT NULL,
    account_name VARCHAR(255) NOT NULL,

    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),

    initiated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    failed_reason TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payouts_vendor_id ON payouts(vendor_id);
CREATE INDEX idx_payouts_transaction_id ON payouts(transaction_id);
CREATE INDEX idx_payouts_status ON payouts(status);

-- Webhook events - track processed webhooks to prevent duplicates
CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    reference VARCHAR(255),
    payload JSONB NOT NULL,
    signature VARCHAR(512),
    status VARCHAR(50) NOT NULL CHECK (status IN ('received', 'processing', 'processed', 'failed')),
    error_message TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_events_reference ON webhook_events(reference);
CREATE INDEX idx_webhook_events_provider ON webhook_events(provider);
CREATE INDEX idx_webhook_events_created_at ON webhook_events(created_at DESC);

-- Create updated_at trigger function if not exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add updated_at triggers
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wallets_updated_at BEFORE UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payment_methods_updated_at BEFORE UPDATE ON payment_methods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE transactions IS 'All payment transactions in the system';
COMMENT ON TABLE wallets IS 'User and vendor internal wallets for balance tracking';
COMMENT ON TABLE escrow_accounts IS 'Escrow accounts for holding funds until service delivery';
COMMENT ON TABLE payment_methods IS 'Stored payment methods for quick checkout';
COMMENT ON TABLE payouts IS 'Vendor payout/withdrawal requests';
COMMENT ON TABLE webhook_events IS 'Payment provider webhook event log';
