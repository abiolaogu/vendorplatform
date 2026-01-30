-- =============================================================================
-- LIFEOS EVENT ORCHESTRATION SCHEMA
-- Migration 006: Life Events, Phases, and Orchestration
-- =============================================================================

-- Life Events table - stores detected and confirmed life events
CREATE TABLE IF NOT EXISTS life_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Event Classification
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'wedding', 'funeral', 'birthday', 'relocation', 'renovation',
        'childbirth', 'travel', 'business_launch', 'graduation', 'retirement'
    )),
    event_subtype VARCHAR(100),
    cluster_type VARCHAR(50) NOT NULL CHECK (cluster_type IN (
        'celebrations', 'home', 'travel', 'health', 'business', 'education'
    )),

    -- Timing
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_date TIMESTAMPTZ,
    event_date_flexibility VARCHAR(20) DEFAULT 'flexible' CHECK (
        event_date_flexibility IN ('fixed', 'flexible', 'open')
    ),
    planning_horizon_days INT DEFAULT 0,

    -- Detection
    detection_method VARCHAR(50) NOT NULL CHECK (detection_method IN (
        'explicit', 'behavioral', 'calendar', 'social', 'transactional', 'partner'
    )),
    detection_confidence DECIMAL(4,3) CHECK (detection_confidence >= 0 AND detection_confidence <= 1),
    detection_signals JSONB DEFAULT '[]'::jsonb,

    -- Event Details
    scale VARCHAR(20) DEFAULT 'medium' CHECK (scale IN (
        'intimate', 'small', 'medium', 'large', 'massive'
    )),
    guest_count INT CHECK (guest_count > 0),
    location JSONB,
    budget JSONB,

    -- Orchestration State
    status VARCHAR(20) NOT NULL DEFAULT 'detected' CHECK (status IN (
        'detected', 'confirmed', 'planning', 'booked', 'in_progress', 'completed', 'cancelled'
    )),
    phase VARCHAR(30) NOT NULL DEFAULT 'discovery' CHECK (phase IN (
        'discovery', 'planning', 'vendor_select', 'booking', 'pre_event', 'event_day', 'post_event'
    )),
    completion_percentage DECIMAL(5,2) DEFAULT 0 CHECK (
        completion_percentage >= 0 AND completion_percentage <= 100
    ),

    -- User Preferences
    preferences JSONB DEFAULT '{}'::jsonb,
    constraints JSONB DEFAULT '[]'::jsonb,

    -- Metadata
    custom_attributes JSONB DEFAULT '{}'::jsonb,
    tags TEXT[],

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Indexes
    CONSTRAINT valid_completion_dates CHECK (
        (status = 'completed' AND completed_at IS NOT NULL) OR
        (status != 'completed')
    )
);

-- Indexes for life_events
CREATE INDEX idx_life_events_user_id ON life_events(user_id);
CREATE INDEX idx_life_events_status ON life_events(status);
CREATE INDEX idx_life_events_event_type ON life_events(event_type);
CREATE INDEX idx_life_events_event_date ON life_events(event_date) WHERE event_date IS NOT NULL;
CREATE INDEX idx_life_events_detection_method ON life_events(detection_method);
CREATE INDEX idx_life_events_created_at ON life_events(created_at DESC);
CREATE INDEX idx_life_events_detected_at ON life_events(detected_at DESC);

-- GIN indexes for JSONB columns
CREATE INDEX idx_life_events_detection_signals ON life_events USING GIN (detection_signals);
CREATE INDEX idx_life_events_preferences ON life_events USING GIN (preferences);
CREATE INDEX idx_life_events_custom_attrs ON life_events USING GIN (custom_attributes);

-- Event Services - tracks required and booked services for an event
CREATE TABLE IF NOT EXISTS event_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES service_categories(id),

    -- Service Requirements
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    is_required BOOLEAN DEFAULT true,

    -- Timing
    phase VARCHAR(30) NOT NULL CHECK (phase IN (
        'discovery', 'planning', 'vendor_select', 'booking', 'pre_event', 'event_day', 'post_event'
    )),
    deadline_days_before_event INT DEFAULT 0,
    book_by_date TIMESTAMPTZ,

    -- Budget
    budget_allocation_percentage DECIMAL(5,2) DEFAULT 0,
    estimated_cost_min DECIMAL(12,2),
    estimated_cost_max DECIMAL(12,2),
    currency VARCHAR(3) DEFAULT 'NGN',

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'searching', 'shortlisted', 'booked', 'skipped'
    )),
    booking_id UUID REFERENCES bookings(id),

    -- Vendor Recommendations
    recommended_vendors JSONB DEFAULT '[]'::jsonb,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(event_id, category_id)
);

