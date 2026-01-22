-- =============================================================================
-- VENDORPLATFORM - SERVICES DATABASE SCHEMA
-- Migration: 003_services_schema.sql
-- Services: Authentication, Payment, Notification, Worker
-- =============================================================================

-- -----------------------------------------------------------------------------
-- AUTHENTICATION TABLES
-- -----------------------------------------------------------------------------

-- Sessions table for JWT refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    device_info VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT sessions_expires_at_future CHECK (expires_at > created_at)
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);

-- Add authentication columns to users if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'password_hash') THEN
        ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'email_verified') THEN
        ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'phone_verified') THEN
        ALTER TABLE users ADD COLUMN phone_verified BOOLEAN DEFAULT FALSE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'last_login_at') THEN
        ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;
    END IF;
END $$;

-- -----------------------------------------------------------------------------
-- PAYMENT TABLES
-- -----------------------------------------------------------------------------

-- Wallets for internal balance tracking
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance BIGINT DEFAULT 0 CHECK (balance >= 0),
    pending_balance BIGINT DEFAULT 0 CHECK (pending_balance >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, currency)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference VARCHAR(50) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    vendor_id UUID REFERENCES vendors(id),
    booking_id UUID REFERENCES bookings(id),
    
    type VARCHAR(20) NOT NULL, -- 'payment', 'payout', 'refund', 'escrow_hold', 'escrow_release'
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    provider VARCHAR(20) NOT NULL, -- 'paystack', 'flutterwave', 'stripe', 'internal'
    
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    fee BIGINT DEFAULT 0,
    net_amount BIGINT DEFAULT 0,
    
    description TEXT,
    metadata JSONB DEFAULT '{}',
    
    provider_ref VARCHAR(100),
    provider_data JSONB DEFAULT '{}',
    
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_vendor_id ON transactions(vendor_id);
CREATE INDEX idx_transactions_booking_id ON transactions(booking_id);
CREATE INDEX idx_transactions_reference ON transactions(reference);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);

-- Escrow accounts for holding payments
CREATE TABLE IF NOT EXISTS escrow_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    booking_id UUID NOT NULL REFERENCES bookings(id),
    customer_id UUID NOT NULL REFERENCES users(id),
    vendor_id UUID NOT NULL REFERENCES users(id),
    
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    
    status VARCHAR(20) NOT NULL DEFAULT 'held', -- 'held', 'released', 'disputed', 'refunded', 'expired'
    release_condition TEXT,
    
    released_at TIMESTAMPTZ,
    dispute_id UUID,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_escrow_booking_id ON escrow_accounts(booking_id);
CREATE INDEX idx_escrow_status ON escrow_accounts(status);
CREATE INDEX idx_escrow_expires_at ON escrow_accounts(expires_at);

-- Bank accounts for payouts
CREATE TABLE IF NOT EXISTS bank_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_name VARCHAR(100) NOT NULL,
    bank_code VARCHAR(10) NOT NULL,
    account_number VARCHAR(20) NOT NULL,
    account_name VARCHAR(100) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bank_accounts_user_id ON bank_accounts(user_id);

-- -----------------------------------------------------------------------------
-- NOTIFICATION TABLES
-- -----------------------------------------------------------------------------

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL, -- 'push', 'email', 'sms', 'in_app'
    
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    priority VARCHAR(20) DEFAULT 'normal',
    
    read_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, channel) WHERE read_at IS NULL;

-- Device tokens for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL,
    platform VARCHAR(20) NOT NULL, -- 'ios', 'android', 'web'
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX idx_device_tokens_token ON device_tokens(token);

-- Notification preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    disabled_types TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- -----------------------------------------------------------------------------
-- WORKER/JOBS TABLES
-- -----------------------------------------------------------------------------

-- Jobs queue table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL,
    payload JSONB DEFAULT '{}',
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority INT DEFAULT 0,
    
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    last_error TEXT,
    
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at);
CREATE INDEX idx_jobs_pending ON jobs(status, scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_jobs_priority ON jobs(priority DESC, scheduled_at ASC) WHERE status = 'pending';

-- -----------------------------------------------------------------------------
-- EMERGENCY SERVICE TABLES (HomeRescue)
-- -----------------------------------------------------------------------------

-- Emergency technicians
CREATE TABLE IF NOT EXISTS emergency_technicians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id),
    user_id UUID NOT NULL REFERENCES users(id),
    
    name VARCHAR(100) NOT NULL,
    photo VARCHAR(500),
    phone VARCHAR(20) NOT NULL,
    
    categories TEXT[] NOT NULL,
    certifications JSONB DEFAULT '[]',
    equipment_list TEXT[] DEFAULT '{}',
    
    is_online BOOLEAN DEFAULT FALSE,
    current_status VARCHAR(20) DEFAULT 'offline',
    current_location GEOGRAPHY(POINT, 4326),
    last_location_update TIMESTAMPTZ,
    
    service_radius_km DECIMAL(5,2) DEFAULT 10.0,
    home_base GEOGRAPHY(POINT, 4326),
    
    rating DECIMAL(3,2) DEFAULT 0,
    completed_jobs INT DEFAULT 0,
    acceptance_rate DECIMAL(5,4) DEFAULT 0,
    avg_response_time_minutes INT DEFAULT 0,
    avg_arrival_time_minutes INT DEFAULT 0,
    on_time_rate DECIMAL(5,4) DEFAULT 0,
    
    active_request_id UUID,
    
    is_verified BOOLEAN DEFAULT FALSE,
    background_checked BOOLEAN DEFAULT FALSE,
    insurance_verified BOOLEAN DEFAULT FALSE,
    
    working_hours JSONB DEFAULT '[]',
    on_call_schedule JSONB DEFAULT '[]',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_emergency_techs_vendor ON emergency_technicians(vendor_id);
