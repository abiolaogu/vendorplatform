-- =============================================================================
-- HOMERESCUE - EMERGENCY HOME SERVICES SCHEMA
-- Version: 1.0.0
-- =============================================================================

-- Emergencies table - stores all emergency service requests
CREATE TABLE IF NOT EXISTS emergencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Emergency classification
    category VARCHAR(50) NOT NULL CHECK (category IN (
        'plumbing', 'electrical', 'locksmith', 'hvac',
        'glass', 'roofing', 'pest', 'security', 'general'
    )),
    subcategory VARCHAR(100),
    urgency VARCHAR(20) NOT NULL CHECK (urgency IN (
        'critical', 'urgent', 'same_day', 'scheduled'
    )),

    -- Description
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    photos JSONB DEFAULT '[]'::jsonb,
    voice_note_url TEXT,

    -- Location
    address TEXT NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    access_instructions TEXT,

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'new' CHECK (status IN (
        'new', 'searching', 'assigned', 'accepted', 'en_route',
        'arrived', 'diagnosing', 'quoted', 'approved', 'in_progress',
        'completed', 'cancelled', 'no_show', 'disputed'
    )),
    status_history JSONB DEFAULT '[]'::jsonb,

    -- Assignment
    assigned_vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    assigned_tech_id UUID REFERENCES users(id) ON DELETE SET NULL,
    assignment_history JSONB DEFAULT '[]'::jsonb,

    -- Response tracking
    response_deadline TIMESTAMPTZ,
    arrival_deadline TIMESTAMPTZ,
    actual_response_time TIMESTAMPTZ,
    actual_arrival_time TIMESTAMPTZ,

    -- Technician GPS tracking
    tech_latitude DECIMAL(10, 8),
    tech_longitude DECIMAL(11, 8),
    estimated_arrival TIMESTAMPTZ,

    -- Work details
    diagnosis_notes TEXT,
    work_performed TEXT,
    parts_used JSONB DEFAULT '[]'::jsonb,
    work_photos JSONB DEFAULT '[]'::jsonb,

    -- Pricing
    estimated_cost DECIMAL(10, 2),
    final_cost DECIMAL(10, 2),
    payment_status VARCHAR(20) DEFAULT 'pending' CHECK (payment_status IN (
        'pending', 'held', 'charged', 'refunded', 'disputed'
    )),

    -- Follow-up
    requires_follow_up BOOLEAN DEFAULT FALSE,
    follow_up_request_id UUID REFERENCES emergencies(id),
    follow_up_notes TEXT,

    -- Customer satisfaction
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    review TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Indexes
    CONSTRAINT valid_coords CHECK (
        latitude >= -90 AND latitude <= 90 AND
        longitude >= -180 AND longitude <= 180
    )
);

-- Indexes for performance
CREATE INDEX idx_emergencies_user_id ON emergencies(user_id);
CREATE INDEX idx_emergencies_status ON emergencies(status);
CREATE INDEX idx_emergencies_category ON emergencies(category);
CREATE INDEX idx_emergencies_urgency ON emergencies(urgency);
CREATE INDEX idx_emergencies_assigned_vendor ON emergencies(assigned_vendor_id) WHERE assigned_vendor_id IS NOT NULL;
CREATE INDEX idx_emergencies_assigned_tech ON emergencies(assigned_tech_id) WHERE assigned_tech_id IS NOT NULL;
CREATE INDEX idx_emergencies_location ON emergencies USING gist (
    ll_to_earth(latitude::float8, longitude::float8)
);
CREATE INDEX idx_emergencies_created_at ON emergencies(created_at DESC);
CREATE INDEX idx_emergencies_active ON emergencies(status) WHERE status NOT IN ('completed', 'cancelled');