-- Indexes for event_services
CREATE INDEX idx_event_services_event_id ON event_services(event_id);
CREATE INDEX idx_event_services_status ON event_services(status);
CREATE INDEX idx_event_services_priority ON event_services(priority);
CREATE INDEX idx_event_services_booking_id ON event_services(booking_id) WHERE booking_id IS NOT NULL;

-- Event Phases - tracks phase-specific tasks and milestones
CREATE TABLE IF NOT EXISTS event_phases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,

    phase VARCHAR(30) NOT NULL CHECK (phase IN (
        'discovery', 'planning', 'vendor_select', 'booking', 'pre_event', 'event_day', 'post_event'
    )),

    -- Timeline
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'active', 'completed', 'blocked'
    )),

    -- Tasks
    tasks JSONB DEFAULT '[]'::jsonb,

    -- Dependencies
    dependencies UUID[],

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_phase_dates CHECK (start_date < end_date),
    UNIQUE(event_id, phase)
);

-- Indexes for event_phases
CREATE INDEX idx_event_phases_event_id ON event_phases(event_id);
CREATE INDEX idx_event_phases_status ON event_phases(status);
CREATE INDEX idx_event_phases_dates ON event_phases(start_date, end_date);

-- Event Milestones - critical checkpoints
CREATE TABLE IF NOT EXISTS event_milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,

    title VARCHAR(200) NOT NULL,
    milestone_date TIMESTAMPTZ NOT NULL,

    -- Related entities
    service_id UUID REFERENCES event_services(id),
    phase VARCHAR(30),

    -- Status
    is_met BOOLEAN DEFAULT false,
    met_at TIMESTAMPTZ,
    blocks_event BOOLEAN DEFAULT false,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for event_milestones
CREATE INDEX idx_event_milestones_event_id ON event_milestones(event_id);
CREATE INDEX idx_event_milestones_date ON event_milestones(milestone_date);
CREATE INDEX idx_event_milestones_is_met ON event_milestones(is_met);

-- Event Bundles - tracks bundle opportunities for events
CREATE TABLE IF NOT EXISTS event_bundle_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,
    bundle_id UUID NOT NULL REFERENCES service_bundles(id),

    -- Pricing
    total_price DECIMAL(12,2) NOT NULL,
    regular_price DECIMAL(12,2) NOT NULL,
    savings DECIMAL(12,2) NOT NULL,
    savings_percentage DECIMAL(5,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'NGN',

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'offered' CHECK (status IN (
        'offered', 'viewed', 'saved', 'accepted', 'rejected', 'expired'
    )),

    -- Included Services
    included_service_categories UUID[],

    -- Metadata
    offered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    viewed_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for event_bundle_offers
CREATE INDEX idx_event_bundle_offers_event_id ON event_bundle_offers(event_id);
CREATE INDEX idx_event_bundle_offers_bundle_id ON event_bundle_offers(bundle_id);
CREATE INDEX idx_event_bundle_offers_status ON event_bundle_offers(status);

-- Event Risks - identified risks and mitigation strategies
CREATE TABLE IF NOT EXISTS event_risks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,

    -- Risk Details
    risk_type VARCHAR(50) NOT NULL CHECK (risk_type IN (
        'timeline', 'budget', 'availability', 'quality', 'coordination', 'weather', 'compliance'
    )),
    description TEXT NOT NULL,

    -- Assessment
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    likelihood VARCHAR(20) NOT NULL CHECK (likelihood IN ('unlikely', 'possible', 'likely', 'certain')),

    -- Impact
    affected_services UUID[],
    estimated_impact_cost DECIMAL(12,2),

    -- Mitigation
    mitigation_steps TEXT[],
    mitigation_status VARCHAR(20) DEFAULT 'pending' CHECK (mitigation_status IN (
        'pending', 'in_progress', 'mitigated', 'accepted'
    )),

    -- Metadata
    identified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for event_risks
CREATE INDEX idx_event_risks_event_id ON event_risks(event_id);
CREATE INDEX idx_event_risks_severity ON event_risks(severity);
CREATE INDEX idx_event_risks_status ON event_risks(mitigation_status);

-- Event Actions - recommended next actions for users
CREATE TABLE IF NOT EXISTS event_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES life_events(id) ON DELETE CASCADE,

    -- Action Details
    title VARCHAR(200) NOT NULL,
    description TEXT,
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN (
        'book', 'confirm', 'review', 'pay', 'contact', 'research', 'decide'
    )),

    -- Priority & Timing
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    due_date TIMESTAMPTZ,

    -- Related Entities
    related_service_id UUID REFERENCES event_services(id),
    related_category_id UUID REFERENCES service_categories(id),
    deep_link TEXT,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'in_progress', 'completed', 'dismissed'
    )),

    -- Metadata
    completed_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for event_actions