CREATE INDEX idx_emergency_techs_status ON emergency_technicians(current_status) WHERE is_online = TRUE;
CREATE INDEX idx_emergency_techs_location ON emergency_technicians USING GIST(current_location);
CREATE INDEX idx_emergency_techs_categories ON emergency_technicians USING GIN(categories);

-- Emergency requests
CREATE TABLE IF NOT EXISTS emergency_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    property_id UUID,
    
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(50),
    urgency VARCHAR(20) NOT NULL, -- 'critical', 'urgent', 'same_day', 'scheduled'
    
    title VARCHAR(255),
    description TEXT NOT NULL,
    photos JSONB DEFAULT '[]',
    voice_note JSONB,
    
    location JSONB NOT NULL,
    access_instructions TEXT,
    
    status VARCHAR(30) NOT NULL DEFAULT 'new',
    status_history JSONB DEFAULT '[]',
    
    assigned_vendor_id UUID REFERENCES vendors(id),
    assigned_tech_id UUID REFERENCES emergency_technicians(id),
    assignment_history JSONB DEFAULT '[]',
    
    response_deadline TIMESTAMPTZ,
    arrival_deadline TIMESTAMPTZ,
    actual_response_time TIMESTAMPTZ,
    actual_arrival_time TIMESTAMPTZ,
    
    diagnosis_notes TEXT,
    work_performed TEXT,
    parts_used JSONB DEFAULT '[]',
    work_photos JSONB DEFAULT '[]',
    
    estimated_cost JSONB,
    final_cost JSONB,
    payment_status VARCHAR(20) DEFAULT 'pending',
    
    requires_follow_up BOOLEAN DEFAULT FALSE,
    follow_up_request_id UUID,
    follow_up_notes TEXT,
    
    rating INT CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_emergency_requests_user ON emergency_requests(user_id);
CREATE INDEX idx_emergency_requests_tech ON emergency_requests(assigned_tech_id);
CREATE INDEX idx_emergency_requests_status ON emergency_requests(status);
CREATE INDEX idx_emergency_requests_urgency ON emergency_requests(urgency);
CREATE INDEX idx_emergency_requests_category ON emergency_requests(category);
CREATE INDEX idx_emergency_requests_created ON emergency_requests(created_at DESC);

-- -----------------------------------------------------------------------------
-- CONVERSATION/CHAT TABLES (EventGPT)
-- -----------------------------------------------------------------------------

-- Conversations
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id UUID REFERENCES life_events(id),
    
    session_type VARCHAR(30) NOT NULL DEFAULT 'general',
    current_intent VARCHAR(50),
    conversation_state VARCHAR(30) DEFAULT 'welcome',
    
    slot_values JSONB DEFAULT '{}',
    short_term_memory JSONB DEFAULT '[]',
    
    channel VARCHAR(20) DEFAULT 'web',
    
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_user ON conversations(user_id);
CREATE INDEX idx_conversations_event ON conversations(event_id);
CREATE INDEX idx_conversations_updated ON conversations(updated_at DESC);

-- Messages
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    
    role VARCHAR(20) NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    
    attachments JSONB DEFAULT '[]',
    quick_replies JSONB DEFAULT '[]',
    cards JSONB DEFAULT '[]',
    actions JSONB DEFAULT '[]',
    
    intent VARCHAR(50),
    entities JSONB DEFAULT '{}',
    confidence DECIMAL(4,3),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_created ON messages(created_at);

-- -----------------------------------------------------------------------------
-- LIFE EVENTS TABLE (LifeOS)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS life_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    event_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    description TEXT,
    
    detection_method VARCHAR(30), -- 'explicit', 'behavioral', 'calendar', 'social', 'transactional'
    detection_confidence DECIMAL(4,3),
    detection_signals JSONB DEFAULT '[]',
    
    status VARCHAR(30) DEFAULT 'detected',
    phase VARCHAR(30) DEFAULT 'discovery',
    
    event_date DATE,
    slot_values JSONB DEFAULT '{}',
    
    required_services JSONB DEFAULT '[]',
    budget JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_life_events_user ON life_events(user_id);
