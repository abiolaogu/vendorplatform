-- =============================================================================
-- LIFEOS - LIFE EVENT ORCHESTRATION SCHEMA
-- Database schema for life event detection, tracking, and orchestration
-- =============================================================================

-- Life Events table
CREATE TABLE IF NOT EXISTS life_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,

    -- Event Classification
    event_type VARCHAR(50) NOT NULL,
    event_subtype VARCHAR(100),
    cluster_type VARCHAR(50) NOT NULL,

    -- Timing
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    event_date TIMESTAMP,
    event_date_flexibility VARCHAR(20) DEFAULT 'flexible',
    planning_horizon_days INTEGER DEFAULT 90,

    -- Detection
    detection_method VARCHAR(50) NOT NULL DEFAULT 'explicit',
    detection_confidence DECIMAL(3,2) DEFAULT 1.0,

    -- Event Details
    scale VARCHAR(20) DEFAULT 'medium',
    guest_count INTEGER,

    -- Orchestration State
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
    phase VARCHAR(30) NOT NULL DEFAULT 'discovery',
    completion_percentage DECIMAL(5,2) DEFAULT 0.0,

    -- Metadata
    custom_attributes JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_life_events_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    -- Constraints
    CHECK (detection_confidence >= 0 AND detection_confidence <= 1),
    CHECK (completion_percentage >= 0 AND completion_percentage <= 100),
    CHECK (event_type IN ('wedding', 'funeral', 'birthday', 'relocation', 'renovation',
                          'childbirth', 'travel', 'business_launch', 'graduation', 'retirement')),
    CHECK (cluster_type IN ('celebrations', 'home', 'travel', 'health', 'business', 'education')),
    CHECK (status IN ('detected', 'confirmed', 'planning', 'booked', 'in_progress', 'completed', 'cancelled')),
    CHECK (phase IN ('discovery', 'planning', 'vendor_select', 'booking', 'pre_event', 'event_day', 'post_event'))
);

-- Detection Signals table (stores evidence for event detection)
CREATE TABLE IF NOT EXISTS life_event_detection_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    signal_type VARCHAR(50) NOT NULL,
    source VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_detection_signals_event FOREIGN KEY (event_id) REFERENCES life_events(id) ON DELETE CASCADE,
    CHECK (confidence >= 0 AND confidence <= 1)
);

-- Event Service Requirements table
CREATE TABLE IF NOT EXISTS life_event_service_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    category_id UUID NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'optional',
    booking_status VARCHAR(20) DEFAULT 'pending',
    booked_service_id UUID,
    budget_allocation_pct DECIMAL(5,2),
    booking_deadline DATE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_service_reqs_event FOREIGN KEY (event_id) REFERENCES life_events(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_reqs_category FOREIGN KEY (category_id) REFERENCES service_categories(id),
    CONSTRAINT fk_service_reqs_service FOREIGN KEY (booked_service_id) REFERENCES services(id),
    CHECK (priority IN ('primary', 'secondary', 'optional')),
    CHECK (booking_status IN ('pending', 'researching', 'quoted', 'booked', 'cancelled', 'completed'))
);

-- Event Budget Tracking
CREATE TABLE IF NOT EXISTS life_event_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    total_budget DECIMAL(15,2),
    allocated_amount DECIMAL(15,2) DEFAULT 0,
    spent_amount DECIMAL(15,2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'NGN',
    budget_flexibility VARCHAR(20) DEFAULT 'moderate',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_event_budgets_event FOREIGN KEY (event_id) REFERENCES life_events(id) ON DELETE CASCADE,
    CHECK (budget_flexibility IN ('strict', 'moderate', 'flexible')),
    UNIQUE(event_id)
);

-- Event Timeline/Milestones
CREATE TABLE IF NOT EXISTS life_event_milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    milestone_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    due_date DATE NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    category_id UUID,
    priority INTEGER DEFAULT 5,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_milestones_event FOREIGN KEY (event_id) REFERENCES life_events(id) ON DELETE CASCADE,
    CONSTRAINT fk_milestones_category FOREIGN KEY (category_id) REFERENCES service_categories(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_life_events_user_id ON life_events(user_id);
CREATE INDEX IF NOT EXISTS idx_life_events_status ON life_events(status);
CREATE INDEX IF NOT EXISTS idx_life_events_event_type ON life_events(event_type);
CREATE INDEX IF NOT EXISTS idx_life_events_event_date ON life_events(event_date);
CREATE INDEX IF NOT EXISTS idx_life_events_created_at ON life_events(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_detection_signals_event_id ON life_event_detection_signals(event_id);
CREATE INDEX IF NOT EXISTS idx_detection_signals_timestamp ON life_event_detection_signals(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_service_reqs_event_id ON life_event_service_requirements(event_id);
CREATE INDEX IF NOT EXISTS idx_service_reqs_category_id ON life_event_service_requirements(category_id);
CREATE INDEX IF NOT EXISTS idx_service_reqs_status ON life_event_service_requirements(booking_status);

CREATE INDEX IF NOT EXISTS idx_event_budgets_event_id ON life_event_budgets(event_id);

CREATE INDEX IF NOT EXISTS idx_milestones_event_id ON life_event_milestones(event_id);
CREATE INDEX IF NOT EXISTS idx_milestones_due_date ON life_event_milestones(due_date);
CREATE INDEX IF NOT EXISTS idx_milestones_is_completed ON life_event_milestones(is_completed);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_life_event_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_life_events_updated_at
    BEFORE UPDATE ON life_events
    FOR EACH ROW
    EXECUTE FUNCTION update_life_event_updated_at();

CREATE TRIGGER trigger_service_reqs_updated_at
    BEFORE UPDATE ON life_event_service_requirements
    FOR EACH ROW
    EXECUTE FUNCTION update_life_event_updated_at();

CREATE TRIGGER trigger_event_budgets_updated_at
    BEFORE UPDATE ON life_event_budgets
    FOR EACH ROW
    EXECUTE FUNCTION update_life_event_updated_at();

-- Comments for documentation
COMMENT ON TABLE life_events IS 'Stores detected and declared life events for orchestration';
COMMENT ON TABLE life_event_detection_signals IS 'Evidence and signals used for event detection';
COMMENT ON TABLE life_event_service_requirements IS 'Service categories required for each life event';
COMMENT ON TABLE life_event_budgets IS 'Budget tracking and allocation for life events';
COMMENT ON TABLE life_event_milestones IS 'Timeline and milestone tracking for life events';

COMMENT ON COLUMN life_events.detection_confidence IS 'Confidence score (0-1) for detected events';
COMMENT ON COLUMN life_events.completion_percentage IS 'Percentage of required services booked (0-100)';
COMMENT ON COLUMN life_events.planning_horizon_days IS 'Days until event date or default planning period';
