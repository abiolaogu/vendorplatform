-- =============================================================================
-- VENDORNET B2B PARTNERSHIP NETWORK SCHEMA
-- =============================================================================
-- This schema supports vendor-to-vendor partnerships, referrals, and
-- collaborative opportunities.

-- =============================================================================
-- PARTNERSHIPS
-- =============================================================================

CREATE TABLE IF NOT EXISTS partnerships (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Partners
    vendor_a_id UUID NOT NULL,
    vendor_b_id UUID NOT NULL,

    -- Partnership Details
    type VARCHAR(50) NOT NULL DEFAULT 'referral', -- referral, preferred, exclusive, joint_venture, white_label
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'proposed', -- proposed, negotiating, active, paused, expired, terminated

    -- Terms
    terms TEXT, -- JSON or plain text describing terms

    -- Performance Tracking
    total_referrals INTEGER NOT NULL DEFAULT 0,
    successful_referrals INTEGER NOT NULL DEFAULT 0,
    total_revenue DECIMAL(12, 2) NOT NULL DEFAULT 0,

    -- Agreement
    signed_by_a BOOLEAN NOT NULL DEFAULT FALSE,
    signed_by_b BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT partnerships_vendor_check CHECK (vendor_a_id != vendor_b_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_partnerships_vendor_a ON partnerships(vendor_a_id);
CREATE INDEX IF NOT EXISTS idx_partnerships_vendor_b ON partnerships(vendor_b_id);
CREATE INDEX IF NOT EXISTS idx_partnerships_status ON partnerships(status);
CREATE INDEX IF NOT EXISTS idx_partnerships_type ON partnerships(type);

-- =============================================================================
-- REFERRALS
-- =============================================================================

CREATE TABLE IF NOT EXISTS referrals (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source and Destination
    source_vendor_id UUID NOT NULL,
    dest_vendor_id UUID NOT NULL,
    partnership_id UUID REFERENCES partnerships(id) ON DELETE SET NULL,

    -- Client Information
    client_user_id UUID, -- If registered user
    client_name VARCHAR(255) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    client_phone VARCHAR(50) NOT NULL,

    -- Referral Context
    event_type VARCHAR(100) NOT NULL,
    event_date DATE,
    service_category_id UUID NOT NULL,
    estimated_value DECIMAL(12, 2) NOT NULL DEFAULT 0,
    notes TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, accepted, declined, contacted, quoted, converted, lost, expired

    -- Outcome
    converted_booking_id UUID, -- Reference to booking if converted
    actual_value DECIMAL(12, 2) NOT NULL DEFAULT 0,

    -- Fee Structure
    fee_type VARCHAR(50) NOT NULL DEFAULT 'percentage', -- percentage, fixed, none
    fee_value DECIMAL(12, 2) NOT NULL DEFAULT 10.0,
    calculated_fee DECIMAL(12, 2) NOT NULL DEFAULT 0,
    fee_paid BOOLEAN NOT NULL DEFAULT FALSE,
    fee_paid_at TIMESTAMPTZ,

    -- Tracking
    tracking_code VARCHAR(50) NOT NULL UNIQUE,
    source_url VARCHAR(500),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_referrals_source_vendor ON referrals(source_vendor_id);
CREATE INDEX IF NOT EXISTS idx_referrals_dest_vendor ON referrals(dest_vendor_id);
CREATE INDEX IF NOT EXISTS idx_referrals_partnership ON referrals(partnership_id) WHERE partnership_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_referrals_status ON referrals(status);
CREATE INDEX IF NOT EXISTS idx_referrals_tracking_code ON referrals(tracking_code);
CREATE INDEX IF NOT EXISTS idx_referrals_created_at ON referrals(created_at);

-- =============================================================================
-- CONNECTIONS (Vendor-to-Vendor Network)
-- =============================================================================

CREATE TABLE IF NOT EXISTS vendor_connections (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Parties
    vendor_a_id UUID NOT NULL,
    vendor_b_id UUID NOT NULL,

    -- Connection Type
    connection_type VARCHAR(50) NOT NULL DEFAULT 'peer', -- peer, complementary, mentor, subcontractor
    relationship_note TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, accepted, declined, blocked
    initiated_by UUID NOT NULL,

    -- Activity
    last_interaction_at TIMESTAMPTZ,
    interaction_count INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT vendor_connections_unique UNIQUE (vendor_a_id, vendor_b_id),
    CONSTRAINT vendor_connections_check CHECK (vendor_a_id < vendor_b_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_vendor_connections_vendor_a ON vendor_connections(vendor_a_id);
CREATE INDEX IF NOT EXISTS idx_vendor_connections_vendor_b ON vendor_connections(vendor_b_id);
CREATE INDEX IF NOT EXISTS idx_vendor_connections_status ON vendor_connections(status);

-- =============================================================================
-- OPPORTUNITIES (Collaborative Bidding)
-- =============================================================================

CREATE TABLE IF NOT EXISTS opportunities (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Client
    client_user_id UUID,
    client_name VARCHAR(255) NOT NULL,
    client_type VARCHAR(50) NOT NULL DEFAULT 'individual', -- individual, corporate, agency

    -- Event Details
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_date DATE,
    event_location TEXT,
    guest_count INTEGER,

    -- Requirements
    required_category_ids UUID[] NOT NULL DEFAULT '{}',
    requirements TEXT[],

    -- Budget
    budget_min DECIMAL(12, 2),
    budget_max DECIMAL(12, 2),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, closed, awarded, cancelled
    visibility VARCHAR(50) NOT NULL DEFAULT 'network', -- public, network, invited

    -- Bidding
    bid_deadline TIMESTAMPTZ NOT NULL,
    bid_count INTEGER NOT NULL DEFAULT 0,
    selected_bid_id UUID,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities(status);
CREATE INDEX IF NOT EXISTS idx_opportunities_event_type ON opportunities(event_type);
CREATE INDEX IF NOT EXISTS idx_opportunities_bid_deadline ON opportunities(bid_deadline);
CREATE INDEX IF NOT EXISTS idx_opportunities_created_at ON opportunities(created_at);

-- =============================================================================
-- COLLABORATIVE BIDS
-- =============================================================================

CREATE TABLE IF NOT EXISTS collaborative_bids (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Opportunity
    opportunity_id UUID NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,

    -- Lead Vendor
    lead_vendor_id UUID NOT NULL,

    -- Bid Details
    total_bid_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    proposal_doc TEXT,

    -- Team (JSON array of team members)
    team_members JSONB NOT NULL DEFAULT '[]',

    -- Revenue Split (JSON array of splits)
    split_agreement JSONB NOT NULL DEFAULT '[]',

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, pending, submitted, under_review, won, lost, withdrawn

    -- Outcome
    won_bid BOOLEAN NOT NULL DEFAULT FALSE,
    won_at TIMESTAMPTZ,
    contract_id UUID,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    deadline_at TIMESTAMPTZ NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_collaborative_bids_opportunity ON collaborative_bids(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_collaborative_bids_lead_vendor ON collaborative_bids(lead_vendor_id);
CREATE INDEX IF NOT EXISTS idx_collaborative_bids_status ON collaborative_bids(status);

-- =============================================================================
-- FUNCTIONS AND TRIGGERS
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_vendornet_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER partnerships_updated_at_trigger
    BEFORE UPDATE ON partnerships
    FOR EACH ROW
    EXECUTE FUNCTION update_vendornet_updated_at();

CREATE TRIGGER referrals_updated_at_trigger
    BEFORE UPDATE ON referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_vendornet_updated_at();

CREATE TRIGGER opportunities_updated_at_trigger
    BEFORE UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_vendornet_updated_at();

-- Function to update partnership stats when referral status changes
CREATE OR REPLACE FUNCTION update_partnership_stats_on_referral()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.partnership_id IS NOT NULL AND OLD.status IS DISTINCT FROM NEW.status THEN
        -- Update total referrals
        IF OLD.status = 'pending' AND NEW.status != 'pending' THEN
            UPDATE partnerships
            SET total_referrals = total_referrals + 1
            WHERE id = NEW.partnership_id;
        END IF;

        -- Update successful referrals and revenue
        IF NEW.status = 'converted' THEN
            UPDATE partnerships
            SET successful_referrals = successful_referrals + 1,
                total_revenue = total_revenue + NEW.actual_value
            WHERE id = NEW.partnership_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for partnership stats
CREATE TRIGGER referral_partnership_stats_trigger
    AFTER UPDATE OF status ON referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_partnership_stats_on_referral();

-- =============================================================================
-- VIEWS FOR REPORTING
-- =============================================================================

-- Partnership performance summary
CREATE OR REPLACE VIEW partnership_performance AS
SELECT
    p.id,
    p.vendor_a_id,
    p.vendor_b_id,
    p.name,
    p.status,
    p.total_referrals,
    p.successful_referrals,
    CASE
        WHEN p.total_referrals > 0
        THEN (p.successful_referrals::float / p.total_referrals * 100)
        ELSE 0
    END as conversion_rate,
    p.total_revenue,
    CASE
        WHEN p.successful_referrals > 0
        THEN (p.total_revenue / p.successful_referrals)
        ELSE 0
    END as avg_referral_value,
    p.created_at,
    p.activated_at
FROM partnerships p;

-- Vendor network stats
CREATE OR REPLACE VIEW vendor_network_stats AS
SELECT
    v.id as vendor_id,
    v.business_name,
    COUNT(DISTINCT CASE WHEN vc.vendor_a_id = v.id THEN vc.vendor_b_id
                        WHEN vc.vendor_b_id = v.id THEN vc.vendor_a_id END) as total_connections,
    COUNT(DISTINCT CASE WHEN p.vendor_a_id = v.id OR p.vendor_b_id = v.id THEN p.id END) as active_partnerships,
    COUNT(DISTINCT CASE WHEN r.source_vendor_id = v.id THEN r.id END) as referrals_sent,
    COUNT(DISTINCT CASE WHEN r.dest_vendor_id = v.id THEN r.id END) as referrals_received,
    COUNT(DISTINCT CASE WHEN r.dest_vendor_id = v.id AND r.status = 'converted' THEN r.id END) as referrals_converted,
    COALESCE(SUM(CASE WHEN r.dest_vendor_id = v.id AND r.status = 'converted' THEN r.actual_value ELSE 0 END), 0) as total_referral_revenue
FROM vendors v
LEFT JOIN vendor_connections vc ON (vc.vendor_a_id = v.id OR vc.vendor_b_id = v.id) AND vc.status = 'accepted'
LEFT JOIN partnerships p ON (p.vendor_a_id = v.id OR p.vendor_b_id = v.id) AND p.status = 'active'
LEFT JOIN referrals r ON r.source_vendor_id = v.id OR r.dest_vendor_id = v.id
GROUP BY v.id, v.business_name;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE partnerships IS 'Formal business partnerships between vendors';
COMMENT ON TABLE referrals IS 'Client referrals between vendors with fee tracking';
COMMENT ON TABLE vendor_connections IS 'Vendor-to-vendor professional network connections';
COMMENT ON TABLE opportunities IS 'Large projects available for collaborative bidding';
COMMENT ON TABLE collaborative_bids IS 'Joint bids from multiple vendors on opportunities';
COMMENT ON VIEW partnership_performance IS 'Performance metrics for each partnership';
COMMENT ON VIEW vendor_network_stats IS 'Aggregated network statistics per vendor';
