-- =============================================================================
-- BOOKINGS SCHEMA
-- =============================================================================
-- This schema supports the booking lifecycle from creation to completion,
-- including payment tracking, ratings, and cancellations.

-- Bookings table
CREATE TABLE IF NOT EXISTS bookings (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_code VARCHAR(50) NOT NULL UNIQUE,

    -- Relationships
    user_id UUID NOT NULL,
    vendor_id UUID NOT NULL,
    service_id UUID NOT NULL,
    project_id UUID,  -- Optional: if booking is part of a larger project

    -- Service details
    service_name VARCHAR(255) NOT NULL,
    service_description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, confirmed, in_progress, completed, cancelled, refunded

    -- Schedule
    scheduled_date DATE,
    scheduled_time VARCHAR(20),
    duration_minutes INTEGER,
    completed_at TIMESTAMPTZ,

    -- Location
    service_location TEXT NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),

    -- Pricing
    base_price DECIMAL(12, 2) NOT NULL,
    tax_amount DECIMAL(12, 2) NOT NULL DEFAULT 0,
    service_fee DECIMAL(12, 2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(12, 2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',

    -- Payment
    payment_status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, paid, failed, refunded
    payment_method VARCHAR(50),
    transaction_ref VARCHAR(255),

    -- Customer information
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,

    -- Additional details
    notes TEXT,
    requirements JSONB,
    cancellation_reason TEXT,

    -- Ratings and reviews
    customer_rating DECIMAL(2, 1) CHECK (customer_rating >= 1 AND customer_rating <= 5),
    customer_review TEXT,
    vendor_rating DECIMAL(2, 1) CHECK (vendor_rating >= 1 AND vendor_rating <= 5),
    vendor_review TEXT,

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for bookings
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_vendor_id ON bookings(vendor_id);
CREATE INDEX IF NOT EXISTS idx_bookings_service_id ON bookings(service_id);
CREATE INDEX IF NOT EXISTS idx_bookings_project_id ON bookings(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_payment_status ON bookings(payment_status);
CREATE INDEX IF NOT EXISTS idx_bookings_scheduled_date ON bookings(scheduled_date);
CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings(created_at);
CREATE INDEX IF NOT EXISTS idx_bookings_booking_code ON bookings(booking_code);

-- GiST index for location-based queries
CREATE INDEX IF NOT EXISTS idx_bookings_location ON bookings USING GIST (
    ll_to_earth(latitude, longitude)
) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- =============================================================================
-- BOOKING TIMELINE TRACKING
-- =============================================================================
-- Track all status changes and important events in a booking's lifecycle

CREATE TABLE IF NOT EXISTS booking_timeline (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- status_change, payment_update, rating_added, message_sent, etc.
    event_data JSONB,
    actor_id UUID, -- Who performed this action
    actor_type VARCHAR(50), -- user, vendor, system, admin
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_booking_timeline_booking_id ON booking_timeline(booking_id);
CREATE INDEX IF NOT EXISTS idx_booking_timeline_created_at ON booking_timeline(created_at);

-- =============================================================================
-- BOOKING MESSAGES
-- =============================================================================
-- Communication between customer and vendor for a specific booking

CREATE TABLE IF NOT EXISTS booking_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    sender_type VARCHAR(50) NOT NULL, -- customer, vendor
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_booking_messages_booking_id ON booking_messages(booking_id);
CREATE INDEX IF NOT EXISTS idx_booking_messages_sender_id ON booking_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_booking_messages_created_at ON booking_messages(created_at);

-- =============================================================================
-- FUNCTIONS AND TRIGGERS
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_booking_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for bookings updated_at
CREATE TRIGGER bookings_updated_at_trigger
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_booking_updated_at();

-- Function to automatically create timeline entry on booking status change
CREATE OR REPLACE FUNCTION create_booking_timeline_on_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO booking_timeline (booking_id, event_type, event_data, actor_type)
        VALUES (
            NEW.id,
            'status_change',
            jsonb_build_object(
                'old_status', OLD.status,
                'new_status', NEW.status
            ),
            'system'
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic timeline creation
CREATE TRIGGER booking_status_change_timeline_trigger
    AFTER UPDATE OF status ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION create_booking_timeline_on_status_change();

-- =============================================================================
-- VIEWS FOR REPORTING
-- =============================================================================

-- Booking summary by vendor (for vendor dashboards)
CREATE OR REPLACE VIEW vendor_booking_summary AS
SELECT
    vendor_id,
    COUNT(*) AS total_bookings,
    COUNT(*) FILTER (WHERE status = 'pending') AS pending_bookings,
    COUNT(*) FILTER (WHERE status = 'confirmed') AS confirmed_bookings,
    COUNT(*) FILTER (WHERE status = 'completed') AS completed_bookings,
    COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled_bookings,
    SUM(total_amount) FILTER (WHERE status = 'completed') AS total_revenue,
    AVG(customer_rating) FILTER (WHERE customer_rating IS NOT NULL) AS avg_rating,
    COUNT(*) FILTER (WHERE customer_rating IS NOT NULL) AS rating_count
FROM bookings
GROUP BY vendor_id;

-- Recent bookings requiring action
CREATE OR REPLACE VIEW bookings_requiring_action AS
SELECT
    b.id,
    b.booking_code,
    b.vendor_id,
    b.user_id,
    b.service_name,
    b.status,
    b.scheduled_date,
    b.total_amount,
    b.created_at,
    CASE
        WHEN b.status = 'pending' AND b.created_at < NOW() - INTERVAL '24 hours' THEN 'pending_confirmation_overdue'
        WHEN b.status = 'confirmed' AND b.scheduled_date < CURRENT_DATE THEN 'should_be_in_progress'
        WHEN b.status = 'in_progress' AND b.created_at < NOW() - INTERVAL '7 days' THEN 'in_progress_too_long'
        WHEN b.status = 'completed' AND b.customer_rating IS NULL THEN 'needs_rating'
        ELSE 'ok'
    END AS action_needed
FROM bookings b
WHERE
    (b.status = 'pending' AND b.created_at < NOW() - INTERVAL '24 hours')
    OR (b.status = 'confirmed' AND b.scheduled_date < CURRENT_DATE)
    OR (b.status = 'in_progress' AND b.created_at < NOW() - INTERVAL '7 days')
    OR (b.status = 'completed' AND b.customer_rating IS NULL);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE bookings IS 'Core bookings table tracking all service bookings from creation to completion';
COMMENT ON TABLE booking_timeline IS 'Audit trail of all important events in a booking lifecycle';
COMMENT ON TABLE booking_messages IS 'Messages exchanged between customer and vendor for a specific booking';
COMMENT ON VIEW vendor_booking_summary IS 'Aggregated booking metrics per vendor for dashboard display';
COMMENT ON VIEW bookings_requiring_action IS 'Bookings that require attention due to status or timing';