CREATE INDEX idx_event_actions_event_id ON event_actions(event_id);
CREATE INDEX idx_event_actions_status ON event_actions(status);
CREATE INDEX idx_event_actions_priority ON event_actions(priority);
CREATE INDEX idx_event_actions_due_date ON event_actions(due_date) WHERE due_date IS NOT NULL;

-- Update trigger for life_events
CREATE OR REPLACE FUNCTION update_life_event_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER life_events_updated_at
    BEFORE UPDATE ON life_events
    FOR EACH ROW
    EXECUTE FUNCTION update_life_event_timestamp();

-- Views for analytics

-- Active events summary
CREATE OR REPLACE VIEW v_active_events_summary AS
SELECT
    le.id,
    le.user_id,
    u.full_name as user_name,
    u.email,
    le.event_type,
    le.event_date,
    le.status,
    le.phase,
    le.completion_percentage,
    le.scale,
    le.guest_count,
    (le.budget->>'total_amount')::numeric as budget_amount,
    COUNT(DISTINCT es.id) as total_services,
    COUNT(DISTINCT es.id) FILTER (WHERE es.status = 'booked') as booked_services,
    COUNT(DISTINCT er.id) FILTER (WHERE er.severity IN ('high', 'critical')) as critical_risks,
    le.created_at,
    le.detected_at,
    EXTRACT(EPOCH FROM (le.event_date - NOW()))/86400 as days_until_event
FROM life_events le
JOIN users u ON u.id = le.user_id
LEFT JOIN event_services es ON es.event_id = le.id
LEFT JOIN event_risks er ON er.event_id = le.id AND er.mitigation_status != 'mitigated'
WHERE le.status NOT IN ('completed', 'cancelled')
GROUP BY le.id, u.full_name, u.email;

-- Event completion analytics
CREATE OR REPLACE VIEW v_event_completion_metrics AS
SELECT
    event_type,
    COUNT(*) as total_events,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_events,
    ROUND(COUNT(*) FILTER (WHERE status = 'completed')::numeric / COUNT(*)::numeric * 100, 2) as completion_rate,
    AVG(completion_percentage) FILTER (WHERE status NOT IN ('cancelled')) as avg_completion_pct,
    AVG((budget->>'spent')::numeric) FILTER (WHERE status = 'completed') as avg_spent,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at))/86400) FILTER (WHERE status = 'completed') as avg_days_to_complete
FROM life_events
GROUP BY event_type;

-- Comments
COMMENT ON TABLE life_events IS 'Detected and confirmed life events requiring service orchestration';
COMMENT ON TABLE event_services IS 'Required services for each life event with recommendations and status';
COMMENT ON TABLE event_phases IS 'Timeline phases for event orchestration with tasks and dependencies';
COMMENT ON TABLE event_milestones IS 'Critical checkpoints in event timeline';
COMMENT ON TABLE event_bundle_offers IS 'Bundle opportunities offered for events';
COMMENT ON TABLE event_risks IS 'Identified risks for events with mitigation strategies';
COMMENT ON TABLE event_actions IS 'Recommended next actions for users to progress their events';

COMMENT ON VIEW v_active_events_summary IS 'Real-time summary of active events with key metrics';
COMMENT ON VIEW v_event_completion_metrics IS 'Event completion analytics by event type';