-- Technician availability table - tracks which technicians are available for emergencies
CREATE TABLE IF NOT EXISTS technician_availability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    technician_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,

    -- Categories they can handle
    categories VARCHAR(50)[] NOT NULL,

    -- Current status
    is_available BOOLEAN DEFAULT TRUE,
    current_latitude DECIMAL(10, 8),
    current_longitude DECIMAL(11, 8),
    last_location_update TIMESTAMPTZ,

    -- Work capacity
    max_concurrent_jobs INTEGER DEFAULT 1,
    current_job_count INTEGER DEFAULT 0,

    -- Availability schedule
    availability_schedule JSONB, -- Weekly schedule

    -- Performance metrics
    avg_response_time_minutes INTEGER,
    avg_rating DECIMAL(3, 2),
    completed_emergencies INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(technician_id)
);

CREATE INDEX idx_tech_availability_vendor ON technician_availability(vendor_id);
CREATE INDEX idx_tech_availability_status ON technician_availability(is_available) WHERE is_available = TRUE;
CREATE INDEX idx_tech_availability_categories ON technician_availability USING gin(categories);
CREATE INDEX idx_tech_availability_location ON technician_availability USING gist (
    ll_to_earth(current_latitude::float8, current_longitude::float8)
) WHERE is_available = TRUE;

-- Emergency notifications table - tracks notifications sent to technicians
CREATE TABLE IF NOT EXISTS emergency_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    emergency_id UUID NOT NULL REFERENCES emergencies(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Notification details
    notification_type VARCHAR(50) NOT NULL, -- 'new_request', 'reminder', 'escalation'
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ,
    responded_at TIMESTAMPTZ,
    response VARCHAR(20), -- 'accepted', 'declined', 'timeout'

    -- Distance at time of notification
    distance_km DECIMAL(8, 2),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emergency_notif_emergency ON emergency_notifications(emergency_id);
CREATE INDEX idx_emergency_notif_tech ON emergency_notifications(technician_id);
CREATE INDEX idx_emergency_notif_sent ON emergency_notifications(sent_at DESC);

-- Emergency response SLAs (for analytics and monitoring)
CREATE TABLE IF NOT EXISTS emergency_sla_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    emergency_id UUID NOT NULL REFERENCES emergencies(id) ON DELETE CASCADE UNIQUE,

    -- SLA targets (in minutes)
    target_response_time INTEGER NOT NULL,
    target_arrival_time INTEGER NOT NULL,

    -- Actual times (in minutes)
    actual_response_time INTEGER,
    actual_arrival_time INTEGER,

    -- SLA status
    response_sla_met BOOLEAN,
    arrival_sla_met BOOLEAN,

    -- Calculated at
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emergency_sla_metrics ON emergency_sla_metrics(emergency_id);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for auto-updating updated_at
CREATE TRIGGER update_emergencies_updated_at
    BEFORE UPDATE ON emergencies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_technician_availability_updated_at
    BEFORE UPDATE ON technician_availability
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to log status changes in status_history
CREATE OR REPLACE FUNCTION log_emergency_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        NEW.status_history = COALESCE(NEW.status_history, '[]'::jsonb) || jsonb_build_object(
            'from', OLD.status,
            'to', NEW.status,
            'changed_at', NOW(),
            'changed_by', current_user
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER log_emergency_status_changes
    BEFORE UPDATE ON emergencies
    FOR EACH ROW
    EXECUTE FUNCTION log_emergency_status_change();

-- Comments for documentation
COMMENT ON TABLE emergencies IS 'Emergency home service requests with real-time tracking';
COMMENT ON TABLE technician_availability IS 'Real-time availability and location of emergency technicians';
COMMENT ON TABLE emergency_notifications IS 'Log of notifications sent to technicians for emergency requests';
COMMENT ON TABLE emergency_sla_metrics IS 'SLA tracking and compliance metrics for emergency responses';

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE ON emergencies TO app_user;
-- GRANT SELECT, INSERT, UPDATE ON technician_availability TO app_user;
-- GRANT SELECT, INSERT ON emergency_notifications TO app_user;
-- GRANT SELECT, INSERT, UPDATE ON emergency_sla_metrics TO app_user;