CREATE INDEX idx_life_events_type ON life_events(event_type);
CREATE INDEX idx_life_events_status ON life_events(status);
CREATE INDEX idx_life_events_date ON life_events(event_date);

-- -----------------------------------------------------------------------------
-- PARTNERSHIP TABLES (VendorNet)
-- -----------------------------------------------------------------------------

-- Vendor connections (follows/connections)
CREATE TABLE IF NOT EXISTS vendor_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    target_vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    
    connection_type VARCHAR(30) DEFAULT 'peer', -- 'peer', 'complementary', 'mentor', 'subcontractor'
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'accepted', 'declined', 'blocked'
    
    mutual_categories TEXT[] DEFAULT '{}',
    interaction_count INT DEFAULT 0,
    last_interaction_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(source_vendor_id, target_vendor_id)
);

CREATE INDEX idx_vendor_connections_source ON vendor_connections(source_vendor_id);
CREATE INDEX idx_vendor_connections_target ON vendor_connections(target_vendor_id);
CREATE INDEX idx_vendor_connections_status ON vendor_connections(status);

-- Partnerships (formal agreements)
CREATE TABLE IF NOT EXISTS partnerships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_a_id UUID NOT NULL REFERENCES vendors(id),
    vendor_b_id UUID NOT NULL REFERENCES vendors(id),
    
    partnership_type VARCHAR(30) NOT NULL, -- 'referral', 'preferred', 'exclusive', 'joint_venture', 'white_label'
    status VARCHAR(20) DEFAULT 'proposed',
    
    terms JSONB DEFAULT '{}',
    performance JSONB DEFAULT '{}',
    
    signed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_partnerships_vendor_a ON partnerships(vendor_a_id);
CREATE INDEX idx_partnerships_vendor_b ON partnerships(vendor_b_id);
CREATE INDEX idx_partnerships_status ON partnerships(status);

-- Referrals
CREATE TABLE IF NOT EXISTS referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_vendor_id UUID NOT NULL REFERENCES vendors(id),
    dest_vendor_id UUID NOT NULL REFERENCES vendors(id),
    
    client_name VARCHAR(100),
    client_email VARCHAR(255),
    client_phone VARCHAR(20),
    
    event_type VARCHAR(50),
    event_date DATE,
    estimated_value BIGINT,
    
    status VARCHAR(30) DEFAULT 'pending', -- 'pending', 'accepted', 'contacted', 'quoted', 'converted', 'lost'
    status_history JSONB DEFAULT '[]',
    
    fee_type VARCHAR(20), -- 'percentage', 'fixed', 'none'
    fee_value DECIMAL(10,2),
    fee_paid BOOLEAN DEFAULT FALSE,
    
    tracking_code VARCHAR(50) UNIQUE,
    notes TEXT,
    feedback TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    converted_at TIMESTAMPTZ
);

CREATE INDEX idx_referrals_source ON referrals(source_vendor_id);
CREATE INDEX idx_referrals_dest ON referrals(dest_vendor_id);
CREATE INDEX idx_referrals_status ON referrals(status);
CREATE INDEX idx_referrals_tracking ON referrals(tracking_code);

-- -----------------------------------------------------------------------------
-- SUBSCRIPTIONS TABLE
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    plan_id VARCHAR(50) NOT NULL,
    tier VARCHAR(30) NOT NULL, -- 'free', 'basic', 'premium', 'enterprise'
    
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'paused', 'cancelled', 'expired'
    
    price BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'NGN',
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    
    started_at TIMESTAMPTZ DEFAULT NOW(),
    current_period_start TIMESTAMPTZ DEFAULT NOW(),
    current_period_end TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    payment_method JSONB,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_tier ON subscriptions(tier);

-- -----------------------------------------------------------------------------
-- AUDIT LOG
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    
    old_data JSONB,
    new_data JSONB,
    
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);

-- Convert to hypertable for time-series
SELECT create_hypertable('audit_logs', 'created_at', if_not_exists => TRUE);

-- -----------------------------------------------------------------------------
-- FUNCTIONS AND TRIGGERS
-- -----------------------------------------------------------------------------

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to relevant tables
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'wallets', 'transactions', 'device_tokens', 'notification_preferences',
        'emergency_technicians', 'emergency_requests', 'conversations',
        'life_events', 'vendor_connections', 'partnerships', 'referrals', 'subscriptions'
    ]
    LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
            CREATE TRIGGER update_%s_updated_at
            BEFORE UPDATE ON %s
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
        ', t, t, t, t);
    END LOOP;
END $$;

-- -----------------------------------------------------------------------------
-- GRANTS (for application user)
-- -----------------------------------------------------------------------------

-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vendorplatform_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vendorplatform_app;
